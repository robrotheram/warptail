package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"warptail/pkg/utils"

	"github.com/go-chi/chi/v5"
)

func TestSafeRedirect(t *testing.T) {
	base := "https://warp.example"
	for _, input := range []string{"https://evil.example", "//evil.example", "/\\evil.example", "/%5cevil.example", "javascript:alert(1)", "https://warp.example@evil.example", "%invalid", "\n//evil.example"} {
		if got := SafeRedirect(input, base, nil); got != base+"/login" {
			t.Errorf("%q redirected to %q", input, got)
		}
	}
	if got := SafeRedirect("/login?next=%2Fsettings#section", base, nil); got != base+"/login?next=%2Fsettings#section" {
		t.Fatal(got)
	}
	allowed := func(target *url.URL) bool { return target.Scheme == "https" && target.Host == "app.example" }
	if got := SafeRedirect("https://app.example/path?existing=yes&token=old#hash", base, allowed); got != "https://app.example/path?existing=yes#hash" {
		t.Fatal(got)
	}
}

func TestDashboardRequestSecurity(t *testing.T) {
	handler := DashboardRequestSecurity("https://warp.example")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, tc := range []struct {
		name, origin, contentType, authorization string
		want                                     int
	}{
		{"same origin", "https://warp.example", "application/json", "", 204},
		{"cross origin form", "https://evil.example", "application/x-www-form-urlencoded", "", 403},
		{"null origin", "null", "application/json", "", 403},
		{"originless form", "", "application/x-www-form-urlencoded", "", 403},
		{"API client", "", "application/json", "token", 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "https://warp.example/auth/logout", nil)
			request.Header.Set("Origin", tc.origin)
			request.Header.Set("Content-Type", tc.contentType)
			request.Header.Set("Authorization", tc.authorization)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.want {
				t.Fatalf("got %d, want %d", response.Code, tc.want)
			}
		})
	}
}

func testAuthentication(t *testing.T) (*Authentication, *chi.Mux, User) {
	t.Helper()
	db, err := utils.NewDatabase(utils.DatabaseConfig{ConnectionType: utils.SQLITE, ConnectionString: filepath.Join(t.TempDir(), "auth.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.NewCreateTable().Model((*User)(nil)).Exec(context.Background()); err != nil {
		t.Fatal(err)
	}
	mux := chi.NewRouter()
	authentication := NewAuthentication(mux, db, utils.AuthenticationConfig{BaseURL: "https://warp.example", SessionSecret: "test-session-secret-at-least-32-bytes"})
	user := NewUser()
	user.Name, user.Email, user.Role, user.Password, user.PasswordReset = "Test User", "user@example.com", USER, "OldPassword1!", true
	if err := authentication.users.Create(user, context.Background()); err != nil {
		t.Fatal(err)
	}
	return authentication, mux, user
}

func TestProfileUpdateUsesSessionIdentityAndPreservesRole(t *testing.T) {
	authentication, mux, user := testAuthentication(t)
	token := GenerateToken(user.ID.String(), JWTAUTH, authentication.JWTProvider.secretKey)
	request := httptest.NewRequest(http.MethodPost, "/auth/profile", strings.NewReader(`{"id":"someone-else","name":"Updated Name","email":"updated@example.com","role":"admin","type":"openid","password":"NewPassword1!"}`))
	request.Header.Set("Authorization", token)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	var profile User
	if err := json.Unmarshal(response.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.ID != user.ID || profile.Role != USER || profile.Type != user.Type || profile.Password != "" || profile.PasswordReset || profile.Name != "Updated Name" {
		t.Fatalf("incorrect profile: %+v", profile)
	}
	saved, err := authentication.users.FindByID(user.ID.String(), context.Background())
	if err != nil || !saved.VerifyPassword("NewPassword1!") {
		t.Fatal("password was not saved")
	}
}

func TestPasswordResetCannotBeSkipped(t *testing.T) {
	authentication, mux, user := testAuthentication(t)
	request := httptest.NewRequest(http.MethodPost, "/auth/profile", strings.NewReader(`{"name":"Test User","email":"user@example.com","password_reset":false}`))
	request.Header.Set("Authorization", GenerateToken(user.ID.String(), JWTAUTH, authentication.JWTProvider.secretKey))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("got %d", response.Code)
	}
	handler := authentication.DashboardMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("protected handler called before password reset")
	}))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("got %d", response.Code)
	}
}

func TestUnauthenticatedProfileAndLogout(t *testing.T) {
	_, mux, _ := testAuthentication(t)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/profile", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", response.Code)
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
	cookies := response.Result().Cookies()
	if len(cookies) == 0 || cookies[0].MaxAge != -1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("invalid logout cookie: %+v", cookies)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("session response can be cached")
	}
}

func TestProxyReturnURLPreservesHTTPSAndRemovesToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "https://app.example/path?token=secret&tab=logs", nil)
	request.URL.Scheme = "" // Server requests normally have a relative request URI.
	if got := ParseUrlFromRequest(request); got != "https://app.example/path?tab=logs" {
		t.Fatal(got)
	}
}

func TestExplicitCredentialsOverrideAnOlderCookie(t *testing.T) {
	authentication, _, first := testAuthentication(t)
	second := NewUser()
	second.Name, second.Email, second.Role, second.Password = "Second User", "second@example.com", ADMIN, "Password123!"
	if err := authentication.users.Create(second, context.Background()); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/auth/profile", nil)
	cookieResponse := httptest.NewRecorder()
	session, _ := authentication.sessionStore.Get(request, "auth-session")
	session.Values["jwt"] = GenerateToken(first.ID.String(), JWTAUTH, authentication.JWTProvider.secretKey)
	if err := session.Save(request, cookieResponse); err != nil {
		t.Fatal(err)
	}
	request.AddCookie(cookieResponse.Result().Cookies()[0])
	request.Header.Set("Authorization", GenerateToken(second.ID.String(), JWTAUTH, authentication.JWTProvider.secretKey))
	response := httptest.NewRecorder()
	actual, err := authentication.GetUser(response, request)
	if err != nil || actual.ID != second.ID {
		t.Fatalf("explicit credentials did not select the second user: %v", err)
	}
	request.Header.Set("Authorization", "expired-token")
	response = httptest.NewRecorder()
	if _, err := authentication.GetUser(response, request); err == nil {
		t.Fatal("invalid header fell back to cookie")
	}
	if len(response.Result().Cookies()) > 0 {
		t.Fatal("a late invalid bearer request cleared the browser's newer cookie")
	}
}
