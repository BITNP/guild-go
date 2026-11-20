package httpapi_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	authz "git.bitnp.net/BITNP/bitnp-authz"

	"git.bitnp.net/BITNP/guild-go/internal/access"
	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/database"
	"git.bitnp.net/BITNP/guild-go/internal/events"
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
	handler := httpapi.New(nil, &auth.Service{}, nil, nil, nil).Handler()
	for _, route := range []string{
		"GET /api/internal/users",
		"GET /api/internal/overview",
		"GET /api/internal/nodes",
		"GET /api/internal/nodes/1",
		"POST /api/internal/nodes",
		"PUT /api/internal/nodes/1",
		"PUT /api/internal/nodes/1/move",
		"DELETE /api/internal/nodes/1",
		"PUT /api/internal/nodes/1/members",
		"GET /api/internal/nodes/1/progress",
		"POST /api/internal/nodes/1/comments",
		"GET /api/internal/events",
		"GET /api/internal/events/stream",
		"GET /api/internal/due-nodes",
		"GET /api/internal/activity-summary",
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
	handler := httpapi.New(nil, &auth.Service{}, nil, nil, nil).
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
	nodeSvc := nodes.NewService(nodes.NewRepository(pool), userSvc, commentSvc, checker, store)

	handler := httpapi.New(pool, &auth.Service{}, userSvc, nodeSvc, commentSvc).
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

type stubInternalFeed struct {
	mu          sync.Mutex
	events      []events.Event
	dueQuery    events.DueNodesQuery
	dueResponse events.DueNodesResponse
	summary     events.ActivitySummary
	summarySpan [2]time.Time
}

func (f *stubInternalFeed) LatestID(_ context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.events) == 0 {
		return 0, nil
	}
	return f.events[len(f.events)-1].ID, nil
}

func (f *stubInternalFeed) ListAfter(_ context.Context, after int64, limit int) (events.ListResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]events.Event, 0)
	for _, ev := range f.events {
		if ev.ID > after {
			items = append(items, ev)
		}
	}
	hasMore := false
	if len(items) > limit {
		hasMore = true
		items = items[:limit]
	}
	nextAfter := after
	if len(items) > 0 {
		nextAfter = items[len(items)-1].ID
	}
	var latest int64
	if len(f.events) > 0 {
		latest = f.events[len(f.events)-1].ID
	}
	return events.ListResponse{
		Items:      items,
		HasMore:    hasMore,
		NextAfter:  nextAfter,
		LatestID:   latest,
		ServerTime: time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (f *stubInternalFeed) ListDueNodes(_ context.Context, query events.DueNodesQuery) (events.DueNodesResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dueQuery = query
	return f.dueResponse, nil
}

func (f *stubInternalFeed) ActivitySummary(_ context.Context, start, end time.Time) (events.ActivitySummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.summarySpan = [2]time.Time{start, end}
	res := f.summary
	res.Start = start
	res.End = end
	return res, nil
}

func (f *stubInternalFeed) append(ev events.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, ev)
}

