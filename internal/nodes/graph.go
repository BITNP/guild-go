package nodes

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
)

const (
	defaultGraphPageSize = 200
	maxGraphPageSize     = 300
)

// Graph returns one stable ID-ordered page of compact graph nodes.
func (s *Service) Graph(ctx context.Context, _ auth.Session, cursor int64, limit int) (GraphPage, error) {
	if limit <= 0 {
		limit = defaultGraphPageSize
	}
	if limit > maxGraphPageSize {
		limit = maxGraphPageSize
	}
	rows, hasMore, err := s.repo.Graph(ctx, cursor, limit)
	if err != nil {
		return GraphPage{}, err
	}
	items := make([]GraphNode, 0, len(rows))
	for _, row := range rows {
		items = append(items, graphNode(row))
	}
	var nextCursor *int64
	if hasMore && len(items) > 0 {
		value := items[len(items)-1].ID
		nextCursor = &value
	}
	return GraphPage{Items: items, HasMore: hasMore, NextCursor: nextCursor}, nil
}

// GraphNeighborhood returns the focused node, its parent and direct children.
func (s *Service) GraphNeighborhood(ctx context.Context, _ auth.Session, id int64) (GraphNeighborhood, error) {
	node, err := s.repo.GraphByID(ctx, id)
	if err != nil {
		return GraphNeighborhood{}, err
	}
	result := GraphNeighborhood{
		Node:     graphNode(node),
		Children: []GraphNode{},
	}
	if node.ParentID != nil {
		parent, err := s.repo.GraphByID(ctx, *node.ParentID)
		if err != nil {
			return GraphNeighborhood{}, err
		}
		parentTitle := parent.Title
		result.Node.ParentTitle = &parentTitle
		parentNode := graphNode(parent)
		result.Parent = &parentNode
	}
	children, err := s.repo.Children(ctx, &id)
	if err != nil {
		return GraphNeighborhood{}, err
	}
	parentTitle := node.Title
	for _, child := range children {
		result.Children = append(result.Children, GraphNode{
			ID:           child.ID,
			ParentID:     child.ParentID,
			ParentTitle:  &parentTitle,
			Title:        child.Title,
			SortOrder:    child.SortOrder,
			CompletedAt:  child.CompletedAt,
			ChildCount:   child.ChildCount,
			CommentCount: child.CommentCount,
		})
	}
	return result, nil
}

func graphNode(row GraphRow) GraphNode {
	return GraphNode{
		ID:           row.ID,
		ParentID:     row.ParentID,
		ParentTitle:  row.ParentTitle,
		Title:        row.Title,
		SortOrder:    row.SortOrder,
		CompletedAt:  row.CompletedAt,
		ChildCount:   row.ChildCount,
		CommentCount: row.CommentCount,
	}
}

// GraphRow is a graph node plus its parent title for progressive rendering.
type GraphRow struct {
	Node
	ChildCount   int
	CommentCount int
	ParentTitle  *string
}

// Graph returns up to limit nodes after cursor, ordered by ID.
func (r *Repository) Graph(ctx context.Context, cursor int64, limit int) ([]GraphRow, bool, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.parent_id, n.title, n.description, n.due_at, n.completed_at, n.tags,
		       n.sort_order, n.created_by, n.created_at, n.updated_at,
		       (SELECT count(*) FROM nodes c WHERE c.parent_id = n.id),
		       (SELECT count(*) FROM comments c WHERE c.node_id = n.id),
		       p.title
		FROM nodes n
		LEFT JOIN nodes p ON p.id = n.parent_id
		WHERE n.id > $1
		ORDER BY n.id
		LIMIT $2`, cursor, limit+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	result := make([]GraphRow, 0, limit+1)
	for rows.Next() {
		scan := newScan()
		var childCount, commentCount int
		var parentTitle pgtype.Text
		args := append(scan.dest(), &childCount, &commentCount, &parentTitle)
		if err := rows.Scan(args...); err != nil {
			return nil, false, err
		}
		row := GraphRow{
			Node:         scan.value(),
			ChildCount:   childCount,
			CommentCount: commentCount,
		}
		if parentTitle.Valid {
			value := parentTitle.String
			row.ParentTitle = &value
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(result) > limit
	if hasMore {
		result = result[:limit]
	}
	return result, hasMore, nil
}

// GraphByID returns one graph row with its list counters and parent title.
func (r *Repository) GraphByID(ctx context.Context, id int64) (GraphRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.parent_id, n.title, n.description, n.due_at, n.completed_at, n.tags,
		       n.sort_order, n.created_by, n.created_at, n.updated_at,
		       (SELECT count(*) FROM nodes c WHERE c.parent_id = n.id),
		       (SELECT count(*) FROM comments c WHERE c.node_id = n.id),
		       p.title
		FROM nodes n
		LEFT JOIN nodes p ON p.id = n.parent_id
		WHERE n.id = $1`, id)
	if err != nil {
		return GraphRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GraphRow{}, err
		}
		return GraphRow{}, apperr.NotFound("节点不存在")
	}
	scan := newScan()
	var childCount, commentCount int
	var parentTitle pgtype.Text
	args := append(scan.dest(), &childCount, &commentCount, &parentTitle)
	if err := rows.Scan(args...); err != nil {
		return GraphRow{}, err
	}
	row := GraphRow{
		Node:         scan.value(),
		ChildCount:   childCount,
		CommentCount: commentCount,
	}
	if parentTitle.Valid {
		value := parentTitle.String
		row.ParentTitle = &value
	}
	return row, rows.Err()
}
