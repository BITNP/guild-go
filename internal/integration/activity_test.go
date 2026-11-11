package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/config"
	"git.bitnp.net/BITNP/guild-go/internal/httpapi"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
)

func TestRecentActivityFromExistingData(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "活跃节点", LeaderSubjects: []string{"alice"}})
	if err != nil {
		t.Fatal(err)
	}
	overview, err := f.nodes.Overview(ctx, f.alice, false)
	if err != nil || len(overview.RecentActivities) != 1 {
		t.Fatalf("creation: %+v, %v", overview.RecentActivities, err)
	}
	initial := overview.RecentActivities[0]
	if initial.Kind != "node_created" || initial.Author == nil || initial.Author.Subject != "admin" || !initial.CreatedAt.Equal(root.CreatedAt) {
		t.Fatalf("creation: %+v", initial)
	}
	due := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	update := nodes.UpdateRequest{Title: root.Title, DueAt: &due, Completed: true}
	if _, err := f.nodes.Update(ctx, f.alice, root.ID, update); err != nil {
		t.Fatal(err)
	}
	overview, err = f.nodes.Overview(ctx, f.alice, false)
	if err != nil || len(overview.RecentActivities) != 1 || overview.RecentActivities[0].CompletedAt == nil {
		t.Fatalf("completed node: %+v, %v", overview.RecentActivities, err)
	}
	update.Completed = false
	reopened, err := f.nodes.Update(ctx, f.alice, root.ID, update)
	if err != nil {
		t.Fatal(err)
	}
	comment, err := f.comments.Create(ctx, f.alice, root.ID, "最新进展", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.comments.ToggleReaction(ctx, comment.ID, "alice", "thumbs_up"); err != nil {
		t.Fatal(err)
	}
	overview, err = f.nodes.Overview(ctx, f.alice, false)
	if err != nil || len(overview.RecentActivities) != 2 {
		t.Fatalf("current activity only: %+v, %v", overview.RecentActivities, err)
	}
	got := overview.RecentActivities[0]
	if got.Kind != "comment" || got.ID != comment.ID || got.Author == nil || got.Author.Subject != "alice" || got.BodyPreview != "最新进展" || len(got.ReactionSummary) != 1 || !got.ReactionSummary[0].ReactedByMe {
		t.Fatalf("comment: %+v", got)
	}
	latest := overview.RecentActivities[1]
	if latest.Kind != "node_updated" || latest.Author != nil || latest.CompletedAt != nil || latest.DueAt == nil || !latest.DueAt.Equal(due) || !latest.CreatedAt.Equal(reopened.UpdatedAt) {
		t.Fatalf("latest node update must not invent an actor or history: %+v", latest)
	}
	// A change made directly to existing fields is visible without any event write.
	if _, err := f.pool.Exec(ctx, `UPDATE nodes SET updated_at=clock_timestamp() WHERE id=$1`, root.ID); err != nil {
		t.Fatal(err)
	}
	overview, err = f.nodes.Overview(ctx, f.alice, false)
	if err != nil || len(overview.RecentActivities) != 2 || overview.RecentActivities[0].Kind != "node_updated" {
		t.Fatalf("direct update sorting: %+v, %v", overview.RecentActivities, err)
	}
	if len(overview.RecentComments) != 1 {
		t.Fatalf("legacy comments: %+v", overview.RecentComments)
	}
	if err := f.nodes.Delete(ctx, f.alice, root.ID); err != nil {
		t.Fatal(err)
	}
	overview, err = f.nodes.Overview(ctx, f.alice, false)
	if err != nil || overview.RecentActivities == nil || len(overview.RecentActivities) != 0 {
		t.Fatalf("deleted nodes: %+v, %v", overview.RecentActivities, err)
	}
}

