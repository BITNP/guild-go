package comments

import (
	"context"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

// DescriptionChange retains the text on both sides of a saved description edit.
type DescriptionChange struct {
	ID        int64         `json:"id"`
	NodeID    int64         `json:"nodeId"`
	Author    users.Summary `json:"author"`
	Before    string        `json:"before"`
	After     string        `json:"after"`
	CreatedAt time.Time     `json:"createdAt"`
}

// ProgressItem is either a user comment or a description change.
type ProgressItem struct {
	Kind              string             `json:"kind"`
	Comment           *Comment           `json:"comment,omitempty"`
	DescriptionChange *DescriptionChange `json:"descriptionChange,omitempty"`
}

// ProgressCursor identifies a position in the combined chronological feed.
type ProgressCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	Kind      string    `json:"kind"`
	ID        int64     `json:"id"`
}

// ProgressList is a page of comments and description changes, newest first.
type ProgressList struct {
	Items      []ProgressItem  `json:"items"`
	HasMore    bool            `json:"hasMore"`
	NextBefore *ProgressCursor `json:"nextBefore"`
}

type progressRow struct {
	CommentRow
	Kind   string
	Before string
	After  string
}

func (r *Repository) progress(ctx context.Context, nodeID int64, before *ProgressCursor, limit int) ([]progressRow, bool, error) {
	var beforeTime *time.Time
	var beforeKind string
	var beforeID int64
	if before != nil {
		beforeTime, beforeKind, beforeID = &before.CreatedAt, before.Kind, before.ID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT kind, id, node_id, author_subject, body, old_description, new_description, created_at
		FROM (
			SELECT 'comment'::text AS kind, id, node_id, author_subject, body,
			       ''::text AS old_description, ''::text AS new_description, created_at
			FROM comments WHERE node_id=$1
			UNION ALL
			SELECT 'description_change', id, node_id, author_subject, '', old_description, new_description, created_at
			FROM node_description_changes WHERE node_id=$1
		) AS progress
		WHERE $2::timestamptz IS NULL OR (created_at, kind, id) < ($2, $3::text, $4::bigint)
		ORDER BY created_at DESC, kind DESC, id DESC
		LIMIT $5`, nodeID, beforeTime, beforeKind, beforeID, limit+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	var result []progressRow
	for rows.Next() {
		var row progressRow
		if err := rows.Scan(&row.Kind, &row.ID, &row.NodeID, &row.Author, &row.Body, &row.Before, &row.After, &row.CreatedAt); err != nil {
			return nil, false, err
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

// Progress combines description history and comments without changing comment APIs.
func (s *Service) Progress(ctx context.Context, session auth.Session, nodeID int64, before *ProgressCursor, limit int) (ProgressList, error) {
	exists, err := s.access.NodeExists(ctx, nodeID)
	if err != nil {
		return ProgressList{}, err
	}
	if !exists {
		return ProgressList{}, apperr.NotFound("节点不存在")
	}
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	rows, hasMore, err := s.repo.progress(ctx, nodeID, before, limit)
	if err != nil {
		return ProgressList{}, err
	}
	var commentRows []CommentRow
	var subjects []string
	for _, row := range rows {
		if row.Kind == "comment" {
			commentRows = append(commentRows, row.CommentRow)
		} else {
			subjects = append(subjects, row.Author)
		}
	}
	commentViews, err := s.assemble(ctx, session, commentRows)
	if err != nil {
		return ProgressList{}, err
	}
	commentsByID := make(map[int64]*Comment, len(commentViews))
	for i := range commentViews {
		commentsByID[commentViews[i].ID] = &commentViews[i]
	}
	authors := s.users.Summaries(ctx, session, subjects)
	items := make([]ProgressItem, 0, len(rows))
	for _, row := range rows {
		item := ProgressItem{Kind: row.Kind}
		if row.Kind == "comment" {
			item.Comment = commentsByID[row.ID]
		} else {
			item.DescriptionChange = &DescriptionChange{
				ID: row.ID, NodeID: row.NodeID, Author: authors[row.Author],
				Before: row.Before, After: row.After, CreatedAt: row.CreatedAt,
			}
		}
		items = append(items, item)
	}
	var nextBefore *ProgressCursor
	if hasMore {
		last := rows[len(rows)-1]
		nextBefore = &ProgressCursor{CreatedAt: last.CreatedAt, Kind: last.Kind, ID: last.ID}
	}
	return ProgressList{Items: items, HasMore: hasMore, NextBefore: nextBefore}, nil
}
