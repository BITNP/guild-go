package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	authz "git.bitnp.net/BITNP/bitnp-authz"
	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/httpapi"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
)

func createMoveNode(t *testing.T, f *fixture, parent *int64, title, leader string, members ...string) nodes.Detail {
	t.Helper()
	node, err := f.nodes.Create(context.Background(), f.admin, nodes.CreateRequest{
		ParentID: parent, Title: title, Description: "Keep description", Tags: []string{"keep"},
		LeaderSubjects: []string{leader}, MemberSubjects: members,
	})
	if err != nil {
		t.Fatal(err)
	}
	return node
}

func TestMoveSubtreePreservesContentAndRecalculatesAccess(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	source := createMoveNode(t, f, nil, "Source", "alice")
	target := createMoveNode(t, f, nil, "Target", "carol")
	child := createMoveNode(t, f, &source.ID, "Moving", "dave", "bob")
	remaining := createMoveNode(t, f, &source.ID, "Staying", "alice")
	leaf := createMoveNode(t, f, &child.ID, "Leaf", "dave")
	first := createMoveNode(t, f, &target.ID, "First", "carol")
	second := createMoveNode(t, f, &target.ID, "Second", "carol")
	imageID, err := f.comments.UploadImage(ctx, f.bob, child.ID, testPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	comment, err := f.comments.Create(ctx, f.bob, child.ID, "Keep comment", []int64{imageID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.comments.ToggleReaction(ctx, comment.ID, "bob", "heart"); err != nil {
		t.Fatal(err)
	}
	assignmentTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := f.pool.Exec(ctx, `UPDATE node_members SET created_at=$2 WHERE node_id=$1`, child.ID, assignmentTime); err != nil {
		t.Fatal(err)
	}
	// A system administrator can move between independently managed roots.
	moved, err := f.nodes.Move(ctx, f.admin, child.ID, nodes.MoveRequest{ParentID: &target.ID})
	if err != nil {
		t.Fatal(err)
	}
	if moved.ParentID == nil || *moved.ParentID != target.ID || moved.SortOrder != second.SortOrder+1 {
		t.Fatalf("wrong destination or ordering: %+v", moved.Node)
	}
	if moved.Description != child.Description || !slices.Equal(moved.Tags, child.Tags) || !moved.CreatedAt.Equal(child.CreatedAt) {
		t.Fatal("moving changed content or creation time")
	}
	assertDirectMembers(t, moved, "bob")
	if len(moved.DirectLeaders) != 1 || moved.DirectLeaders[0].Subject != "dave" {
		t.Fatalf("direct leaders changed: %+v", moved.DirectLeaders)
	}
	if len(moved.Breadcrumb) != 1 || moved.Breadcrumb[0].ID != target.ID || moved.SubtreeNodeCount != 2 {
		t.Fatalf("subtree or breadcrumb changed incorrectly: %+v", moved)
	}
	leafAfter, err := f.nodes.Get(ctx, f.admin, leaf.ID)
	if err != nil {
		t.Fatal(err)
	}
	if leafAfter.ParentID == nil || *leafAfter.ParentID != child.ID || len(leafAfter.Breadcrumb) != 2 || leafAfter.Breadcrumb[0].ID != target.ID {
		t.Fatalf("descendant did not follow subtree: %+v", leafAfter)
	}
	for _, id := range []int64{child.ID, leaf.ID} {
		oldLeader, err := f.nodes.Get(ctx, f.alice, id)
		if err != nil {
			t.Fatal(err)
		}
		newLeader, err := f.nodes.Get(ctx, f.carol, id)
		if err != nil {
			t.Fatal(err)
		}
		if oldLeader.Permissions.CanEdit || oldLeader.Permissions.CanComment || !newLeader.Permissions.CanEdit || !newLeader.Permissions.CanComment {
			t.Fatalf("inherited permissions did not follow the move: old=%+v new=%+v", oldLeader.Permissions, newLeader.Permissions)
		}
	}
	oldRoot, err := f.nodes.Get(ctx, f.bob, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	newRoot, err := f.nodes.Get(ctx, f.bob, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if oldRoot.Permissions.CanComment || !newRoot.Permissions.CanComment {
		t.Fatal("upward member access did not follow subtree")
	}
	if len(oldRoot.Children) != 1 || oldRoot.Children[0].ID != remaining.ID || len(newRoot.Children) != 3 || newRoot.Children[0].ID != first.ID || newRoot.Children[2].ID != child.ID {
		t.Fatal("source or destination children are incorrect")
	}
	if !oldRoot.UpdatedAt.After(source.UpdatedAt) || !newRoot.UpdatedAt.After(target.UpdatedAt) {
		t.Fatal("parent timestamps were not refreshed")
	}
	var preserved bool
	if err := f.pool.QueryRow(ctx, `SELECT bool_and(created_at=$2) FROM node_members WHERE node_id=$1`, child.ID, assignmentTime).Scan(&preserved); err != nil || !preserved {
		t.Fatalf("membership assignment times changed: %v", err)
	}
	comments, err := f.comments.List(ctx, f.carol, child.ID, nil, 30)
	if err != nil || len(comments.Items) != 1 || comments.Items[0].ID != comment.ID || len(comments.Items[0].Attachments) != 1 || len(comments.Items[0].Reactions) != 1 || len(f.store.Keys()) != 1 {
		t.Fatalf("comment, reaction or attachment changed: %+v, %v", comments, err)
	}
	var changes int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM node_description_changes WHERE node_id=$1`, child.ID).Scan(&changes); err != nil || changes != 1 {
		t.Fatalf("description history changed: %d, %v", changes, err)
	}
	// Cycle validation still applies when the caller leads both nodes.
	moved, err = f.nodes.Move(ctx, f.dave, child.ID, nodes.MoveRequest{ParentID: &leaf.ID})
	if apperr.From(err).Status != 400 {
		t.Fatalf("descendant move should fail: %+v, %v", moved, err)
	}
}

func TestMoveValidationPermissionsAndRootPromotion(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	root := createMoveNode(t, f, nil, "Root", "alice")
	other := createMoveNode(t, f, nil, "Other", "carol")
	child := createMoveNode(t, f, &root.ID, "Child", "dave")
	target := createMoveNode(t, f, &root.ID, "Target", "alice")
	missing, invalid := int64(999999), int64(0)
	for _, tc := range []struct {
		name    string
		session auth.Session
		id      int64
		parent  *int64
		status  int
	}{
		{"self", f.admin, child.ID, &child.ID, 400},
		{"descendant", f.admin, root.ID, &child.ID, 400},
		{"missing source", f.admin, missing, &root.ID, 404},
		{"missing target", f.admin, child.ID, &missing, 404},
		{"invalid target", f.admin, child.ID, &invalid, 400},
		{"non-leader", f.bob, child.ID, &target.ID, 403},
		{"target only leader", f.carol, child.ID, &other.ID, 403},
		{"source only leader", f.dave, child.ID, &target.ID, 403},
		{"unmanaged destination", f.alice, child.ID, &other.ID, 403},
		{"non-manager root promotion", f.alice, child.ID, nil, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.nodes.Move(ctx, tc.session, tc.id, nodes.MoveRequest{ParentID: tc.parent})
			if apperr.From(err).Status != tc.status {
				t.Fatalf("status=%d, want %d: %v", apperr.From(err).Status, tc.status, err)
			}
			unchanged, err := f.nodes.Get(ctx, f.admin, child.ID)
			if err != nil || unchanged.ParentID == nil || *unchanged.ParentID != root.ID {
				t.Fatalf("rejected move changed source: %+v, %v", unchanged, err)
			}
		})
	}
	unchanged, err := f.nodes.Move(ctx, f.dave, child.ID, nodes.MoveRequest{ParentID: &root.ID})
	if err != nil || unchanged.SortOrder != child.SortOrder || !unchanged.UpdatedAt.Equal(child.UpdatedAt) {
		t.Fatalf("same parent should be a no-op: %+v, %v", unchanged, err)
	}
	moved, err := f.nodes.Move(ctx, f.alice, child.ID, nodes.MoveRequest{ParentID: &target.ID})
	if err != nil || moved.ParentID == nil || *moved.ParentID != target.ID {
		t.Fatalf("common ancestor leader should be allowed: %+v, %v", moved, err)
	}
	manager := session("dave", authz.FromRoles([]string{"permission-techdept-admin"}))
	if !manager.Permissions.Manager {
		t.Fatal("fixture must be a manager")
	}
	moved, err = f.nodes.Move(ctx, manager, child.ID, nodes.MoveRequest{})
	if err != nil || moved.ParentID != nil || moved.SortOrder != other.SortOrder+1 || len(moved.Breadcrumb) != 0 {
		t.Fatalf("manager root promotion failed: %+v, %v", moved, err)
	}
	// Roots can also become children of another root.
	moved, err = f.nodes.Move(ctx, f.admin, child.ID, nodes.MoveRequest{ParentID: &other.ID})
	if err != nil || moved.ParentID == nil || *moved.ParentID != other.ID {
		t.Fatalf("moving root into tree failed: %+v, %v", moved, err)
	}
}

func TestConcurrentMovesCannotCreateCycle(t *testing.T) {
	f := setup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a := createMoveNode(t, f, nil, "A", "alice")
	b := createMoveNode(t, f, nil, "B", "bob")
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, pair := range [][2]int64{{a.ID, b.ID}, {b.ID, a.ID}} {
		go func() {
			<-start
			_, err := f.nodes.Move(ctx, f.admin, pair[0], nodes.MoveRequest{ParentID: &pair[1]})
			results <- err
		}()
	}
	close(start)
	succeeded, rejected := 0, 0
	for range 2 {
		if err := <-results; err == nil {
			succeeded++
		} else if apperr.From(err).Status == 400 {
			rejected++
		} else {
			t.Fatalf("unexpected concurrent move failure: %v", err)
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("moves: %d succeeded, %d rejected", succeeded, rejected)
	}
}

func TestQueuedMoveRechecksInheritedLeadership(t *testing.T) {
	f := setup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	root := createMoveNode(t, f, nil, "Root", "alice")
	other := createMoveNode(t, f, nil, "Other", "carol")
	child := createMoveNode(t, f, &root.ID, "Child", "dave")
	target := createMoveNode(t, f, &root.ID, "Target", "alice")
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `LOCK TABLE nodes IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := f.nodes.Move(ctx, f.alice, child.ID, nodes.MoveRequest{ParentID: &target.ID})
		result <- err
	}()
	waitForBlockedOperation(t, ctx, f.pool, tx.Conn().PgConn().PID(), result)
	// Simulate a preceding move removing Alice's inherited authority.
	if _, err := tx.Exec(ctx, `UPDATE nodes SET parent_id=$2 WHERE id=$1`, child.ID, other.ID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-result; apperr.From(err).Status != 403 {
		t.Fatalf("queued move must lose permission after ancestry changes: %v", err)
	}
	after, err := f.nodes.Get(ctx, f.admin, child.ID)
	if err != nil || after.ParentID == nil || *after.ParentID != other.ID {
		t.Fatalf("rejected move changed node: %+v, %v", after, err)
	}
}

func TestMoveHTTPRequiresExplicitDestinationAndCSRF(t *testing.T) {
	f := setup(t)
	root := createMoveNode(t, f, nil, "Root", "alice")
	child := createMoveNode(t, f, &root.ID, "Child", "alice")
	authService := auth.NewService(nil, auth.NewStore(time.Minute), auth.NewPendingStore(), false, nil)
	if err := authService.EnableDevAuthBypass("admin", "Admin"); err != nil {
		t.Fatal(err)
	}
	handler := httpapi.New(f.pool, authService, f.users, f.nodes, f.comments).Handler()
	csrfResponse := httptest.NewRecorder()
	handler.ServeHTTP(csrfResponse, httptest.NewRequest(http.MethodGet, "/api/auth/csrf", nil))
	var csrf struct{ Token string }
	if err := json.Unmarshal(csrfResponse.Body.Bytes(), &csrf); err != nil || csrf.Token == "" {
		t.Fatalf("csrf response: %s, %v", csrfResponse.Body, err)
	}
	path := fmt.Sprintf("/api/nodes/%d/move", child.ID)
	for _, tc := range []struct {
		body  string
		token string
		want  int
	}{
		{`{"parentId":null}`, "", 403},
		{`{}`, csrf.Token, 400},
		{`null`, csrf.Token, 400},
		{`{"parentId":"root"}`, csrf.Token, 400},
		{`{"parentId":0}`, csrf.Token, 400},
		{`{"parentId":null}`, csrf.Token, 200},
	} {
		request := httptest.NewRequest(http.MethodPut, path, strings.NewReader(tc.body))
		request.Header.Set("X-CSRF-Token", tc.token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != tc.want {
			t.Fatalf("body=%s status=%d want=%d: %s", tc.body, response.Code, tc.want, response.Body)
		}
		if response.Code == 200 {
			var moved nodes.Detail
			if err := json.Unmarshal(response.Body.Bytes(), &moved); err != nil || moved.ParentID != nil {
				t.Fatalf("root move response: %+v, %v", moved, err)
			}
		}
	}
}
