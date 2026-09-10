package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"warptail/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/sessions"
	"github.com/uptrace/bun"
)

type Provider interface {
	Login(w http.ResponseWriter, r *http.Request)
	IsValid(w http.ResponseWriter, r *http.Request) bool
}

type Authentication struct {
	OICDProvider *OpenIDAuth
	JWTProvider  *JWTAuth
	sessionStore *sessions.CookieStore

	users   *Users
	baseUrl string
}

func NewAuthentication(mux *chi.Mux, db *bun.DB, config utils.AuthenticationConfig, allowedRedirect ...func(*url.URL) bool) *Authentication {
	auth := Authentication{
		baseUrl: config.BaseURL,
	}

	auth.sessionStore = sessions.NewCookieStore([]byte(config.SessionSecret))
	auth.sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
		Secure:   strings.HasPrefix(config.BaseURL, "https://"),
		SameSite: http.SameSiteLaxMode,
	}
	auth.users = NewUserStore(db, config.Provider.Basic)

	if provider, err := NewOpenIdProvider(config, auth.sessionStore, auth.users); err == nil {
		provider.safeRedirect = func(value string) string {
			var allowed func(*url.URL) bool
			if len(allowedRedirect) > 0 {
				allowed = allowedRedirect[0]
			}
			return SafeRedirect(value, config.BaseURL, allowed)
		}
		auth.OICDProvider = provider
		mux.HandleFunc("/auth/callback", auth.OICDProvider.Callback)
	}

	if provider, err := NewJWTAuthProvider(config, auth.sessionStore, auth.users); err == nil {
		auth.JWTProvider = provider
	}

	mux.HandleFunc("/auth/login", auth.Login)
	mux.Post("/auth/logout", auth.Logout)
	mux.Get("/auth/profile", auth.HandleGetProfile)
	mux.Post("/auth/profile", auth.HandleUpdateProfile)
	return &auth
}

func ParseUrlFromRequest(r *http.Request) string {
	target := *r.URL
	target.Host = r.Host
	if target.Scheme == "" {
		target.Scheme = "http"
		if r.TLS != nil {
			target.Scheme = "https"
		}
	}
	query := target.Query()
	query.Del("token")
	target.RawQuery = query.Encode()
	return target.String()
}

func (auth *Authentication) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if auth.OICDProvider != nil {
			auth.OICDProvider.Login(w, r)
		} else {
			http.Error(w, "OIDC provider not configured", http.StatusNotImplemented)
		}
	} else if r.Method == http.MethodPost {
		if auth.JWTProvider != nil {
			auth.JWTProvider.Login(w, r)
		} else {
			http.Error(w, "JWT provider not configured", http.StatusNotImplemented)
		}
	} else {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (auth *Authentication) Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	session, _ := auth.sessionStore.Get(r, "auth-session")
	session.Options.MaxAge = -1 // Delete session
	if err := session.Save(r, w); err != nil {
		http.Error(w, "Unable to end session", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("You have been logged out."))
}

func (auth *Authentication) Authenticate(w http.ResponseWriter, r *http.Request, handler func(w http.ResponseWriter, r *http.Request)) {
	token := r.URL.Query().Get("token")
	if len(token) > 0 {
		session, _ := auth.sessionStore.Get(r, "auth-session")
		session.Values["jwt"] = token
		session.Save(r, w)
		http.Redirect(w, r, ParseUrlFromRequest(r), http.StatusTemporaryRedirect)
		return
	}

	_, err := auth.GetUser(w, r)
	if err != nil {
		if r.Method == "GET" {
			path, _ := url.JoinPath(auth.baseUrl, "/login")
			redirectURL := fmt.Sprintf("%s?next=%s", path, url.QueryEscape(ParseUrlFromRequest(r)))
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}
		http.Error(w, "proxy authentication required", http.StatusUnauthorized)
		return
	}
	handler(w, r)
}

func (auth *Authentication) GetUser(w http.ResponseWriter, r *http.Request) (User, error) {
	var user User
	session, _ := auth.sessionStore.Get(r, "auth-session")
	// Explicit credentials take precedence over an older browser session.
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		token, _ = session.Values["jwt"].(string)
	}

	if auth.JWTProvider == nil {
		return user, fmt.Errorf("no JWT provider configured")
	}

	// Try decoding with JWT provider secret key first
	identifier, tokenType, err := DecodeToken(token, auth.JWTProvider.secretKey)
	if err != nil && auth.OICDProvider != nil {
		// If JWT decoding fails and OIDC provider exists, try with OIDC secret key
		identifier, tokenType, err = DecodeToken(token, auth.OICDProvider.secretKey)
	}

	if err != nil {
		if r.Header.Get("Authorization") == "" {
			session.Options.MaxAge = -1
			session.Save(r, w)
		}
		return user, err
	}
	switch tokenType {
	case JWTAUTH:
		if auth.JWTProvider == nil {
			return user, fmt.Errorf("JWT provider not available")
		}
		return auth.JWTProvider.GetUser(identifier)
	case OIDCAUTH:
		if auth.OICDProvider == nil {
			return user, fmt.Errorf("OIDC provider not available")
		}
		return auth.OICDProvider.GetUser(identifier, r.Context())
	}
	return user, fmt.Errorf("no authentication providers found")
}

