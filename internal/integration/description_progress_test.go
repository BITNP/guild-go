package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/config"
	"git.bitnp.net/BITNP/guild-go/internal/httpapi"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
)

func TestDescriptionProgress(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{
		Title: "Description history", Description: "初始需求", LeaderSubjects: []string{"alice"}, MemberSubjects: []string{"bob"},
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := f.comments.Progress(ctx, f.carol, root.ID, nil, 30)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("initial description: %+v, %v", page, err)
	}
	initial := page.Items[0].DescriptionChange
	if initial == nil || initial.Before != "" || initial.After != "初始需求" || initial.Author.Subject != "admin" || !initial.CreatedAt.Equal(root.CreatedAt) {
		t.Fatalf("initial description event: %+v", initial)
	}
	input := nodes.UpdateRequest{Title: "Renamed", Description: root.Description, Completed: true}
	if _, err := f.nodes.Update(ctx, f.alice, root.ID, input); err != nil {
		t.Fatal(err)
	}
	input.Description = "未授权修改"
	if _, err := f.nodes.Update(ctx, f.bob, root.ID, input); apperr.From(err).Status != 403 {
		t.Fatalf("ordinary member edit: %v", err)
	}
	input.Description = "新需求\n保留完整文本 <script>alert(1)</script>"
	edited, err := f.nodes.Update(ctx, f.alice, root.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	comment, err := f.comments.Create(ctx, f.bob, root.ID, "确认新需求", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.comments.ToggleReaction(ctx, comment.ID, "carol", "thumbs_up"); err != nil {
		t.Fatal(err)
	}
	input.Description = ""
	if _, err := f.nodes.Update(ctx, f.admin, root.ID, input); err != nil {
		t.Fatal(err)
	}
	page, err = f.comments.Progress(ctx, f.bob, root.ID, nil, 30)
	if err != nil || len(page.Items) != 4 || page.HasMore || page.NextBefore != nil {
		t.Fatalf("combined timeline: %+v, %v", page, err)
	}
	if change := page.Items[0].DescriptionChange; change == nil || change.Before != edited.Description || change.After != "" || change.Author.Subject != "admin" {
		t.Fatalf("clear description: %+v", change)
	}
	if got := page.Items[1].Comment; got == nil || got.ID != comment.ID || !got.CanDelete || len(got.Reactions) != 1 || got.Reactions[0].Count != 1 {
		t.Fatalf("comment actions survive in progress: %+v", got)
	}
	if change := page.Items[2].DescriptionChange; change == nil || change.Before != root.Description || change.After != edited.Description || change.Author.Subject != "alice" || !change.CreatedAt.Equal(edited.UpdatedAt) {
		t.Fatalf("edit description: %+v", change)
	}
	oldAPI, err := f.comments.List(ctx, f.carol, root.ID, nil, 30)
	if err != nil || len(oldAPI.Items) != 1 || oldAPI.Items[0].ID != comment.ID || oldAPI.Items[0].CanDelete {
		t.Fatalf("comment API and viewer permissions: %+v, %v", oldAPI, err)
	}

	// Simulate a persistence failure after UPDATE but before the history insert.
	if _, err := f.pool.Exec(ctx, `ALTER TABLE node_description_changes ADD CONSTRAINT test_reject_description CHECK (new_description <> 'rejected')`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(ctx, `ALTER TABLE node_description_changes DROP CONSTRAINT IF EXISTS test_reject_description`)
	})
	input.Title, input.Description = "Must roll back", "rejected"
	if _, err := f.nodes.Update(ctx, f.alice, root.ID, input); err == nil {
		t.Fatal("history storage failure must fail the edit")
	}
	current, err := f.nodes.Get(ctx, f.alice, root.ID)
	if err != nil || current.Description != "" || current.Title != "Renamed" {
		t.Fatalf("description and history must commit atomically: %+v, %v", current, err)
	}
	if _, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Must not exist", Description: "rejected"}); err == nil {
		t.Fatal("history storage failure must roll back node creation")
	}
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM nodes`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("failed creation left a node: count=%d, %v", count, err)
	}
	if err := f.nodes.Delete(ctx, f.admin, root.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM node_description_changes`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("node deletion must cascade history: count=%d, %v", count, err)
	}
	if _, err := f.comments.Progress(ctx, f.admin, root.ID, nil, 30); apperr.From(err).Status != 404 {
		t.Fatalf("deleted node progress: %v", err)
	}
}

