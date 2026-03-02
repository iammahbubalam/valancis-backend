package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"valancis-backend/internal/domain"
	"valancis-backend/pkg/utils"
	"strings"
	"time"
)

type AuthUsecase struct {
	userRepo           domain.UserRepository
	clientID           string
	clientSecret       string
	tokenInfoURL       string
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

func NewAuthUsecase(userRepo domain.UserRepository, clientID, clientSecret, tokenInfoURL string, atExpiry, rtExpiry time.Duration) *AuthUsecase {
	return &AuthUsecase{
		userRepo:           userRepo,
		clientID:           clientID,
		clientSecret:       clientSecret,
		tokenInfoURL:       tokenInfoURL,
		accessTokenExpiry:  atExpiry,
		refreshTokenExpiry: rtExpiry,
	}
}

type GoogleUser struct {
	ID            string      `json:"sub"`
	Email         string      `json:"email"`
	EmailVerified interface{} `json:"email_verified"` // Can be string ("true") or bool (true)
	Name          string      `json:"name"`
	GivenName     string      `json:"given_name"`
	FamilyName    string      `json:"family_name"`
	Picture       string      `json:"picture"`
	Aud           string      `json:"aud"`
}

func (u *AuthUsecase) AuthenticateGoogle(ctx context.Context, code string) (string, string, *domain.User, error) {
	slog.Info("Authenticating with Google via Code Flow", "code_length", len(code))

	// 1. Exchange Code for Tokens
	googleTokens, err := u.exchangeCodeForToken(code)
	if err != nil {
		slog.Error("Google code exchange failed", "error", err)
		return "", "", nil, fmt.Errorf("google code exchange failed: %v", err)
	}

	// 2. Fetch User Info using Access Token
	userInfo, err := u.fetchGoogleUserInfo(googleTokens.AccessToken)
	if err != nil {
		slog.Error("Failed to fetch google user info", "error", err)
		return "", "", nil, fmt.Errorf("failed to fetch user info: %v", err)
	}

	// In production, verify Audience matches Client ID if returned in ID Token
	// (googleTokens.IDToken can be parsed, but UserInfo endpoint is also authoritative source for this flow)

	// 2. Find or Create User
	user, err := u.userRepo.GetByEmail(ctx, userInfo.Email)
	if err != nil {
		return "", "", nil, err
	}

	if user == nil {
		slog.Info("Creating new user", "email", userInfo.Email)
		// Create new user
		user = &domain.User{
			ID:        utils.GenerateUUID(),
			Email:     userInfo.Email,
			FirstName: userInfo.GivenName,
			LastName:  userInfo.FamilyName,
			Avatar:    userInfo.Picture,
			Role:      "customer",
		}
		if err := u.userRepo.Create(ctx, user); err != nil {
			slog.Error("Failed to create user", "error", err)
			return "", "", nil, err
		}
	} else {
		slog.Info("Existing user found", "user_id", user.ID)
		// Sync Profile Data from Google (in case it changed)
		user.FirstName = userInfo.GivenName
		user.LastName = userInfo.FamilyName
		user.Avatar = userInfo.Picture
		// Ensure Role doesn't get overwritten unless we want to logic it

		if err := u.userRepo.Update(ctx, user); err != nil {
			slog.Error("Failed to update user profile", "error", err)
			// Non-critical error, continue login
		}
	}

	// 3. Generate Access Token (JWT)
	accessToken, err := utils.GenerateJWT(user.ID, user.Email, user.Role, u.accessTokenExpiry)
	if err != nil {
		return "", "", nil, err
	}

	// 4. Generate Refresh Token (UUID) and Save
	refreshTokenStr := utils.GenerateUUID()
	refreshToken := &domain.RefreshToken{
		Token:     refreshTokenStr,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(u.refreshTokenExpiry),
		CreatedAt: time.Now(),
		Device:    "unknown", // Could be passed from handler
	}

	if err := u.userRepo.SaveRefreshToken(ctx, refreshToken); err != nil {
		return "", "", nil, err
	}

	// We return AccessToken as string, User object, and RefreshToken (needs struct change or return value change)
	// Currently signature is (string, *User, error).
	// To minimize breaking changes elsewhere immediately (though we are refactoring), let's return AccessToken.
	// BUT we need to pass RefreshToken to handler to set cookie.
	// So we should return separate object or just return refreshTokenStr as well.
	// Let's change return signature to (accessToken string, refreshToken string, user *domain.User, err error)
	// Wait, that might break interface if one exists. No interface for Usecase yet, only struct.

	// Modifying return to: (accessToken, refreshToken, user, error)
	return accessToken, refreshTokenStr, user, nil
}

func (u *AuthUsecase) RefreshAccessToken(ctx context.Context, refreshTokenStr string) (string, error) {
	// 1. Verify Refresh Token
	rt, err := u.userRepo.GetRefreshToken(ctx, refreshTokenStr)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token")
	}

	if rt.Revoked {
		return "", fmt.Errorf("token revoked")
	}
	if time.Now().After(rt.ExpiresAt) {
		return "", fmt.Errorf("token expired")
	}

	// 2. Refresh it? Or just issue new Access?
	// User said "secure access and refresh token strategy".
	// Usually invalidating old refresh and issuing new one (Rotating Refresh Tokens) is strictly more secure.
	// But "don't make it complex". So keep Refresh Token valid until expiry.

	user, err := u.userRepo.GetByID(ctx, rt.UserID)
	if err != nil {
		return "", err
	}

	newAccessToken, err := utils.GenerateJWT(user.ID, user.Email, user.Role, u.accessTokenExpiry)
	return newAccessToken, err
}

