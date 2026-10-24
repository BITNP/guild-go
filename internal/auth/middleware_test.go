package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestDevAuthBypassProvidesAdminSession(t *testing.T) {
	publicURL, err := url.Parse("http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(nil, NewStore(time.Minute), NewPendingStore(), publicURL, false, nil)
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
