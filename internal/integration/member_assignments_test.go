package integration_test

import (
	"context"
	"testing"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/nodes"
)

func TestRoleChangeRefreshesOnlyChangedAssignmentTime(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	node, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{
		Title: "Role changes", LeaderSubjects: []string{"alice"}, MemberSubjects: []string{"bob"},
	})
	if err != nil {
		t.Fatal(err)
	}
	old := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := f.pool.Exec(ctx, `UPDATE node_members SET created_at=$2 WHERE node_id=$1`, node.ID, old); err != nil {
		t.Fatal(err)
	}
	if _, err := f.nodes.ReplaceMembers(ctx, f.admin, node.ID, nodes.MembersRequest{
		Leaders: []string{"alice", "bob"},
	}); err != nil {
		t.Fatal(err)
	}
	var alice, bob time.Time
	if err := f.pool.QueryRow(ctx, `SELECT created_at FROM node_members WHERE node_id=$1 AND subject='alice'`, node.ID).Scan(&alice); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT created_at FROM node_members WHERE node_id=$1 AND subject='bob'`, node.ID).Scan(&bob); err != nil {
		t.Fatal(err)
	}
	if !alice.Equal(old) {
		t.Fatalf("unchanged leader got a new assignment time: %v", alice)
	}
	if !bob.After(old) {
		t.Fatalf("promoted member kept old assignment time: %v", bob)
	}
	if _, err := f.nodes.ReplaceMembers(ctx, f.admin, node.ID, nodes.MembersRequest{
		Leaders: []string{"alice", "bob"},
	}); err != nil {
		t.Fatal(err)
	}
	var saved time.Time
	if err := f.pool.QueryRow(ctx, `SELECT created_at FROM node_members WHERE node_id=$1 AND subject='bob'`, node.ID).Scan(&saved); err != nil {
		t.Fatal(err)
	}
	if !saved.Equal(bob) {
		t.Fatalf("saving unchanged roles reassigns members: %v -> %v", bob, saved)
	}
}