func TestRelatedActivityBeforeLimit(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	create := func(parent *int64, title, leader string) nodes.Detail {
		t.Helper()
		node, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{ParentID: parent, Title: title, LeaderSubjects: []string{leader}})
		if err != nil {
			t.Fatal(err)
		}
		return node
	}
	root := create(nil, "Alice root", "alice")
	child := create(&root.ID, "Inherited leader", "bob")
	memberRoot := create(nil, "Ancestor membership", "carol")
	memberChild := create(&memberRoot.ID, "Direct member", "bob")
	if _, err := f.nodes.ReplaceMembers(ctx, f.admin, memberChild.ID, nodes.MembersRequest{Leaders: []string{"bob"}, Members: []string{"alice"}}); err != nil {
		t.Fatal(err)
	}
	unrelated := create(&memberRoot.ID, "Sibling without membership", "carol")
	// Own actions remain relevant after the actor's membership has been removed.
	if _, err := f.comments.Create(ctx, f.alice, memberChild.ID, "我发布的进展", nil); err != nil {
		t.Fatal(err)
	}
	former := create(nil, "Former membership", "bob")
	if _, err := f.nodes.ReplaceMembers(ctx, f.admin, former.ID, nodes.MembersRequest{Leaders: []string{"bob"}, Members: []string{"alice"}}); err != nil {
		t.Fatal(err)
	}
	own, err := f.comments.Create(ctx, f.alice, former.ID, "旧节点上的本人操作", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.nodes.ReplaceMembers(ctx, f.admin, former.ID, nodes.MembersRequest{Leaders: []string{"bob"}}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 55; i++ {
		if _, err := f.comments.Create(ctx, f.carol, unrelated.ID, fmt.Sprintf("无关评论 %d", i), nil); err != nil {
			t.Fatal(err)
		}
	}
	all, err := f.nodes.Overview(ctx, f.alice, false)
	if err != nil || len(all.RecentActivities) != 50 {
		t.Fatalf("all feed limit: len=%d, %v", len(all.RecentActivities), err)
	}
	related, err := f.nodes.Overview(ctx, f.alice, true)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int64]bool{}
	seenOwn := false
	for _, item := range related.RecentActivities {
		seen[item.NodeID] = true
		if item.NodeID == unrelated.ID || item.NodeID == former.ID && (item.Kind != "comment" || item.ID != own.ID) {
			t.Fatalf("unrelated event included: %+v", item)
		}
		seenOwn = seenOwn || item.Kind == "comment" && item.ID == own.ID
	}
	for _, id := range []int64{root.ID, child.ID, memberRoot.ID, memberChild.ID} {
		if !seen[id] {
			t.Fatalf("missing related node %d in %+v", id, related.RecentActivities)
		}
	}
	if !seenOwn || len(related.Roots) != len(all.Roots) {
		t.Fatalf("own event or roots missing: %+v", related)
	}
	empty, err := f.nodes.Overview(ctx, f.dave, true)
	if err != nil || len(empty.RecentActivities) != 0 || empty.RecentActivities == nil {
		t.Fatalf("empty feed must be []: %+v, %v", empty.RecentActivities, err)
	}
}

func TestOverviewActivityHTTP(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if _, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Root"}); err != nil {
		t.Fatal(err)
	}
	sessions := auth.NewStore(time.Minute)
	id, _, err := sessions.Create("carol", "Carol", "", "", f.carol.Permissions)
	if err != nil {
		t.Fatal(err)
	}
	publicURL, _ := url.Parse("http://guild.example")
	handler := httpapi.New(config.Config{}, f.pool, auth.NewService(nil, sessions, nil, publicURL, false, nil), f.users, f.nodes, f.comments, f.store).Handler()
	for _, test := range []struct {
		query string
		login bool
		want  int
		count int
	}{
		{"", false, 401, 0}, {"", true, 200, 1}, {"?scope=all", true, 200, 1},
		{"?scope=related", true, 200, 0}, {"?scope=invalid", true, 400, 0},
	} {
		request := httptest.NewRequest(http.MethodGet, "/api/overview"+test.query, nil)
		if test.login {
			request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: id})
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Fatalf("query %q login=%v: status=%d body=%s", test.query, test.login, response.Code, response.Body.String())
		}
		if test.want == 200 {
			var overview nodes.Overview
			if err := json.Unmarshal(response.Body.Bytes(), &overview); err != nil || overview.RecentActivities == nil || len(overview.RecentActivities) != test.count {
				t.Fatalf("query %q: activities=%+v, %v", test.query, overview.RecentActivities, err)
			}
		}
	}
}
