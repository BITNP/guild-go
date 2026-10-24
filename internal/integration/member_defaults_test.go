package integration_test

import (
	"context"
	"slices"
	"testing"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
)

func TestDirectMemberDefaults(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{
		Title: "Root", LeaderSubjects: []string{"alice"}, MemberSubjects: []string{"bob"},
	})
	if err != nil {
		t.Fatal(err)
	}
	parent, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{
		ParentID: &root.ID, Title: "Parent", LeaderSubjects: []string{"carol"}, MemberSubjects: []string{"admin", "dave"},
	})
	if err != nil {
		t.Fatal(err)
	}

	child, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{ParentID: &parent.ID, Title: "Default members"})
	if err != nil {
		t.Fatal(err)
	}
	assertDirectMembers(t, child, "dave")
	if len(child.DirectLeaders) != 1 || child.DirectLeaders[0].Subject != "admin" {
		t.Fatalf("default leader: %+v", child.DirectLeaders)
	}
	explicit, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{
		ParentID: &parent.ID, Title: "Explicit members", MemberSubjects: []string{"bob"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertDirectMembers(t, explicit, "bob")

	// Choosing a parent member as leader must not also add them as a member.
	child, err = f.nodes.ReplaceMembers(ctx, f.admin, child.ID, nodes.MembersRequest{Leaders: []string{"dave"}})
	if err != nil {
		t.Fatal(err)
	}
	assertDirectMembers(t, child, "admin")
	if _, err := f.nodes.ReplaceMembers(ctx, f.admin, child.ID, nodes.MembersRequest{
		Leaders: []string{"dave"}, Members: []string{"dave"},
	}); apperr.From(err).Status != 400 {
		t.Fatalf("explicit overlapping roles must still fail: %v", err)
	}
	if _, err := f.nodes.ReplaceMembers(ctx, f.bob, child.ID, nodes.MembersRequest{Leaders: []string{"bob"}}); apperr.From(err).Status != 403 {
		t.Fatalf("non-leader must not replace members: %v", err)
	}

	if _, err := f.nodes.ReplaceMembers(ctx, f.admin, parent.ID, nodes.MembersRequest{
		Leaders: []string{"carol"}, Members: []string{"bob"},
	}); err != nil {
		t.Fatal(err)
	}
	child, err = f.nodes.Get(ctx, f.admin, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertDirectMembers(t, child, "admin")
	child, err = f.nodes.ReplaceMembers(ctx, f.admin, child.ID, nodes.MembersRequest{Leaders: []string{"dave"}, Members: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	assertDirectMembers(t, child, "bob")

	// Roots have no parent to copy; an empty immediate parent does not fall back
	// to a more distant ancestor's members.
	emptyRoot, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Empty root"})
	if err != nil {
		t.Fatal(err)
	}
	assertDirectMembers(t, emptyRoot)
	root, err = f.nodes.ReplaceMembers(ctx, f.admin, root.ID, nodes.MembersRequest{Leaders: []string{"alice"}})
	if err != nil {
		t.Fatal(err)
	}
	assertDirectMembers(t, root)
	emptyParent, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{
		ParentID: &parent.ID, Title: "All parent members are leaders", LeaderSubjects: []string{"bob"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertDirectMembers(t, emptyParent)
	leaf, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{ParentID: &emptyParent.ID, Title: "No grandparent fallback"})
	if err != nil {
		t.Fatal(err)
	}
	assertDirectMembers(t, leaf)
}

func assertDirectMembers(t *testing.T, node nodes.Detail, want ...string) {
	t.Helper()
	got := make([]string, 0, len(node.DirectMembers))
	for _, member := range node.DirectMembers {
		got = append(got, member.Subject)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("node %d direct members = %v, want %v", node.ID, got, want)
	}
}
