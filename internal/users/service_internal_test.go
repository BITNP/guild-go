package users

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLookupKeycloakSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/realms/bitnp/protocol/openid-connect/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "service-token",
				"expires_in":   300,
			})
		case "/admin/realms/bitnp/users/sub-leader":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":        "sub-leader",
				"username":  "alice",
				"firstName": "Alice",
				"lastName":  "Wang",
				"enabled":   true,
			})
		case "/admin/realms/bitnp/users/sub-leader/role-mappings/realm/composite":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"name": "permission-techdept-admin"},
				{"name": "permission-techdept-access"},
			})
		case "/admin/realms/bitnp/users/sub-disabled":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "sub-disabled",
				"enabled": false,
			})
		case "/admin/realms/bitnp/users/sub-unadmitted":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       "sub-unadmitted",
				"username": "outsider",
				"enabled":  true,
			})
		case "/admin/realms/bitnp/users/sub-unadmitted/role-mappings/realm/composite":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"name": "default-roles-bitnp"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	kc := NewKeycloakAdmin(
		server.URL+"/admin/realms/bitnp",
		"guild-admin",
		"secret",
		server.URL+"/realms/bitnp",
	)
	svc := NewService(nil, NewDirectory(kc))
	ctx := context.Background()

	session, ok, err := svc.lookupKeycloakSession(ctx, "sub-leader")
	if err != nil {
		t.Fatalf("lookup admitted leader: %v", err)
	}
	if !ok {
		t.Fatal("expected sub-leader to be admitted")
	}
	if session.Name != "Alice Wang" || !session.Permissions.Manager {
		t.Fatalf("unexpected session: %+v", session)
	}

	if _, ok, err := svc.lookupKeycloakSession(ctx, "sub-disabled"); err != nil || ok {
		t.Fatalf("disabled user must be rejected: ok=%v err=%v", ok, err)
	}
	if _, ok, err := svc.lookupKeycloakSession(ctx, "sub-unadmitted"); err != nil || ok {
		t.Fatalf("unadmitted user must be rejected: ok=%v err=%v", ok, err)
	}
	if _, ok, err := svc.lookupKeycloakSession(ctx, "sub-missing"); err != nil || ok {
		t.Fatalf("missing user must be rejected: ok=%v err=%v", ok, err)
	}
}
