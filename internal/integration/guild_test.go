package integration_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
	"time"

	authz "git.bitnp.net/BITNP/bitnp-authz"
	"github.com/jackc/pgx/v5/pgxpool"

	"git.bitnp.net/BITNP/guild-go/internal/access"
	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/database"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
	"git.bitnp.net/BITNP/guild-go/internal/storage"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

type fixture struct {
	pool     *pgxpool.Pool
	nodes    *nodes.Service
	comments *comments.Service
	users    *users.Service
	store    *storage.MemoryStore
	admin    auth.Session
	alice    auth.Session
	bob      auth.Session
	carol    auth.Session
	dave     auth.Session
}

func setup(t *testing.T) *fixture {
	t.Helper()
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

	userRepo := users.NewRepository(pool)
	for _, subject := range []string{"admin", "alice", "bob", "carol", "dave"} {
		if err := userRepo.Upsert(ctx, subject); err != nil {
			t.Fatal(err)
		}
	}
	userService := users.NewService(userRepo, users.NewDirectory(users.NewKeycloakAdmin("", "", "", "")))
	checker := access.New(pool)
	store := storage.NewMemoryStore()
	nodeService := nodes.NewService(nodes.NewRepository(pool), userService, checker, store)
	commentService := comments.NewService(comments.NewRepository(pool), userService, checker, store)

	return &fixture{
		pool:     pool,
		nodes:    nodeService,
		comments: commentService,
		users:    userService,
		store:    store,
		admin:    session("admin", authz.FromRoles([]string{authz.SystemAdminRole})),
		alice:    session("alice", authz.FromRoles([]string{"permission-techdept-access"})),
		bob:      session("bob", authz.FromRoles([]string{"permission-techdept-access"})),
		carol:    session("carol", authz.FromRoles([]string{"permission-techdept-access"})),
		dave:     session("dave", authz.FromRoles([]string{"permission-techdept-access"})),
	}
}

func session(subject string, permissions authz.SitePermissions) auth.Session {
	return auth.Session{Subject: subject, Name: subject, Permissions: permissions}
}

