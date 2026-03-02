package v1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"valancis-backend/internal/domain"
	"valancis-backend/internal/usecase"
	"strconv"
)

type AuthHandler struct {
	authUC *usecase.AuthUsecase
}

func NewAuthHandler(authUC *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUC: authUC}
}

type googleLoginReq struct {
	IDToken string `json:"idToken"`
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	slog.Info("GoogleLogin request received")
	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Updated call to AuthenticateGoogle
	accessToken, refreshToken, user, err := h.authUC.AuthenticateGoogle(r.Context(), req.Code)
	if err != nil {
		slog.Error("Authentication failed", "error", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Set Refresh Token as HttpOnly Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/", // Allows access for refresh endpoint
		HttpOnly: true,
		Secure:   true, // Should be true in production/https
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	slog.Info("User authenticated successfully", "user_id", user.ID, "email", user.Email)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accessToken": accessToken,
		"user":        user,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "Refresh token missing", http.StatusUnauthorized)
		return
	}

	newAccessToken, err := h.authUC.RefreshAccessToken(r.Context(), cookie.Value)
	if err != nil {
		slog.Error("Token refresh failed", "error", err)
		// Clear cookie if invalid
		http.SetCookie(w, &http.Cookie{Name: "refresh_token", MaxAge: -1, Path: "/"})
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accessToken": newAccessToken,
	})
}

// --- Address Handlers ---

func (h *AuthHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := user.ID

	var req domain.Address
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	addr, err := h.authUC.AddAddress(r.Context(), userID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(addr)
}

func (h *AuthHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := user.ID
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Address ID required", http.StatusBadRequest)
		return
	}

	var req domain.Address
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	req.ID = id

	addr, err := h.authUC.UpdateAddress(r.Context(), userID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(addr)
}

func (h *AuthHandler) GetAddresses(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := user.ID

	addrs, err := h.authUC.GetAddresses(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(addrs)
}

func (h *AuthHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Address ID required", http.StatusBadRequest)
		return
	}

	if err := h.authUC.DeleteAddress(r.Context(), id, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Phone     string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedUser, err := h.authUC.UpdateProfile(r.Context(), user.ID, req.FirstName, req.LastName, req.Phone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedUser)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	// Assumes AuthMiddleware has run and set User in context
	userCtx, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Delegate to Usecase to get full user profile if needed
	// Or just return the user from context if it has enough info (ID, Email, Role)
	// But let's fetch full profile to get addresses etc.
	user, err := h.authUC.GetUserByID(r.Context(), userCtx.ID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// 1. Attempt to get the refresh token to revoke it
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		// Log error but don't fail the logout process if revocation fails (client still wants to clear cookies)
		if err := h.authUC.RevokeToken(r.Context(), cookie.Value); err != nil {
			slog.Error("Failed to revoke token on logout", "error", err)
		} else {
			slog.Info("Refresh token revoked successfully")
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:   "refresh_token",
		MaxAge: -1,
		Path:   "/",
	})
	w.WriteHeader(http.StatusOK)
}

// ListUsers - Admin endpoint to get all users
// ListUsers - Admin endpoint to get all users
func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page := 1
	limit := 10

	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	offset := (page - 1) * limit
	users, count, err := h.authUC.GetAllUsers(r.Context(), limit, offset)
	if err != nil {
		slog.Error("Failed to list users", "error", err)
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	totalPages := 0
	if count > 0 {
		totalPages = int((count + int64(limit) - 1) / int64(limit))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"meta": map[string]interface{}{
			"total":      count,
			"page":       page,
			"limit":      limit,
			"totalPages": totalPages,
		},
	})
}
