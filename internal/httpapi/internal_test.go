package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	authz "git.bitnp.net/BITNP/bitnp-authz"

	"git.bitnp.net/BITNP/guild-go/internal/access"
	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/config"
	"git.bitnp.net/BITNP/guild-go/internal/database"
	"git.bitnp.net/BITNP/guild-go/internal/httpapi"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
	"git.bitnp.net/BITNP/guild-go/internal/storage"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

type stubInternalAuthorizer struct {
	sessions        map[string]auth.Session
	validEnsure     map[string]bool
	resolvedActor   string
	ensuredSubjects []string
}

func (s *stubInternalAuthorizer) ResolveActorSession(_ context.Context, subject string) (auth.Session, error) {
	s.resolvedActor = subject
	session, ok := s.sessions[subject]
	if !ok {
		return auth.Session{}, apperr.Forbidden("用户不存在、已禁用或没有公会准入权限")
	}
	return session, nil
}

func (s *stubInternalAuthorizer) EnsureSubjects(_ context.Context, subjects []string) error {
	s.ensuredSubjects = append([]string(nil), subjects...)
	for _, subject := range subjects {
		if !s.validEnsure[subject] {
			return apperr.BadRequest("包含不存在或未准入的用户")
		}
	}
	return nil
}

func TestInternalRoutesDisabledWithoutToken(t *testing.T) {
	handler := httpapi.New(config.Config{}, nil, &auth.Service{}, nil, nil, nil, nil).Handler()
	for _, route := range []string{
		"GET /api/internal/users",
		"GET /api/internal/overview",
		"GET /api/internal/nodes",
		"GET /api/internal/nodes/1",
		"POST /api/internal/nodes",
		"PUT /api/internal/nodes/1",
		"PUT /api/internal/nodes/1/members",
		"GET /api/internal/nodes/1/progress",
		"POST /api/internal/nodes/1/comments",
	} {
		method, path, _ := strings.Cut(route, " ")
		req := httptest.NewRequest(method, path, nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusNotFound {
			t.Fatalf("%s expected 404 when disabled, got %d (%s)", route, res.Code, res.Body.String())
		}
	}
}

func TestInternalAuthValidation(t *testing.T) {
	authzStub := &stubInternalAuthorizer{
		sessions: map[string]auth.Session{
			"alice": {
				Subject:     "alice",
				Name:        "Alice",
				Permissions: authz.FromRoles([]string{"permission-techdept-admin"}),
			},
		},
		validEnsure: map[string]bool{"bob": true},
	}
	handler := httpapi.New(config.Config{}, nil, &auth.Service{}, nil, nil, nil, nil).
		WithInternalAPI("secret-internal-token", authzStub).
		Handler()

	t.Run("missing bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/nodes", strings.NewReader(`{}`))
		req.Header.Set("X-Act-As-Subject", "alice")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d (%s)", res.Code, res.Body.String())
		}
	})

	t.Run("wrong bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/nodes", strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer wrong-token")
		req.Header.Set("X-Act-As-Subject", "alice")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d (%s)", res.Code, res.Body.String())
		}
	})

	t.Run("missing acting subject", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/nodes", strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer secret-internal-token")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d (%s)", res.Code, res.Body.String())
		}
	})

	t.Run("unknown acting subject rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/nodes", strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer secret-internal-token")
		req.Header.Set("X-Act-As-Subject", "outsider")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d (%s)", res.Code, res.Body.String())
		}
	})

	t.Run("ensures subjects before creating node", func(t *testing.T) {
		body := strings.NewReader(`{"title":"T","leaderSubjects":["alice"],"memberSubjects":["unknown"]}`)
		req := httptest.NewRequest(http.MethodPost, "/api/internal/nodes", body)
		req.Header.Set("Authorization", "Bearer secret-internal-token")
		req.Header.Set("X-Act-As-Subject", "alice")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid ensured subject, got %d (%s)", res.Code, res.Body.String())
		}
		if authzStub.resolvedActor != "alice" {
			t.Fatalf("expected resolvedActor=alice, got %q", authzStub.resolvedActor)
		}
		if !slices.Equal(authzStub.ensuredSubjects, []string{"alice", "unknown"}) {
			t.Fatalf("unexpected ensuredSubjects: %v", authzStub.ensuredSubjects)
		}
	})

	t.Run("rejects malformed JSON on create node", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/nodes", strings.NewReader(`{"unknown":1}`))
		req.Header.Set("Authorization", "Bearer secret-internal-token")
		req.Header.Set("X-Act-As-Subject", "alice")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for unknown field, got %d (%s)", res.Code, res.Body.String())
		}
	})

	t.Run("ensures subjects before replacing members", func(t *testing.T) {
		body := strings.NewReader(`{"leaders":["alice"],"members":["unknown"]}`)
		req := httptest.NewRequest(http.MethodPut, "/api/internal/nodes/1/members", body)
		req.Header.Set("Authorization", "Bearer secret-internal-token")
		req.Header.Set("X-Act-As-Subject", "alice")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid ensured member, got %d (%s)", res.Code, res.Body.String())
		}
		if !slices.Equal(authzStub.ensuredSubjects, []string{"alice", "unknown"}) {
			t.Fatalf("unexpected ensuredSubjects: %v", authzStub.ensuredSubjects)
		}
	})
}

