// Package access centralizes the recursive node-tree authorization queries.
// It contains no HTTP or display logic.
package access

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Checker evaluates effective membership and leadership in the node tree.
type Checker struct {
	db queryer
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// New creates a tree access checker.
func New(pool *pgxpool.Pool) *Checker {
	return &Checker{db: pool}
}

// InTx evaluates permissions against the same transaction as a tree change.
func InTx(tx pgx.Tx) *Checker {
	return &Checker{db: tx}
}

// NodeExists reports whether a node exists.
func (c *Checker) NodeExists(ctx context.Context, nodeID int64) (bool, error) {
	var exists bool
	err := c.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM nodes WHERE id=$1)`, nodeID).Scan(&exists)
	return exists, err
}

// EffectiveLeaders returns direct leaders of the node and all of its ancestors.
func (c *Checker) EffectiveLeaders(ctx context.Context, nodeID int64) ([]string, error) {
	const query = `
WITH RECURSIVE ancestors AS (
    SELECT id, parent_id FROM nodes WHERE id = $1
    UNION ALL
    SELECT n.id, n.parent_id FROM nodes n JOIN ancestors a ON n.id = a.parent_id
)
SELECT DISTINCT m.subject
FROM node_members m
JOIN ancestors a ON a.id = m.node_id
WHERE m.role = 'leader'
ORDER BY m.subject`
	return c.querySubjects(ctx, query, nodeID)
}

// EffectiveMembers returns node/descendant members (leaders included) plus all
// ancestor leaders.
func (c *Checker) EffectiveMembers(ctx context.Context, nodeID int64) ([]string, error) {
	const query = `
WITH RECURSIVE subtree AS (
    SELECT id FROM nodes WHERE id = $1
    UNION ALL
    SELECT n.id FROM nodes n JOIN subtree s ON n.parent_id = s.id
),
ancestors AS (
    SELECT id, parent_id FROM nodes WHERE id = $1
    UNION ALL
    SELECT n.id, n.parent_id FROM nodes n JOIN ancestors a ON n.id = a.parent_id
)
SELECT DISTINCT subject FROM (
    SELECT m.subject
    FROM node_members m
    JOIN subtree s ON s.id = m.node_id
    UNION
    SELECT m.subject
    FROM node_members m
    JOIN ancestors a ON a.id = m.node_id
    WHERE m.role = 'leader'
) members
ORDER BY subject`
	return c.querySubjects(ctx, query, nodeID)
}

// IsEffectiveLeader reports whether subject leads the node or an ancestor.
func (c *Checker) IsEffectiveLeader(ctx context.Context, nodeID int64, subject string) (bool, error) {
	const query = `
WITH RECURSIVE ancestors AS (
    SELECT id, parent_id FROM nodes WHERE id = $1
    UNION ALL
    SELECT n.id, n.parent_id FROM nodes n JOIN ancestors a ON n.id = a.parent_id
)
SELECT EXISTS (
    SELECT 1 FROM node_members m
    JOIN ancestors a ON a.id = m.node_id
    WHERE m.subject = $2 AND m.role = 'leader'
)`
	var ok bool
	err := c.db.QueryRow(ctx, query, nodeID, subject).Scan(&ok)
	return ok, err
}

// IsEffectiveMember reports whether subject may comment on the node.
func (c *Checker) IsEffectiveMember(ctx context.Context, nodeID int64, subject string) (bool, error) {
	const query = `
WITH RECURSIVE subtree AS (
    SELECT id FROM nodes WHERE id = $1
    UNION ALL
    SELECT n.id FROM nodes n JOIN subtree s ON n.parent_id = s.id
),
ancestors AS (
    SELECT id, parent_id FROM nodes WHERE id = $1
    UNION ALL
    SELECT n.id, n.parent_id FROM nodes n JOIN ancestors a ON n.id = a.parent_id
)
SELECT EXISTS (
    SELECT 1 FROM node_members m JOIN subtree s ON s.id = m.node_id WHERE m.subject = $2
) OR EXISTS (
    SELECT 1 FROM node_members m JOIN ancestors a ON a.id = m.node_id
    WHERE m.subject = $2 AND m.role = 'leader'
)`
	var ok bool
	err := c.db.QueryRow(ctx, query, nodeID, subject).Scan(&ok)
	return ok, err
}

// Relations returns the nodes the subject effectively leads and participates in.
func (c *Checker) Relations(ctx context.Context, subject string) (leaders map[int64]struct{}, members map[int64]struct{}, err error) {
	const query = `
WITH RECURSIVE direct_leader_nodes AS (
    SELECT node_id AS id FROM node_members WHERE subject = $1 AND role = 'leader'
),
managed AS (
    SELECT id FROM direct_leader_nodes
    UNION
    SELECT n.id FROM nodes n JOIN managed m ON n.parent_id = m.id
),
direct_member_nodes AS (
    SELECT node_id AS id FROM node_members WHERE subject = $1
),
participated AS (
    SELECT id FROM direct_member_nodes
    UNION
    SELECT n.parent_id FROM nodes n JOIN participated p ON n.id = p.id
    WHERE n.parent_id IS NOT NULL
)
SELECT id, true FROM managed
UNION ALL
SELECT id, false FROM participated`
	leaders = make(map[int64]struct{})
	members = make(map[int64]struct{})
	rows, err := c.db.Query(ctx, query, subject)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var isLeader bool
		if err := rows.Scan(&id, &isLeader); err != nil {
			return nil, nil, err
		}
		members[id] = struct{}{}
		if isLeader {
			leaders[id] = struct{}{}
		}
	}
	return leaders, members, rows.Err()
}

// MemberDetail describes one source of an effective membership.
type MemberDetail struct {
	Subject   string
	Role      string
	NodeID    int64
	NodeTitle string
}

// EffectiveMemberDetails returns every direct/ancestor-leader membership row
// contributing to a node's effective membership.
func (c *Checker) EffectiveMemberDetails(ctx context.Context, nodeID int64) ([]MemberDetail, error) {
	const query = `
WITH RECURSIVE subtree AS (
    SELECT id FROM nodes WHERE id = $1
    UNION ALL
    SELECT n.id FROM nodes n JOIN subtree s ON n.parent_id = s.id
),
ancestors AS (
    SELECT id, parent_id FROM nodes WHERE id = $1
    UNION ALL
    SELECT n.id, n.parent_id FROM nodes n JOIN ancestors a ON n.id = a.parent_id
)
SELECT m.subject, m.role, m.node_id, n.title
FROM node_members m
JOIN subtree s ON s.id = m.node_id
JOIN nodes n ON n.id = m.node_id
UNION
SELECT m.subject, m.role, m.node_id, n.title
FROM node_members m
JOIN ancestors a ON a.id = m.node_id
JOIN nodes n ON n.id = m.node_id
WHERE m.role = 'leader'
ORDER BY 1, 3`
	rows, err := c.db.Query(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []MemberDetail
	for rows.Next() {
		var detail MemberDetail
		if err := rows.Scan(&detail.Subject, &detail.Role, &detail.NodeID, &detail.NodeTitle); err != nil {
			return nil, err
		}
		result = append(result, detail)
	}
	return result, rows.Err()
}

func (c *Checker) querySubjects(ctx context.Context, query string, nodeID int64) ([]string, error) {
	rows, err := c.db.Query(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subjects []string
	for rows.Next() {
		var subject string
		if err := rows.Scan(&subject); err != nil {
			return nil, err
		}
		subjects = append(subjects, subject)
	}
	return subjects, rows.Err()
}
