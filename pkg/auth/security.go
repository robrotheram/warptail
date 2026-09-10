package auth

import (
	"net/http"
	"net/url"
	"strings"
	"unicode"
)

// SafeRedirect permits the dashboard origin and explicitly configured proxy destinations.
func SafeRedirect(value, baseURL string, allowed func(*url.URL) bool) string {
	base, err := url.Parse(baseURL)
	if err != nil || base.Host == "" {
		return "/login"
	}
	fallback := base.ResolveReference(&url.URL{Path: "/login"}).String()
	decoded, err := url.QueryUnescape(value)
	if err != nil || value == "" || strings.HasPrefix(decoded, "//") || strings.ContainsAny(decoded, "\\") || strings.ContainsFunc(decoded, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return fallback
	}
	target, err := url.Parse(value)
	if err != nil || target.User != nil {
		return fallback
	}
	target = base.ResolveReference(target)
	if target.Scheme != "http" && target.Scheme != "https" {
		return fallback
	}
	if !(target.Scheme == base.Scheme && strings.EqualFold(target.Host, base.Host)) && (allowed == nil || !allowed(target)) {
		return fallback
	}
	query := target.Query()
	query.Del("token")
	target.RawQuery = query.Encode()
	return target.String()
}

// DashboardRequestSecurity runs after proxy dispatch, so upstream applications keep their own policy.
func DashboardRequestSecurity(baseURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
				origin := r.Header.Get("Origin")
				if origin != "" {
					parsed, err := url.Parse(origin)
					base, _ := url.Parse(baseURL)
					if err != nil || parsed.Host == "" || (parsed.Host != r.Host && (base == nil || parsed.Host != base.Host || parsed.Scheme != base.Scheme)) {
						http.Error(w, "Cross-origin request denied", http.StatusForbidden)
						return
					}
				} else if r.Header.Get("Authorization") == "" && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
					http.Error(w, "JSON or authorization required", http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