func TestExistingDescriptionAndConcurrentEdits(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Existing node"})
	if err != nil {
		t.Fatal(err)
	}
	// An existing description predates history support; do not invent past events.
	if _, err := f.pool.Exec(ctx, `UPDATE nodes SET description='已有描述' WHERE id=$1`, root.ID); err != nil {
		t.Fatal(err)
	}
	page, err := f.comments.Progress(ctx, f.admin, root.ID, nil, 30)
	if err != nil || len(page.Items) != 0 || page.Items == nil {
		t.Fatalf("existing descriptions must not acquire fictional history: %+v, %v", page, err)
	}
	const edits = 12
	var wg sync.WaitGroup
	errors := make(chan error, edits)
	for i := range edits {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.nodes.Update(ctx, f.admin, root.ID, nodes.UpdateRequest{Title: root.Title, Description: fmt.Sprintf("revision %d", i)})
			errors <- err
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	page, err = f.comments.Progress(ctx, f.admin, root.ID, nil, 30)
	if err != nil || len(page.Items) != edits {
		t.Fatalf("concurrent history count: %+v, %v", page, err)
	}
	previous := "已有描述"
	for i := len(page.Items) - 1; i >= 0; i-- {
		change := page.Items[i].DescriptionChange
		if change == nil || change.Before != previous {
			t.Fatalf("concurrent edits broke the revision chain at %d: %+v, want before %q", i, change, previous)
		}
		previous = change.After
	}
	current, err := f.nodes.Get(ctx, f.admin, root.ID)
	if err != nil || current.Description != previous {
		t.Fatalf("latest description disagrees with history: %+v, %v", current, err)
	}
}

func TestProgressPagination(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Paging", Description: "original"})
	if err != nil {
		t.Fatal(err)
	}
	for i := range 5 {
		if _, err := f.comments.Create(ctx, f.admin, root.ID, fmt.Sprintf("comment %d", i), nil); err != nil {
			t.Fatal(err)
		}
	}
	for i := range 3 {
		if _, err := f.nodes.Update(ctx, f.admin, root.ID, nodes.UpdateRequest{Title: root.Title, Description: fmt.Sprintf("edit %d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Other node", Description: "Not in this feed"}); err != nil {
		t.Fatal(err)
	}
	// Colliding timestamps and independent IDs must still paginate without gaps.
	for _, table := range []string{"comments", "node_description_changes"} {
		if _, err := f.pool.Exec(ctx, `UPDATE `+table+` SET created_at='2026-10-25T01:00:00Z' WHERE node_id=$1`, root.ID); err != nil {
			t.Fatal(err)
		}
	}
	var cursor *comments.ProgressCursor
	var keys []string
	for {
		page, err := f.comments.Progress(ctx, f.carol, root.ID, cursor, 2)
		if err != nil || len(page.Items) == 0 || len(page.Items) > 2 {
			t.Fatalf("page: %+v, %v", page, err)
		}
		for _, item := range page.Items {
			id := int64(0)
			if item.Kind == "comment" {
				id = item.Comment.ID
			} else {
				id = item.DescriptionChange.ID
			}
			keys = append(keys, fmt.Sprintf("%s:%d", item.Kind, id))
		}
		if !page.HasMore {
			if page.NextBefore != nil {
				t.Fatal("last page should not return a cursor")
			}
			break
		}
		cursor = page.NextBefore
		if cursor.Kind == "comment" {
			if err := f.comments.Delete(ctx, f.admin, cursor.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	want := "description_change:4,description_change:3,description_change:2,description_change:1,comment:5,comment:4,comment:3,comment:2,comment:1"
	if strings.Join(keys, ",") != want {
		t.Fatalf("timeline order = %v", keys)
	}
}

func TestProgressHTTP(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "HTTP progress", Description: "记录描述"})
	if err != nil {
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
	}{
		{"", false, 401}, {"", true, 200}, {"?limit=0", true, 400},
		{"?beforeId=1", true, 400}, {"?before=invalid&beforeKind=comment&beforeId=1", true, 400},
		{"?before=2026-10-25T01:00:00Z&beforeKind=invalid&beforeId=1", true, 400},
		{"?before=2026-10-25T01:00:00Z&beforeKind=comment&beforeId=-1", true, 400},
		{"?before=2026-10-25T01:00:00Z&beforeKind=description_change&beforeId=1", true, 200},
	} {
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/nodes/%d/progress%s", root.ID, test.query), nil)
		if test.login {
			request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: id})
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Fatalf("query %q login=%v: status=%d body=%s", test.query, test.login, response.Code, response.Body.String())
		}
	}
}