func TestReactionUsers(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Reactions"})
	if err != nil {
		t.Fatal(err)
	}
	comment, err := f.comments.Create(ctx, f.admin, root.ID, "First", nil)
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.comments.Create(ctx, f.admin, root.ID, "Second", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, reaction := range []struct {
		commentID int64
		subject   string
		emoji     string
	}{
		{comment.ID, "dave", "thumbs_up"},
		{comment.ID, "alice", "thumbs_up"},
		{comment.ID, "carol", "heart"},
		{other.ID, "bob", "thumbs_up"},
	} {
		if _, err := f.comments.ToggleReaction(ctx, reaction.commentID, reaction.subject, reaction.emoji); err != nil {
			t.Fatal(err)
		}
	}
	f.dave.Name = "Dave Display Name"
	f.dave.Picture = "https://example.com/dave.png"
	people, err := f.comments.ReactionUsers(ctx, f.dave, comment.ID, "thumbs_up")
	if err != nil {
		t.Fatal(err)
	}
	if len(people) != 2 || people[0].Subject != "dave" || people[1].Subject != "alice" {
		t.Fatalf("people for first comment's thumbs up: %+v", people)
	}
	if people[0].UserName != f.dave.Name || people[0].Avatar == nil || *people[0].Avatar != f.dave.Picture {
		t.Fatalf("current user's display data: %+v", people[0])
	}
	if people[1].UserName != "alice" || people[1].Positions == nil {
		t.Fatalf("directory fallback: %+v", people[1])
	}
	if _, err := f.comments.ToggleReaction(ctx, comment.ID, "dave", "thumbs_up"); err != nil {
		t.Fatal(err)
	}
	people, err = f.comments.ReactionUsers(ctx, f.dave, comment.ID, "thumbs_up")
	if err != nil || len(people) != 1 || people[0].Subject != "alice" {
		t.Fatalf("people after cancellation: %+v, %v", people, err)
	}
	if _, err := f.comments.ToggleReaction(ctx, comment.ID, "alice", "thumbs_up"); err != nil {
		t.Fatal(err)
	}
	people, err = f.comments.ReactionUsers(ctx, f.dave, comment.ID, "thumbs_up")
	if err != nil || people == nil || len(people) != 0 {
		t.Fatalf("empty people must be an array: %+v, %v", people, err)
	}
	if _, err := f.comments.ReactionUsers(ctx, f.dave, comment.ID, "unsupported"); apperr.From(err).Status != 400 {
		t.Fatalf("unsupported emoji: %v", err)
	}
	if err := f.comments.Delete(ctx, f.admin, comment.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.comments.ReactionUsers(ctx, f.dave, comment.ID, "thumbs_up"); apperr.From(err).Status != 404 {
		t.Fatalf("deleted comment: %v", err)
	}
}

func TestTreePermissionsCommentsAndAttachments(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{
		Title:          "Root",
		Tags:           []string{"root"},
		LeaderSubjects: []string{"alice"},
		MemberSubjects: []string{"bob"},
	})
	if err != nil {
		t.Fatal(err)
	}
	child, err := f.nodes.Create(ctx, f.alice, nodes.CreateRequest{
		ParentID:       &root.ID,
		Title:          "Child",
		LeaderSubjects: []string{"carol"},
		MemberSubjects: []string{"dave"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Alice leads the root, so she effectively leads and may comment on the child.
	childAsAlice, err := f.nodes.Get(ctx, f.alice, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !childAsAlice.Permissions.CanComment || !childAsAlice.Permissions.CanEdit {
		t.Fatalf("ancestor leader should manage child: %+v", childAsAlice.Permissions)
	}
	if childAsAlice.InheritedLeaderCount != 1 {
		t.Fatalf("inherited leaders = %d, want 1", childAsAlice.InheritedLeaderCount)
	}
	if childAsAlice.EffectiveMemberCount != 3 {
		t.Fatalf("effective members = %d, want 3 (carol, dave, alice)", childAsAlice.EffectiveMemberCount)
	}
	effectiveMembers, err := f.nodes.EffectiveMembers(ctx, f.alice, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(effectiveMembers.Leaders) != 2 {
		t.Fatalf("effective leaders = %d, want 2 (alice inherited, carol direct)", len(effectiveMembers.Leaders))
	}
	if len(effectiveMembers.Members) != 1 || effectiveMembers.Members[0].User.Subject != "dave" {
		t.Fatalf("effective members without leaders = %+v", effectiveMembers.Members)
	}

	// Bob is a root member, not a child member, and cannot comment on the child.
	childAsBob, err := f.nodes.Get(ctx, f.bob, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if childAsBob.Permissions.CanComment {
		t.Fatal("root ordinary member must not comment on a child")
	}
	if _, err := f.comments.Create(ctx, f.bob, child.ID, "nope", nil); apperr.From(err).Status != 403 {
		t.Fatalf("bob child comment status = %d, want 403", apperr.From(err).Status)
	}

	// A child member is an effective member of every ancestor.
	if _, err := f.comments.Create(ctx, f.dave, root.ID, "hello root", nil); err != nil {
		t.Fatalf("child member should comment on root: %v", err)
	}
	// The system administrator may comment without being a member.
	if _, err := f.comments.Create(ctx, f.admin, child.ID, "admin note", nil); err != nil {
		t.Fatalf("system admin comment: %v", err)
	}
	rootComment, err := f.comments.Create(ctx, f.bob, root.ID, "root discussion", nil)
	if err != nil {
		t.Fatalf("root member comment: %v", err)
	}

	// Reactions are open to every admitted user, including non-members.
	reactions, err := f.comments.ToggleReaction(ctx, rootComment.ID, "dave", "thumbs_up")
	if err != nil {
		t.Fatal(err)
	}
	if len(reactions) != 1 || reactions[0].Count != 1 || !reactions[0].ReactedByMe {
		t.Fatalf("reaction summary: %+v", reactions)
	}
	reactions, err = f.comments.ToggleReaction(ctx, rootComment.ID, "dave", "thumbs_up")
	if err != nil || reactions == nil || len(reactions) != 0 {
		t.Fatalf("reaction toggle off: %+v %v", reactions, err)
	}

	// Due date and actual completion time.
	due := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	updated, err := f.nodes.Update(ctx, f.alice, child.ID, nodes.UpdateRequest{
		Title: "Child updated", DueAt: &due, Tags: []string{"one", "two"}, Completed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.CompletedAt == nil || updated.DueAt == nil || !updated.DueAt.Equal(due) {
		t.Fatalf("completion/due not stored: %+v", updated)
	}
	reopened, err := f.nodes.Update(ctx, f.alice, child.ID, nodes.UpdateRequest{Title: "Child updated", Completed: false})
	if err != nil {
		t.Fatal(err)
	}
	if reopened.CompletedAt != nil {
		t.Fatal("reopening should clear completedAt")
	}

	// Last direct leader cannot be removed.
	if _, err := f.nodes.ReplaceMembers(ctx, f.carol, child.ID, nodes.MembersRequest{Leaders: []string{}, Members: []string{}}); apperr.From(err).Status != 400 {
		t.Fatalf("empty leader replacement status = %d, want 400", apperr.From(err).Status)
	}
	if _, err := f.nodes.ReplaceMembers(ctx, f.carol, child.ID, nodes.MembersRequest{Leaders: []string{"carol"}, Members: []string{"dave"}}); err != nil {
		t.Fatal(err)
	}

	// My relation.
	list, err := f.nodes.List(ctx, f.dave)
	if err != nil {
		t.Fatal(err)
	}
	var rootRelation, childRelation string
	for _, item := range list {
		switch item.ID {
		case root.ID:
			if item.MyRelation != nil {
				rootRelation = *item.MyRelation
			}
		case child.ID:
			if item.MyRelation != nil {
				childRelation = *item.MyRelation
			}
		}
	}
	if rootRelation != "member" || childRelation != "member" {
		t.Fatalf("dave relations root=%q child=%q", rootRelation, childRelation)
	}

	// Child ordering requires the parent leader.
	second, err := f.nodes.Create(ctx, f.alice, nodes.CreateRequest{ParentID: &root.ID, Title: "Second", LeaderSubjects: []string{"alice"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.nodes.Reorder(ctx, f.bob, &root.ID, []int64{second.ID, child.ID}); apperr.From(err).Status != 403 {
		t.Fatalf("bob reorder status = %d, want 403", apperr.From(err).Status)
	}
	if err := f.nodes.Reorder(ctx, f.alice, &root.ID, []int64{second.ID, child.ID}); err != nil {
		t.Fatal(err)
	}
	rootAsAlice, err := f.nodes.Get(ctx, f.alice, root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rootAsAlice.Children) != 2 || rootAsAlice.Children[0].ID != second.ID || rootAsAlice.Children[1].ID != child.ID {
		t.Fatalf("child order not applied: %+v", rootAsAlice.Children)
	}

	// Attachment upload and authenticated read.
	imageBytes := testPNG(t)
	imageID, err := f.comments.UploadImage(ctx, f.alice, child.ID, imageBytes)
	if err != nil {
		t.Fatal(err)
	}
	withImage, err := f.comments.Create(ctx, f.alice, child.ID, "see image", []int64{imageID})
	if err != nil {
		t.Fatal(err)
	}
	if len(withImage.Attachments) != 1 {
		t.Fatalf("attachments = %d, want 1", len(withImage.Attachments))
	}
	reader, contentType, size, err := f.comments.GetAttachment(ctx, withImage.Attachments[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	got := new(bytes.Buffer)
	_, _ = got.ReadFrom(reader)
	_ = reader.Close()
	if contentType != "image/png" || size != int64(len(got.Bytes())) || !bytes.Equal(got.Bytes(), imageBytes) {
		t.Fatalf("attachment mismatch: content=%s size=%d len=%d", contentType, size, got.Len())
	}

	// Overview includes roots and recent comments.
	overview, err := f.nodes.Overview(ctx, f.alice)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.Roots) != 1 || overview.Roots[0].ID != root.ID || len(overview.RecentComments) == 0 {
		t.Fatalf("overview: %+v", overview)
	}

	// Deleting the child cascades its comments and attachments.
	keysBefore := len(f.store.Keys())
	if keysBefore == 0 {
		t.Fatal("expected an uploaded object before deletion")
	}
	if err := f.nodes.Delete(ctx, f.alice, child.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.nodes.Get(ctx, f.alice, child.ID); apperr.From(err).Status != 404 {
		t.Fatalf("deleted child status = %d, want 404", apperr.From(err).Status)
	}
	if len(f.store.Keys()) != 0 {
		t.Fatalf("attachment objects remain after cascade delete: %v", f.store.Keys())
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
