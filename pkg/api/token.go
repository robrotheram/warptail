package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey = "mysecretkey" // Replace with a secure key

// Middleware for API_TOKEN authentication
func TokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "API token required", http.StatusUnauthorized)
			return
		}

		_, err := ValidateToken(token)
		if err != nil {
			http.Error(w, "Invalid API token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Generate JWT token
func GenerateToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Validate JWT token
func ValidateToken(tokenString string) (*jwt.Token, error) {
	tokenString = strings.ReplaceAll(tokenString, "Bearer ", "")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secretKey), nil
	})
	return token, err
}

func (api *api) loginHandler(w http.ResponseWriter, r *http.Request) {
	var loginData struct {
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Example: Replace this with your actual authentication logic
	if loginData.Password == api.config.Dasboard.Token {
		token, err := GenerateToken(api.config.Dasboard.Token)
		if err != nil {
			http.Error(w, "Could not generate token", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"authorization_token": token})
	} else {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
	}
}
