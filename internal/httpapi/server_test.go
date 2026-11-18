package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	authz "git.bitnp.net/BITNP/bitnp-authz"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/httpapi"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

func TestPagePolicyAllowsLocalImagePreviews(t *testing.T) {
	server := httpapi.New(nil, &auth.Service{}, nil, nil, nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/login", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("login page status = %d", response.Code)
	}
	policy := make(map[string][]string)
	for _, directive := range strings.Split(response.Header().Get("Content-Security-Policy"), ";") {
		fields := strings.Fields(directive)
		if len(fields) > 0 {
			policy[fields[0]] = fields[1:]
		}
	}
	if !slices.Contains(policy["img-src"], "blob:") {
		t.Fatal("image policy blocks object URLs used by upload previews")
	}
	for _, directive := range []string{"default-src", "script-src", "connect-src"} {
		if !slices.Equal(policy[directive], []string{"'self'"}) {
			t.Fatalf("%s must remain restricted to self: %v", directive, policy[directive])
		}
	}
}

func TestSessionAndCSRFRoutes(t *testing.T) {
	sessions := auth.NewStore(time.Minute)
	sessionID, _, err := sessions.Create("alice", "Alice", "", "", authz.FromRoles([]string{"permission-techdept-access"}))
	if err != nil {
		t.Fatal(err)
	}
	publicURL, _ := url.Parse("http://guild.example")
	authService := auth.NewService(nil, sessions, nil, publicURL, false, nil)
	handler := httpapi.New(nil, authService, users.NewService(nil, nil), nil, nil).Handler()

	// Anonymous request to protected route -> 401 UNAUTHORIZED.
	anonRes := httptest.NewRecorder()
	handler.ServeHTTP(anonRes, httptest.NewRequest(http.MethodGet, "/api/auth/me", nil))
	if anonRes.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous /api/auth/me status = %d, want 401", anonRes.Code)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(anonRes.Body.Bytes(), &errBody); err != nil || errBody.Error != "UNAUTHORIZED" {
		t.Fatalf("unexpected anonymous error body: %s (%v)", anonRes.Body.String(), err)
	}

	// Authenticated request for CSRF token -> 200.
	csrfReq := httptest.NewRequest(http.MethodGet, "/api/auth/csrf", nil)
	csrfReq.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sessionID})
	csrfRes := httptest.NewRecorder()
	handler.ServeHTTP(csrfRes, csrfReq)
	if csrfRes.Code != http.StatusOK {
		t.Fatalf("GET /api/auth/csrf status = %d", csrfRes.Code)
	}
	var csrf struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(csrfRes.Body.Bytes(), &csrf); err != nil || csrf.Token == "" {
		t.Fatalf("invalid csrf payload: %s (%v)", csrfRes.Body.String(), err)
	}

	// POST /api/auth/logout without X-CSRF-Token -> 403 CSRF_INVALID.
	noCSRFReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	noCSRFReq.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sessionID})
	noCSRFRes := httptest.NewRecorder()
	handler.ServeHTTP(noCSRFRes, noCSRFReq)
	if noCSRFRes.Code != http.StatusForbidden {
		t.Fatalf("logout without CSRF status = %d, want 403", noCSRFRes.Code)
	}
	errBody.Error = ""
	if err := json.Unmarshal(noCSRFRes.Body.Bytes(), &errBody); err != nil || errBody.Error != "CSRF_INVALID" {
		t.Fatalf("unexpected CSRF error body: %s (%v)", noCSRFRes.Body.String(), err)
	}

	// POST /api/auth/logout with valid X-CSRF-Token -> 204.
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sessionID})
	logoutReq.Header.Set("X-CSRF-Token", csrf.Token)
	logoutRes := httptest.NewRecorder()
	handler.ServeHTTP(logoutRes, logoutReq)
	if logoutRes.Code != http.StatusNoContent {
		t.Fatalf("logout with CSRF status = %d, want 204", logoutRes.Code)
	}

	// Unknown /api/ path -> 404 NOT_FOUND.
	notFoundRes := httptest.NewRecorder()
	handler.ServeHTTP(notFoundRes, httptest.NewRequest(http.MethodGet, "/api/unknown", nil))
	if notFoundRes.Code != http.StatusNotFound {
		t.Fatalf("unknown API status = %d, want 404", notFoundRes.Code)
	}
	errBody.Error = ""
	if err := json.Unmarshal(notFoundRes.Body.Bytes(), &errBody); err != nil || errBody.Error != "NOT_FOUND" {
		t.Fatalf("unexpected 404 error body: %s (%v)", notFoundRes.Body.String(), err)
	}
}
