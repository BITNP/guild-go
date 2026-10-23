package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/config"
	"git.bitnp.net/BITNP/guild-go/internal/httpapi"
)

func TestPagePolicyAllowsLocalImagePreviews(t *testing.T) {
	server := httpapi.New(config.Config{}, nil, &auth.Service{}, nil, nil, nil, nil)
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
