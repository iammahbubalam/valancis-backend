package utils

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SecretKey should be passed from config, not global
// But to keep signature simple for this refactor without breaking unrelated calls:
// We will set it via a Setup function or pass it in.
// A cleaner way for "Production": Pass config to Usecase, Usecase calls TokenGenerator.
// For now, let's export a Setup function.

var secretKey []byte

func SetSecret(key string) {
	secretKey = []byte(key)
}

func GenerateJWT(userID, email, role string, expiry time.Duration) (string, error) {
	if len(secretKey) == 0 {
		return "", fmt.Errorf("jwt secret not set")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(expiry).Unix(),
	})

	return token.SignedString(secretKey)
}

func ValidateJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func GenerateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

type Claims struct {
	UserID string
	Email  string
	Role   string
}

// ExtractClaims extracts JWT claims from the request header or cookie
func ExtractClaims(r *http.Request) (*Claims, error) {
	tokenString := ""
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		tokenString = authHeader[7:]
	} else {
		cookie, err := r.Cookie("accessToken")
		if err == nil {
			tokenString = cookie.Value
		}
	}

	if tokenString == "" {
		return nil, fmt.Errorf("no token found")
	}

	mapClaims, err := ValidateJWT(tokenString)
	if err != nil {
		return nil, err
	}

	userID, _ := mapClaims["sub"].(string)
	email, _ := mapClaims["email"].(string)
	role, _ := mapClaims["role"].(string)

	return &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
	}, nil
}
