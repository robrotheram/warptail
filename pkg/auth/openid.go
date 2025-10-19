package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"warptail/pkg/utils"

	"github.com/coreos/go-oidc"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

const OIDCAUTH = "OIDC"

type OpenIDAuth struct {
	oauth2Config *oauth2.Config
	provider     *oidc.Provider
	sessionStore *sessions.CookieStore
	users        *Users
	clientId     string
	secretKey    string
}

func NewOpenIdProvider(config utils.AuthenticationConfig, sessionStore *sessions.CookieStore, users *Users) (*OpenIDAuth, error) {

	if config.Provider.OIDC == nil {
		return nil, fmt.Errorf("OIDC configuration is missing")
	}

	redirectURL, _ := url.JoinPath(config.BaseURL, "/auth/callback")

	provider, err := oidc.NewProvider(context.Background(), config.Provider.OIDC.IssuerURL)
	if err != nil {
		return nil, err
	}

	// Configure OAuth2
	oauth2Config := &oauth2.Config{
		ClientID:    config.Provider.OIDC.ClientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: redirectURL,
		Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
	}

	auth := OpenIDAuth{
		clientId:     config.Provider.OIDC.ClientID,
		secretKey:    config.SessionSecret,
		sessionStore: sessionStore,
		users:        users,
		provider:     provider,
		oauth2Config: oauth2Config,
	}

	return &auth, nil
}

func (auth *OpenIDAuth) Login(w http.ResponseWriter, r *http.Request) {

	// Generate a random code verifier
	codeVerifier := generateRandomString(64)

	// Create a code challenge from the verifier
	codeChallenge := createCodeChallenge(codeVerifier)

	// Generate a random state (for CSRF protection)
	state := generateRandomString(32)

	// Save state and verifier in the session
	session, _ := auth.sessionStore.Get(r, "auth-session")
	session.Values["state"] = state
	session.Values["code_verifier"] = codeVerifier
	session.Values["redirect_next"] = r.URL.Query().Get("next")
	session.Save(r, w)

	// Redirect to the OIDC provider's authorization endpoint
	authURL := auth.oauth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	http.Redirect(w, r, authURL, http.StatusFound)
}

func (auth *OpenIDAuth) Callback(w http.ResponseWriter, r *http.Request) {
	// Retrieve state and code verifier from the session
	session, _ := auth.sessionStore.Get(r, "auth-session")
	storedState, ok := session.Values["state"].(string)
	if !ok || storedState != r.URL.Query().Get("state") {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	codeVerifier, ok := session.Values["code_verifier"].(string)
	if !ok {
		http.Error(w, "Missing code verifier", http.StatusInternalServerError)
		return
	}

	// Exchange code for token
	code := r.URL.Query().Get("code")
	token, err := auth.oauth2Config.Exchange(context.Background(), code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract and verify ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No ID token in response", http.StatusInternalServerError)
		return
	}

	verifier := auth.provider.Verifier(&oidc.Config{ClientID: auth.clientId})
	idToken, err := verifier.Verify(context.Background(), rawIDToken)
	if err != nil {
		http.Error(w, "Failed to verify ID token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Parse claims and save in session
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to parse ID token claims: "+err.Error(), http.StatusInternalServerError)
		return
	}

	auth.createUser(r.Context(), token)
	session.Values["authenticated"] = true
	if err := session.Save(r, w); err != nil {
		http.Error(w, "Failed to save session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if rediect, ok := session.Values["redirect_next"].(string); ok {
		rediect += "?token=" + GenerateToken(token.AccessToken, OIDCAUTH, auth.secretKey)
		http.Redirect(w, r, rediect, http.StatusTemporaryRedirect)
	}
	w.Write([]byte("Login successful! You can now access protected routes."))
}

func (auth *OpenIDAuth) GetUser(rawAccessToken string, ctx context.Context) (User, error) {
	oauth2Token := &oauth2.Token{
		AccessToken: rawAccessToken,
		TokenType:   "Bearer",
	}
	var user User
	userInfo, err := auth.provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		return user, err
	}
	var profile map[string]interface{}
	if err := userInfo.Claims(&profile); err != nil {
		return user, err
	}
	if email, ok := profile["email"].(string); ok {
		return auth.users.FindByEamil(email, ctx)
	}
	return user, fmt.Errorf("email not found")
}

func (auth *OpenIDAuth) createUser(ctx context.Context, oauth2Token *oauth2.Token) error {
	userInfo, err := auth.provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		return err
	}
	var profile map[string]interface{}
	if err := userInfo.Claims(&profile); err != nil {
		return err
	}

	name, ok := profile["name"].(string)
	if !ok {
		name = "Unknown User"
	}

	email, ok := profile["email"].(string)
	if !ok {
		return fmt.Errorf("email not found in profile")
	}

	user := User{
		ID:    uuid.New(),
		Name:  name,
		Email: email,
		Type:  "openid",
		Role:  ADMIN,
	}

	_, err = auth.users.FindByEamil(user.Email, ctx)
	if err != nil {
		auth.users.Create(user, ctx)
	}
	return nil
}

func createCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func generateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Failed to generate random string: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length]
}
