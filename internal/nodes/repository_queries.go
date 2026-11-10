package nodes

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
)

type nodeScan struct {
	node      Node
	parent    pgtype.Int8
	due       pgtype.Timestamptz
	completed pgtype.Timestamptz
	tags      []string
}

func newScan() *nodeScan {
	return &nodeScan{}
}

func (s *nodeScan) dest() []any {
	return []any{
		&s.node.ID, &s.parent, &s.node.Title, &s.node.Description,
		&s.due, &s.completed, &s.tags, &s.node.SortOrder,
		&s.node.CreatedBy, &s.node.CreatedAt, &s.node.UpdatedAt,
	}
}

func (s *nodeScan) value() Node {
	s.node.ParentID = int64Ptr(s.parent)
	s.node.DueAt = timePtr(s.due)
	s.node.CompletedAt = timePtr(s.completed)
	s.node.Tags = s.tags
	if s.node.Tags == nil {
		s.node.Tags = []string{}
	}

	return s.node
}

func int64Ptr(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}

	result := value.Int64

	return &result
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	result := value.Time

	return &result
}

// NodeRow is a node plus list counters.
type NodeRow struct {
	Node
	ChildCount      int
	CommentCount    int
	LatestCommentAt *time.Time
}

// MemberRow is a direct member row.
type MemberRow struct {
	Subject string
	Role    string
}

// Get returns one node.
func (r *Repository) Get(ctx context.Context, id int64) (Node, error) {
	scan := newScan()
	err := r.pool.QueryRow(ctx, `
		SELECT id, parent_id, title, description, due_at, completed_at, tags,
		       sort_order, created_by, created_at, updated_at
		FROM nodes WHERE id=$1`, id).Scan(scan.dest()...)
	if err == pgx.ErrNoRows {
		return Node{}, apperr.NotFound("节点不存在")
	}
	if err != nil {
		return Node{}, err
	}

	return scan.value(), nil
}

// List returns every node with list counters.
func (r *Repository) List(ctx context.Context) ([]NodeRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.parent_id, n.title, n.description, n.due_at, n.completed_at, n.tags,
		       n.sort_order, n.created_by, n.created_at, n.updated_at,
		       (SELECT count(*) FROM nodes c WHERE c.parent_id = n.id),
		       (SELECT count(*) FROM comments c WHERE c.node_id = n.id),
		       (SELECT max(c.created_at) FROM comments c WHERE c.node_id = n.id)
		FROM nodes n
		ORDER BY n.parent_id NULLS FIRST, n.sort_order, n.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []NodeRow
	for rows.Next() {
		scan := newScan()
		var childCount, commentCount int
		var latest pgtype.Timestamptz
		args := append(scan.dest(), &childCount, &commentCount, &latest)

		if err := rows.Scan(args...); err != nil {
			return nil, err
		}

		result = append(result, NodeRow{
			Node:            scan.value(),
			ChildCount:      childCount,
			CommentCount:    commentCount,
			LatestCommentAt: timePtr(latest),
		})
	}

	return result, rows.Err()
}

// Children returns direct children, or roots when parentID is nil.
func (r *Repository) Children(ctx context.Context, parentID *int64) ([]NodeRow, error) {
	rows, err := r.pool.Query(ctx, `
  SELECT n.id, n.parent_id, n.title, n.description, n.due_at, n.completed_at, n.tags,
         n.sort_order, n.created_by, n.created_at, n.updated_at,
         (SELECT count(*) FROM nodes c WHERE c.parent_id = n.id),
         (SELECT count(*) FROM comments c WHERE c.node_id = n.id)
  FROM nodes n
  WHERE ($1::bigint IS NULL AND n.parent_id IS NULL) OR n.parent_id = $1
  ORDER BY n.sort_order, n.id`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []NodeRow
	for rows.Next() {
		scan := newScan()
		var childCount, commentCount int
		args := append(scan.dest(), &childCount, &commentCount)
		if err := rows.Scan(args...); err != nil {
			return nil, err
		}
		result = append(result, NodeRow{Node: scan.value(), ChildCount: childCount, CommentCount: commentCount})
	}
	return result, rows.Err()
}

// Breadcrumb returns the path from the root to the node.
func (r *Repository) Breadcrumb(ctx context.Context, id int64) ([]Breadcrumb, error) {
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE chain AS (
			SELECT id, parent_id, title, 0 AS depth FROM nodes WHERE id=$1
			UNION ALL
			SELECT n.id, n.parent_id, n.title, c.depth + 1
			FROM nodes n JOIN chain c ON n.id = c.parent_id
		)
		SELECT id, title FROM chain WHERE depth > 0 ORDER BY depth DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Breadcrumb
	for rows.Next() {
		var item Breadcrumb

		if err := rows.Scan(&item.ID, &item.Title); err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	return result, rows.Err()
}

// Members returns direct leaders and members.
func (r *Repository) Members(ctx context.Context, nodeID int64) ([]MemberRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT subject, role FROM node_members WHERE node_id=$1
		ORDER BY CASE role WHEN 'leader' THEN 0 ELSE 1 END, created_at, subject`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MemberRow
	for rows.Next() {
		var member MemberRow

		if err := rows.Scan(&member.Subject, &member.Role); err != nil {
			return nil, err
		}

		result = append(result, member)
	}

	return result, rows.Err()
}

// RootLeaders returns direct leader subjects for root nodes.
func (r *Repository) RootLeaders(ctx context.Context, rootIDs []int64) (map[int64][]string, error) {
	result := make(map[int64][]string, len(rootIDs))
	if len(rootIDs) == 0 {
		return result, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT node_id, subject FROM node_members
		WHERE node_id = ANY($1) AND role='leader'
		ORDER BY node_id, created_at, subject`, rootIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var nodeID int64
		var subject string

		if err := rows.Scan(&nodeID, &subject); err != nil {
			return nil, err
		}

		result[nodeID] = append(result[nodeID], subject)
	}

	return result, rows.Err()
}

// Counts summarizes a subtree.
type Counts struct {
	Nodes       int
	Comments    int
	Attachments int
}

// SubtreeCounts counts nodes, comments and attachments in a subtree.
func (r *Repository) SubtreeCounts(ctx context.Context, nodeID int64) (Counts, error) {
	const query = `
WITH RECURSIVE subtree AS (
    SELECT id FROM nodes WHERE id = $1
    UNION ALL
    SELECT n.id FROM nodes n JOIN subtree s ON n.parent_id = s.id
)
SELECT
    (SELECT count(*) FROM subtree),
    (SELECT count(*) FROM comments c JOIN subtree s ON s.id = c.node_id),
    (SELECT count(*) FROM comment_attachments a
     JOIN comments c ON c.id = a.comment_id
     JOIN subtree s ON s.id = c.node_id)`
	var counts Counts
	err := r.pool.QueryRow(ctx, query, nodeID).Scan(&counts.Nodes, &counts.Comments, &counts.Attachments)
	return counts, err
}
