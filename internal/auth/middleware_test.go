package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestDevAuthBypassProvidesAdminSession(t *testing.T) {
	service := newTestService(t)
	if err := service.EnableDevAuthBypass("dev-user", "Development User"); err != nil {
		t.Fatal(err)
	}

	var got Session
	handler := service.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := SessionFrom(r.Context())
		if !ok {
			t.Fatal("development session missing from request context")
		}
		got = session
		w.WriteHeader(http.StatusNoContent)
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/overview", nil))

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d", response.Code)
	}
	if got.Subject != "dev-user" || got.Name != "Development User" {
		t.Fatalf("unexpected development session: subject=%q name=%q", got.Subject, got.Name)
	}
	if !got.Permissions.Admitted || !got.Permissions.SystemAdmin || !got.Permissions.Manager {
		t.Fatalf("development session must have administrator permissions: %+v", got.Permissions)
	}
	if got.CSRF == "" {
		t.Fatal("development session must retain a CSRF token")
	}
}

func TestRequireSessionRejectsAnonymousRequest(t *testing.T) {
	service := newTestService(t)
	handler := service.RequireSession(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("anonymous request must not reach the handler")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/nodes", nil))

	assertAuthError(t, response, "需要登录")
}

func TestRequireSessionReportsExpiredSession(t *testing.T) {
	service := newTestService(t)
	handler := service.RequireSession(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("expired session must not reach the handler")
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "unknown-or-expired"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assertAuthError(t, response, "登录已过期")
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	publicURL, err := url.Parse("http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	return NewService(nil, NewStore(time.Minute), NewPendingStore(), publicURL, false, nil)
}

func assertAuthError(t *testing.T, response *httptest.ResponseRecorder, message string) {
	t.Helper()
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", response.Code)
	}
	var body struct {
		Status  int    `json:"status"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != http.StatusUnauthorized || body.Error != "UNAUTHORIZED" || body.Message != message {
		t.Fatalf("unexpected auth error: %+v", body)
	}
}