func TestInternalCreateNodeWithDatabase(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE comment_reactions, comment_attachments, comments, node_members, nodes, users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}

	kc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/realms/bitnp/protocol/openid-connect/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 300})
		case "/admin/realms/bitnp/users/manager-sub":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "manager-sub", "username": "mgr", "enabled": true})
		case "/admin/realms/bitnp/users/manager-sub/role-mappings/realm/composite":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "permission-techdept-admin"}})
		case "/admin/realms/bitnp/users/member-sub":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "member-sub", "username": "mem", "enabled": true})
		case "/admin/realms/bitnp/users/member-sub/role-mappings/realm/composite":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "permission-techdept-access"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer kc.Close()

	userRepo := users.NewRepository(pool)
	kcAdmin := users.NewKeycloakAdmin(kc.URL+"/admin/realms/bitnp", "c", "s", kc.URL+"/realms/bitnp")
	userSvc := users.NewService(userRepo, users.NewDirectory(kcAdmin))
	checker := access.New(pool)
	store := storage.NewMemoryStore()
	commentSvc := comments.NewService(comments.NewRepository(pool), userSvc, checker, store)
	nodeSvc := nodes.NewService(nodes.NewRepository(pool), userSvc, checker, store)

	handler := httpapi.New(config.Config{}, pool, &auth.Service{}, userSvc, nodeSvc, commentSvc, store).
		WithInternalAPI("bot-token", userSvc).
		Handler()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/internal/nodes",
		strings.NewReader(`{"title":"Bot Created Node","description":"Created via berry","memberSubjects":["member-sub"]}`),
	)
	req.Header.Set("Authorization", "Bearer bot-token")
	req.Header.Set("X-Act-As-Subject", "manager-sub")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.Code, res.Body.String())
	}

	var detail nodes.Detail
	if err := json.Unmarshal(res.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Title != "Bot Created Node" || detail.CreatedBy != "manager-sub" {
		t.Fatalf("unexpected node detail: %+v", detail)
	}
	if len(detail.DirectLeaders) != 1 || detail.DirectLeaders[0].Subject != "manager-sub" {
		t.Fatalf("expected creator defaulted as leader, got %+v", detail.DirectLeaders)
	}
	if len(detail.DirectMembers) != 1 || detail.DirectMembers[0].Subject != "member-sub" {
		t.Fatalf("expected member-sub provisioned & attached, got %+v", detail.DirectMembers)
	}
}
