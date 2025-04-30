package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

func NewAuthentication(mux *chi.Mux, db *bun.DB, config utils.AuthenticationConfig) *Authentication {
	auth := Authentication{
		baseUrl: config.BaseURL,
	}

	providerConfig := config.Provider
	providerConfig.BaseURL = config.BaseURL
	providerConfig.Secret = config.Secret

	auth.sessionStore = sessions.NewCookieStore([]byte(config.Secret))
	auth.sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
		Secure:   false,
	}
	auth.users = NewUserStore(db)

	if provider, err := NewOpenIdProvider(providerConfig, auth.sessionStore, auth.users); err == nil {
		auth.OICDProvider = provider
		mux.HandleFunc("/auth/callback", auth.OICDProvider.Callback)
	}
	// else {
	// 	utils.Logger.Error(err, "unable to create oidc")
	// }

	if provider, err := NewJWTAuthProvider(providerConfig, auth.sessionStore, auth.users); err == nil {
		auth.JWTProvider = provider
	}
	mux.HandleFunc("/auth/login", auth.Login)
	mux.HandleFunc("/auth/logout", auth.Logout)
	mux.HandleFunc("/auth/profile", auth.HandleGetProfile)
	return &auth
}

func ParseUrlFromRequest(r *http.Request) string {
	fullURL := r.URL.Scheme + "://" + r.Host + r.URL.RequestURI()
	if r.URL.Scheme == "" {
		fullURL = "http://" + r.Host + r.URL.RequestURI()
	}
	parsedURL, _ := url.Parse(fullURL)
	queryParams := parsedURL.Query()
	queryParams.Del("token")
	parsedURL.RawQuery = queryParams.Encode()
	return parsedURL.String()
}

func (auth *Authentication) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		auth.OICDProvider.Login(w, r)
	} else {
		auth.JWTProvider.Login(w, r)
	}
}

func (auth *Authentication) Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.sessionStore.Get(r, "auth-session")
	session.Options.MaxAge = -1 // Delete session
	session.Save(r, w)
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
	token, ok := session.Values["jwt"].(string)
	if !ok {
		token = r.Header.Get("Authorization")
	}
	identifier, tokenType, err := DecodeToken(token, auth.JWTProvider.secretKey)
	if err != nil {
		session.Options.MaxAge = -1
		session.Save(r, w)
		return user, err
	}
	switch tokenType {
	case JWTAUTH:
		return auth.JWTProvider.GetUser(identifier)
	case OIDCAUTH:
		return auth.OICDProvider.GetUser(identifier, r.Context())
	}
	return user, fmt.Errorf("no authentication providers found")
}

func (auth *Authentication) DashboardAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := auth.GetUser(w, r)
		if err != nil {
			http.Error(w, "Authentication failed. Invalid token.", http.StatusUnauthorized)
			utils.LogHttpError(r, fmt.Errorf("Authentication failed. Invalid token: %v", err))
			return
		}
		if user.Role != ADMIN {
			http.Error(w, "Authentication failed. Permission Denied.", http.StatusUnauthorized)
			utils.LogHttpError(r, fmt.Errorf("Authentication failed. Permission Denied: %v", err))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (auth *Authentication) DashboardMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := auth.GetUser(w, r)
		if err != nil {
			http.Error(w, "Authentication failed. Invalid token.", http.StatusUnauthorized)
			utils.LogHttpError(r, fmt.Errorf("Authentication failed. Invalid token: %v", err))
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
	}
	if auth.users.Create(user, r.Context()) != nil {
		utils.WriteErrorResponse(w, utils.BadReqError("unable to create user"))
	}
	utils.WriteStatus(w, http.StatusCreated)
}

func (auth *Authentication) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	user := NewUser()
	err := decoder.Decode(&user)
	if err != nil {
		utils.WriteErrorResponse(w, utils.BadReqError("unable to update user"))
	}
	if err := auth.users.Update(user, chi.URLParam(r, "id"), r.Context()); err != nil {
		utils.WriteErrorResponse(w, utils.BadReqError("unable to update user"))
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
	user, err := auth.GetUser(w, r)
	if err != nil {
		utils.WriteErrorResponse(w, utils.BadReqError("unable find profile"))
		return
	}
	utils.WriteData(w, user.Sanatize())
}