func (u *AuthUsecase) RevokeToken(ctx context.Context, refreshTokenStr string) error {
	return u.userRepo.RevokeRefreshToken(ctx, refreshTokenStr)
}

// --- Address Management ---

func (u *AuthUsecase) AddAddress(ctx context.Context, userID string, req domain.Address) (*domain.Address, error) {
	// Sanitize UserID
	req.UserID = userID
	req.ID = utils.GenerateUUID()
	req.CreatedAt = time.Now()

	if err := u.userRepo.AddAddress(ctx, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (u *AuthUsecase) UpdateAddress(ctx context.Context, userID string, req domain.Address) (*domain.Address, error) {
	req.UserID = userID
	if req.ID == "" {
		return nil, fmt.Errorf("address ID required")
	}
	// L9: Address ownership is enforced by repository layer via WHERE user_id=? AND id=? clause
	// This prevents users from updating addresses that don't belong to them

	if err := u.userRepo.UpdateAddress(ctx, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (u *AuthUsecase) GetAddresses(ctx context.Context, userID string) ([]domain.Address, error) {
	return u.userRepo.GetAddresses(ctx, userID)
}

func (u *AuthUsecase) DeleteAddress(ctx context.Context, id, userID string) error {
	return u.userRepo.DeleteAddress(ctx, id, userID)
}

func (u *AuthUsecase) UpdateProfile(ctx context.Context, userID, firstName, lastName, phone string) (*domain.User, error) {
	return u.userRepo.UpdateProfile(ctx, userID, firstName, lastName, phone)
}

type GoogleTokens struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"` // Only if access_type=offline
}

func (u *AuthUsecase) exchangeCodeForToken(code string) (*GoogleTokens, error) {
	// https://oauth2.googleapis.com/token
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", u.clientID)
	data.Set("client_secret", u.clientSecret)
	data.Set("redirect_uri", "postmessage") // Important for React Google Login 'auth-code' flow
	data.Set("grant_type", "authorization_code")

	req, _ := http.NewRequest("POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("google token endpoint returned status: %d", resp.StatusCode)
	}

	var tokens GoogleTokens
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}
	return &tokens, nil
}

func (u *AuthUsecase) fetchGoogleUserInfo(accessToken string) (*GoogleUser, error) {
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("google userinfo returned status: %d", resp.StatusCode)
	}

	var gUser GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return nil, err
	}
	return &gUser, nil
}

func (u *AuthUsecase) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	return u.userRepo.GetByID(ctx, id)
}

func (u *AuthUsecase) GetAllUsers(ctx context.Context, limit, offset int) ([]*domain.User, int64, error) {
	return u.userRepo.GetAll(ctx, limit, offset)
}
