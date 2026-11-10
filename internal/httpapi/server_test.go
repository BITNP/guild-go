package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	authz "git.bitnp.net/BITNP/bitnp-authz"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/httpapi"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

func TestProtectedRoutes(t *testing.T) {
	sessions := auth.NewStore(time.Minute)
	id, _, err := sessions.Create("member", "Member", "", "", authz.FromRoles(nil))
	if err != nil {
		t.Fatal(err)
	}
	service := auth.NewService(nil, sessions, nil, false, nil)
	handler := httpapi.New(nil, service, nil, nil, nil).Handler()

	for _, route := range []string{
		"GET /api/auth/csrf", "GET /api/auth/me", "POST /api/auth/logout",
		"GET /api/users", "GET /api/users/member", "GET /api/overview",
		"GET /api/nodes", "GET /api/nodes/1", "POST /api/nodes", "PUT /api/nodes/order",
		"PUT /api/nodes/1", "PUT /api/nodes/1/move", "DELETE /api/nodes/1",
		"GET /api/nodes/1/members", "PUT /api/nodes/1/members",
		"GET /api/nodes/1/comments", "GET /api/nodes/1/progress", "POST /api/nodes/1/comments",
		"POST /api/nodes/1/comment-images", "DELETE /api/comment-images/1", "DELETE /api/comments/1",
		"GET /api/comments/1/reactions/heart", "PUT /api/comments/1/reactions/heart", "GET /api/attachments/1",
	} {
		t.Run(route, func(t *testing.T) {
			method, path, _ := strings.Cut(route, " ")
			request := httptest.NewRequest(method, path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("anonymous request: status=%d body=%s", response.Code, response.Body)
			}
			if method == http.MethodGet {
				return
			}

			for _, token := range []string{"", "wrong-token"} {
				request := httptest.NewRequest(method, path, nil)
				request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: id})
				request.Header.Set("X-CSRF-Token", token)
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"CSRF_INVALID"`) {
					t.Fatalf("invalid CSRF: status=%d body=%s", response.Code, response.Body)
				}
			}
		})
	}
}

func TestSessionEndpoints(t *testing.T) {
	sessions := auth.NewStore(time.Minute)
	id, csrf, err := sessions.Create("member", "Member", "", "", authz.FromRoles(nil))
	if err != nil {
		t.Fatal(err)
	}
	handler := httpapi.New(nil, auth.NewService(nil, sessions, nil, false, nil), users.NewService(nil, nil), nil, nil).Handler()
	for _, test := range []struct {
		method, path, token string
		want                int
	}{
		{http.MethodGet, "/api/auth/csrf", "", http.StatusOK},
		{http.MethodGet, "/api/auth/me", "", http.StatusOK},
		{http.MethodPost, "/api/auth/logout", "wrong-token", http.StatusForbidden},
		{http.MethodGet, "/api/auth/me", "", http.StatusOK},
		{http.MethodPost, "/api/auth/logout", csrf, http.StatusNoContent},
		{http.MethodGet, "/api/auth/me", "", http.StatusUnauthorized},
	} {
		request := httptest.NewRequest(test.method, test.path, nil)
		request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: id})
		request.Header.Set("X-CSRF-Token", test.token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Fatalf("%s %s: status=%d body=%s", test.method, test.path, response.Code, response.Body)
		}
		if test.path == "/api/auth/csrf" {
			var result struct{ Token, HeaderName string }
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.Token != csrf || result.HeaderName != "X-CSRF-Token" {
				t.Fatalf("session CSRF response: %s, %v", response.Body, err)
			}
		}
		if test.want == http.StatusNoContent {
			cookies := response.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Name != auth.SessionCookieName || cookies[0].MaxAge != -1 {
				t.Fatalf("logout must expire the session cookie: %+v", cookies)
			}
		}
	}
}

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