func TestInternalServiceFeedAuthAndQueries(t *testing.T) {
	nodeID := int64(42)
	feed := &stubInternalFeed{
		events: []events.Event{
			{
				ID:           1,
				Kind:         events.KindNodeCreated,
				NodeID:       &nodeID,
				ActorSubject: "alice",
				Payload:      json.RawMessage(`{"title":"Task A"}`),
				CreatedAt:    time.Date(2026, 11, 18, 8, 0, 0, 0, time.UTC),
			},
		},
		dueResponse: events.DueNodesResponse{
			Items: []events.DueNodeItem{
				{
					ID:      nodeID,
					Title:   "Task A",
					DueAt:   time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC),
					Leaders: []string{"alice"},
					Members: []string{"bob"},
				},
			},
			ServerTime: time.Date(2026, 11, 18, 8, 5, 0, 0, time.UTC),
		},
		summary: events.ActivitySummary{
			CreatedCount:   1,
			OpenCount:      2,
			CreatedNodes:   []events.ActivitySummaryNode{},
			CompletedNodes: []events.ActivitySummaryNode{},
			RecentComments: []events.ActivitySummaryComment{},
			OverdueNodes:   []events.ActivitySummaryNode{},
		},
	}

	// Note: WithInternalFeed alone does not require InternalAuthorizer / X-Act-As-Subject.
	handler := httpapi.New(nil, &auth.Service{}, nil, nil, nil).
		WithInternalFeed("service-secret", feed, events.NewBroker()).
		Handler()

	t.Run("rejects missing bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/internal/events", nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", res.Code)
		}
	})

	t.Run("lists events without X-Act-As-Subject", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/internal/events?after=0&limit=10", nil)
		req.Header.Set("Authorization", "Bearer service-secret")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (%s)", res.Code, res.Body.String())
		}
		var out events.ListResponse
		if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if len(out.Items) != 1 || out.NextAfter != 1 || out.LatestID != 1 {
			t.Fatalf("unexpected events response: %+v", out)
		}
	})

	t.Run("parses due-nodes parameters", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/internal/due-nodes?since=2026-11-18T08:00:00Z&advanceHours=24&includeOverdue=true&limit=50",
			nil,
		)
		req.Header.Set("Authorization", "Bearer service-secret")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (%s)", res.Code, res.Body.String())
		}
		if feed.dueQuery.Since == nil || feed.dueQuery.Advance != 24*time.Hour || !feed.dueQuery.IncludeOverdue || feed.dueQuery.Limit != 50 {
			t.Fatalf("unexpected parsed due query: %+v", feed.dueQuery)
		}
	})

	t.Run("validates activity-summary window", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/internal/activity-summary?start=2026-11-10T00:00:00Z", nil)
		req.Header.Set("Authorization", "Bearer service-secret")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 when only start given, got %d", res.Code)
		}

		req2 := httptest.NewRequest(
			http.MethodGet,
			"/api/internal/activity-summary?start=2026-11-09T16:00:00Z&end=2026-11-16T16:00:00Z",
			nil,
		)
		req2.Header.Set("Authorization", "Bearer service-secret")
		res2 := httptest.NewRecorder()
		handler.ServeHTTP(res2, req2)
		if res2.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (%s)", res2.Code, res2.Body.String())
		}
		if !feed.summarySpan[0].Equal(time.Date(2026, 11, 9, 16, 0, 0, 0, time.UTC)) {
			t.Fatalf("unexpected start passed to ActivitySummary: %v", feed.summarySpan[0])
		}
	})
}

func TestInternalEventsStreamSSELivePush(t *testing.T) {
	nodeID := int64(7)
	broker := events.NewBroker()
	feed := &stubInternalFeed{
		events: []events.Event{
			{
				ID:           1,
				Kind:         events.KindNodeCreated,
				NodeID:       &nodeID,
				ActorSubject: "alice",
				Payload:      json.RawMessage(`{"title":"Initial"}`),
				CreatedAt:    time.Now().UTC(),
			},
		},
	}

	handler := httpapi.New(nil, &auth.Service{}, nil, nil, nil).
		WithInternalFeed("stream-secret", feed, broker).
		Handler()

	srv := httptest.NewServer(handler)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/internal/events/stream?after=0", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer stream-secret")

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("unexpected content-type: %q", ct)
	}

	type sseFrame struct {
		id    string
		event string
		data  string
	}
	frames := make(chan sseFrame, 8)
	scanErr := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(resp.Body)
		var cur sseFrame
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				if cur.data != "" {
					frames <- cur
					cur = sseFrame{}
				}
				continue
			}
			if strings.HasPrefix(line, ":") {
				continue
			}
			if v, ok := strings.CutPrefix(line, "id: "); ok {
				cur.id = v
			} else if v, ok := strings.CutPrefix(line, "event: "); ok {
				cur.event = v
			} else if v, ok := strings.CutPrefix(line, "data: "); ok {
				cur.data = v
			}
		}
		scanErr <- scanner.Err()
	}()

	// Expect initial replay of event 1.
	select {
	case first := <-frames:
		if first.id != "1" || first.event != events.KindNodeCreated {
			t.Fatalf("unexpected initial frame: %+v", first)
		}
	case err := <-scanErr:
		t.Fatalf("scanner exited early: %v", err)
	case <-ctx.Done():
		t.Fatal("timed out waiting for initial SSE frame")
	}

	// Append event 2 and wake subscribers through Broker.
	feed.append(events.Event{
		ID:           2,
		Kind:         events.KindMemberAssigned,
		NodeID:       &nodeID,
		ActorSubject: "alice",
		Payload:      json.RawMessage(`{"subject":"bob","role":"member","nodeTitle":"Initial"}`),
		CreatedAt:    time.Now().UTC(),
	})
	broker.Publish(2)

	select {
	case second := <-frames:
		if second.id != "2" || second.event != events.KindMemberAssigned {
			t.Fatalf("unexpected live frame: %+v", second)
		}
		var parsed events.Event
		if err := json.Unmarshal([]byte(second.data), &parsed); err != nil {
			t.Fatal(err)
		}
		if parsed.ActorSubject != "alice" {
			t.Fatalf("unexpected parsed actorSubject: %+v", parsed)
		}
	case err := <-scanErr:
		t.Fatalf("scanner exited before live event: %v", err)
	case <-ctx.Done():
		t.Fatal("timed out waiting for live SSE frame")
	}
}
