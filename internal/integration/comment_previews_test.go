package integration_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
)

func TestCommentPreviewsAcrossNodeViews(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	f.alice.Name = "Alice Display"
	root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Root", LeaderSubjects: []string{"alice"}})
	if err != nil {
		t.Fatal(err)
	}
	child, err := f.nodes.Create(ctx, f.alice, nodes.CreateRequest{ParentID: &root.ID, Title: "Child"})
	if err != nil {
		t.Fatal(err)
	}
	empty, err := f.nodes.Create(ctx, f.alice, nodes.CreateRequest{ParentID: &root.ID, Title: "Empty"})
	if err != nil {
		t.Fatal(err)
	}
	older, err := f.comments.Create(ctx, f.admin, root.ID, "Older root progress", nil)
	if err != nil {
		t.Fatal(err)
	}
	latest, err := f.comments.Create(ctx, f.alice, root.ID, strings.Repeat("新", 210), nil)
	if err != nil {
		t.Fatal(err)
	}
	imageID, err := f.comments.UploadImage(ctx, f.alice, child.ID, testPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	childComment, err := f.comments.Create(ctx, f.alice, child.ID, "Child progress", []int64{imageID})
	if err != nil {
		t.Fatal(err)
	}
	for _, reaction := range []struct{ subject, emoji string }{
		{"carol", "eyes"}, {"alice", "heart"}, {"bob", "thumbs_up"},
	} {
		if _, err := f.comments.ToggleReaction(ctx, latest.ID, reaction.subject, reaction.emoji); err != nil {
			t.Fatal(err)
		}
	}
	wantReactions := []comments.Reaction{
		{Emoji: "thumbs_up", Count: 1}, {Emoji: "heart", Count: 1, ReactedByMe: true}, {Emoji: "eyes", Count: 1},
	}

	detail, err := f.nodes.Get(ctx, f.alice, root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Children) != 2 || detail.Children[0].ID != child.ID || detail.Children[1].ID != empty.ID {
		t.Fatalf("children: %+v", detail.Children)
	}
	preview := detail.Children[0].LatestComment
	if preview == nil || preview.ID != childComment.ID || preview.AttachmentCount != 1 || preview.Author.UserName != f.alice.Name || preview.ReactionSummary == nil {
		t.Fatalf("child preview: %+v", preview)
	}
	if detail.Children[0].LatestCommentAt == nil || !detail.Children[0].LatestCommentAt.Equal(childComment.CreatedAt) || detail.Children[1].LatestComment != nil || detail.Children[1].LatestCommentAt != nil {
		t.Fatalf("latest comment timestamps: %+v", detail.Children)
	}

	overview, err := f.nodes.Overview(ctx, f.alice)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.Roots) != 1 || len(overview.RecentComments) != 3 {
		t.Fatalf("overview: %+v", overview)
	}
	if overview.RecentComments[0].ID != childComment.ID || overview.RecentComments[1].ID != latest.ID || overview.RecentComments[2].ID != older.ID {
		t.Fatalf("recent comment order: %+v", overview.RecentComments)
	}
	preview = overview.Roots[0].LatestComment
	if preview == nil || preview.ID != latest.ID || preview.BodyPreview != strings.Repeat("新", 200)+"…" || preview.Author.UserName != f.alice.Name || !reflect.DeepEqual(preview.ReactionSummary, wantReactions) {
		t.Fatalf("root preview: %+v", preview)
	}
	item := overview.RecentComments[1]
	if item.NodeTitle != root.Title || item.RootTitle != root.Title || item.Author.UserName != f.alice.Name || item.BodyPreview != strings.Repeat("新", 120)+"…" || !reflect.DeepEqual(item.ReactionSummary, wantReactions) {
		t.Fatalf("recent root preview: %+v", item)
	}
	item = overview.RecentComments[0]
	if item.NodeTitle != child.Title || item.RootTitle != root.Title || item.ReactionSummary == nil {
		t.Fatalf("recent child preview: %+v", item)
	}
	page, err := f.comments.List(ctx, f.alice, root.ID, nil, 30)
	if err != nil || len(page.Items) != 2 || !reflect.DeepEqual(page.Items[0].Reactions, wantReactions) {
		t.Fatalf("full comments must agree with previews: %+v, %v", page, err)
	}

	if err := f.comments.Delete(ctx, f.alice, latest.ID); err != nil {
		t.Fatal(err)
	}
	overview, err = f.nodes.Overview(ctx, f.alice)
	if err != nil {
		t.Fatal(err)
	}
	preview = overview.Roots[0].LatestComment
	if preview == nil || preview.ID != older.ID || len(preview.ReactionSummary) != 0 || preview.ReactionSummary == nil {
		t.Fatalf("deleted latest comment must reveal its predecessor: %+v", preview)
	}
}
