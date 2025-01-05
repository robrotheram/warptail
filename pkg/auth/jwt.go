package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"warptail/pkg/utils"

	"github.com/gorilla/sessions"
)

const JWTAUTH = "JWTAUTH"

type JWTAuth struct {
	secretKey    string
	sessionStore *sessions.CookieStore
	users        *Users
}

func NewJWTAuthProvider(config utils.AuthenticationProvider, sessionStore *sessions.CookieStore, users *Users) (*JWTAuth, error) {
	return &JWTAuth{
		secretKey:    config.Secret,
		sessionStore: sessionStore,
		users:        users,
	}, nil
}

func (auth *JWTAuth) Login(w http.ResponseWriter, r *http.Request) {

	var loginData struct {
		Username string `json:"Username"`
		Password string `json:"password"`
	}
	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := auth.users.FindByEamil(loginData.Username, r.Context())
	if err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Example: Replace this with your actual authentication logic
	if user.VerifyPassword(loginData.Password) {
		token := GenerateToken(user.ID.String(), JWTAUTH, auth.secretKey)

		session, _ := auth.sessionStore.Get(r, "auth-session")
		session.Values["jwt"] = token
		session.Save(r, w)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"authorization_token": token,
			"role":                string(user.Role),
		})
	} else {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
	}
}

func (auth *JWTAuth) GetUser(identifier string) (User, error) {
	return auth.users.FindByID(identifier, context.Background())
}
