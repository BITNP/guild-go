package httpapi_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	authz "git.bitnp.net/BITNP/bitnp-authz"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/config"
	"git.bitnp.net/BITNP/guild-go/internal/httpapi"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

func TestLoginEmailInCurrentProfile(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name       string
		claims     map[string]any
		want       string
		loginError string
	}{
		{"verified", map[string]any{"email": "member@example.com", "email_verified": true}, "member@example.com", ""},
		{"unverified", map[string]any{"email": "member@example.com", "email_verified": false}, "member@example.com", ""},
		{"verification omitted", map[string]any{"email": " member@example.com "}, "member@example.com", ""},
		{"email omitted", map[string]any{"email_verified": true}, "", ""},
		{"wrong nonce", map[string]any{"email": "member@example.com", "nonce": "wrong"}, "", "oidc_failed"},
		{"no membership", map[string]any{"email": "member@example.com", "realm_access": map[string]any{"roles": []string{}}}, "", "no_roles"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var issuer *httptest.Server
			issuer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/.well-known/openid-configuration":
					_ = json.NewEncoder(w).Encode(map[string]any{
						"issuer": issuer.URL, "authorization_endpoint": issuer.URL + "/authorize",
						"token_endpoint": issuer.URL + "/token", "jwks_uri": issuer.URL + "/keys",
						"id_token_signing_alg_values_supported": []string{"RS256"},
					})
				case "/keys":
					_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, Algorithm: "RS256", Use: "sig"}}})
				case "/token":
					claims := map[string]any{
						"iss": issuer.URL, "aud": "guild", "sub": "member-subject",
						"exp": time.Now().Add(time.Minute).Unix(), "nonce": "login-nonce",
						"name": "Member", "realm_access": map[string]any{"roles": []string{authz.SystemAdminRole}},
					}
					for claim, value := range test.claims {
						claims[claim] = value
					}
					token, err := jwt.Signed(signer).Claims(claims).Serialize()
					if err != nil {
						t.Error(err)
						http.Error(w, "token signing failed", http.StatusInternalServerError)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"id_token": token, "access_token": "test-access-token", "token_type": "Bearer"})
				default:
					http.NotFound(w, r)
				}
			}))
			defer issuer.Close()

			sessions := auth.NewStore(time.Minute)
			pending := auth.NewPendingStore()
			pending.Put("login-state", auth.PendingLogin{Nonce: "login-nonce", Verifier: "test-verifier", ReturnTo: "/account"})
			publicURL, _ := url.Parse("http://guild.example")
			authService := auth.NewService(
				auth.NewOIDC(issuer.URL, "guild", "test-secret", publicURL.String()+"/api/auth/callback"),
				sessions, pending, publicURL, false,
				func(context.Context, string) error { return nil },
			)
			handler := httpapi.New(config.Config{}, nil, authService, users.NewService(nil, nil), nil, nil, nil).Handler()
			callback := httptest.NewRecorder()
			handler.ServeHTTP(callback, httptest.NewRequest(http.MethodGet, "/api/auth/callback?state=login-state&code=test-code", nil))
			cookies := callback.Result().Cookies()
			if test.loginError != "" {
				if callback.Code != http.StatusFound || len(cookies) != 0 || callback.Header().Get("Location") != "/login?error="+test.loginError {
					t.Fatal("email must not bypass login validation")
				}
				return
			}
			if callback.Code != http.StatusFound || callback.Header().Get("Location") != "/account" || len(cookies) != 1 {
				t.Fatalf("login failed: status=%d, location=%q", callback.Code, callback.Header().Get("Location"))
			}
			request := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
			request.AddCookie(cookies[0])
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var status struct {
				State string        `json:"state"`
				User  users.Profile `json:"user"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK || status.State != "active" || status.User.Subject != "member-subject" {
				t.Fatalf("invalid current profile: status=%d, state=%q, subject=%q", response.Code, status.State, status.User.Subject)
			}
			if test.want == "" {
				if status.User.Email != nil {
					t.Fatalf("missing email became %q", *status.User.Email)
				}
			} else if status.User.Email == nil || *status.User.Email != test.want {
				t.Errorf("current profile must preserve the provider email %q", test.want)
			}
		})
	}
}