func (auth *Authentication) DashboardAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		user, err := auth.GetUser(w, r)
		if err != nil {
			http.Error(w, "Authentication failed. Invalid token.", http.StatusUnauthorized)
			utils.LogHttpError(r, fmt.Errorf("Authentication failed. Invalid token: %v", err))
			return
		}
		if user.Role != ADMIN || user.PasswordReset {
			http.Error(w, "Authentication failed. Permission Denied.", http.StatusForbidden)
			utils.LogHttpError(r, fmt.Errorf("Authentication failed. Permission Denied: %v", err))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (auth *Authentication) DashboardMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		user, err := auth.GetUser(w, r)
		if err != nil {
			http.Error(w, "Authentication failed. Invalid token.", http.StatusUnauthorized)
			utils.LogHttpError(r, fmt.Errorf("Authentication failed. Invalid token: %v", err))
			return
		}
		if user.PasswordReset {
			http.Error(w, "Password reset required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GenerateToken(identifier, authType, secretKey string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"identifier": identifier,
		"type":       authType,
		"exp":        time.Now().Add(time.Hour * 6).Unix(),
	})
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return ""
	}
	return tokenString
}

func DecodeToken(tokenString, secretKey string) (string, string, error) {
	var identifier string
	var authType string

	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return identifier, authType, err
	}

	identifier, _ = claims["identifier"].(string)
	authType, _ = claims["type"].(string)

	return identifier, authType, nil
}

func (auth *Authentication) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := auth.users.List(r.Context())
	if err != nil {
		utils.WriteErrorResponse(w, utils.BadReqError("unable to list users"))
		return
	}

	sanitized := []User{}
	for _, u := range users {
		sanitized = append(sanitized, u.Sanatize())
	}
	utils.WriteData(w, sanitized)
}

func (auth *Authentication) HandleCreateUsers(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	user := NewUser()
	err := decoder.Decode(&user)
	if err != nil {
		utils.WriteErrorResponse(w, utils.BadReqError("unable to create user"))
		return
	}
	if auth.users.Create(user, r.Context()) != nil {
		utils.WriteErrorResponse(w, utils.BadReqError("unable to create user"))
		return
	}
	utils.WriteStatus(w, http.StatusCreated)
}

func (auth *Authentication) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	user := NewUser()
	err := decoder.Decode(&user)
	if err != nil {
		utils.WriteErrorResponse(w, utils.BadReqError("unable to update user"))
		return
	}
	if err := auth.users.Update(user, chi.URLParam(r, "id"), r.Context()); err != nil {
		utils.WriteErrorResponse(w, utils.BadReqError("unable to update user"))
		return
	}
	utils.WriteStatus(w, http.StatusCreated)
}

func (auth *Authentication) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if auth.users.Delete(chi.URLParam(r, "id"), r.Context()) != nil {
		utils.WriteErrorResponse(w, utils.BadReqError("unable to create user"))
		return
	}
	utils.WriteStatus(w, http.StatusOK)
}

func (auth *Authentication) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	user, err := auth.GetUser(w, r)
	if err != nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	utils.WriteData(w, user.Sanatize())
}

// Profile changes derive identity and permissions from the session, never the request body.
func (auth *Authentication) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	user, err := auth.GetUser(w, r)
	if err != nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	if user.Type == "openid" {
		http.Error(w, "Profile managed by identity provider", http.StatusForbidden)
		return
	}
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&input); err != nil {
		http.Error(w, "Invalid profile", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Name) == "" || !strings.Contains(input.Email, "@") ||
		((user.PasswordReset || input.Password != "") && (len(input.Password) < 8 || len(input.Password) > 72)) {
		http.Error(w, "Invalid profile or password", http.StatusBadRequest)
		return
	}
	user.Name, user.Email, user.Password = strings.TrimSpace(input.Name), strings.TrimSpace(input.Email), input.Password
	if err := auth.users.Update(user, user.ID.String(), r.Context()); err != nil {
		http.Error(w, "Unable to update profile", http.StatusBadRequest)
		return
	}
	auth.HandleGetProfile(w, r)
}
