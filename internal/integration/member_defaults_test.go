package integration_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
)

func TestDirectMembersMayBeEmpty(t *testing.T) {
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

	for _, tc := range []struct {
		name    string
		members []string
		want    []string
	}{
		{name: "omitted"},
		{name: "empty", members: []string{}},
		{name: "blank subjects", members: []string{"", "  "}},
		{name: "explicit", members: []string{"bob"}, want: []string{"bob"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			child, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{
				ParentID: &parent.ID, Title: tc.name, MemberSubjects: tc.members,
			})
			if err != nil {
				t.Fatal(err)
			}
			assertDirectMembers(t, child, tc.want...)
			if len(child.DirectLeaders) != 1 || child.DirectLeaders[0].Subject != "admin" {
				t.Fatalf("default leader: %+v", child.DirectLeaders)
			}

			// Seed an existing member, then verify replacement can clear the list.
			if _, err := f.nodes.ReplaceMembers(ctx, f.admin, child.ID, nodes.MembersRequest{
				Leaders: []string{"admin"}, Members: []string{"dave"},
			}); err != nil {
				t.Fatal(err)
			}
			child, err = f.nodes.ReplaceMembers(ctx, f.admin, child.ID, nodes.MembersRequest{
				Leaders: []string{"admin"}, Members: tc.members,
			})
			if err != nil {
				t.Fatal(err)
			}
			assertDirectMembers(t, child, tc.want...)
			child, err = f.nodes.Get(ctx, f.admin, child.ID)
			if err != nil {
				t.Fatal(err)
			}
			assertDirectMembers(t, child, tc.want...)

			// Saving again must not repopulate an empty list from the parent.
			child, err = f.nodes.ReplaceMembers(ctx, f.admin, child.ID, nodes.MembersRequest{
				Leaders: []string{"admin"}, Members: tc.members,
			})
			if err != nil {
				t.Fatal(err)
			}
			assertDirectMembers(t, child, tc.want...)
			viewer, err := f.nodes.Get(ctx, f.dave, child.ID)
			if err != nil {
				t.Fatal(err)
			}
			if viewer.Permissions.CanComment {
				t.Fatal("parent member must not gain comment access to child")
			}
			viewer, err = f.nodes.Get(ctx, f.alice, child.ID)
			if err != nil {
				t.Fatal(err)
			}
			if !viewer.Permissions.CanComment {
				t.Fatal("ancestor leader must retain comment access")
			}
		})
	}

	if _, err := f.nodes.ReplaceMembers(ctx, f.admin, parent.ID, nodes.MembersRequest{
		Leaders: []string{"dave"}, Members: []string{"dave"},
	}); apperr.From(err).Status != 400 {
		t.Fatalf("overlapping roles must still fail: %v", err)
	}
	if _, err := f.nodes.ReplaceMembers(ctx, f.admin, parent.ID, nodes.MembersRequest{}); apperr.From(err).Status != 400 {
		t.Fatalf("empty leaders must still fail: %v", err)
	}
	if _, err := f.nodes.ReplaceMembers(ctx, f.bob, parent.ID, nodes.MembersRequest{Leaders: []string{"bob"}}); apperr.From(err).Status != 403 {
		t.Fatalf("non-leader must not replace members: %v", err)
	}

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

// Saving the same membership set must preserve the original assignment time.
func TestMembershipSavePreservesUnchangedRows(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	node, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{
		Title: "Stable membership", LeaderSubjects: []string{"alice"}, MemberSubjects: []string{"bob"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var before, after time.Time
	if err := f.pool.QueryRow(ctx, `SELECT created_at FROM node_members WHERE node_id=$1 AND subject='alice'`, node.ID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, err := f.nodes.ReplaceMembers(ctx, f.admin, node.ID, nodes.MembersRequest{
		Leaders: []string{"alice"}, Members: []string{"carol"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT created_at FROM node_members WHERE node_id=$1 AND subject='alice'`, node.ID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if !before.Equal(after) {
		t.Fatalf("unchanged leader was reassigned: %v -> %v", before, after)
	}
	var removed bool
	if err := f.pool.QueryRow(ctx, `SELECT NOT EXISTS (SELECT 1 FROM node_members WHERE node_id=$1 AND subject='bob')`, node.ID).Scan(&removed); err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("removed member still exists")
	}
}
