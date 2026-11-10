package comments

import (
	"context"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
)

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
	var commentRows []commentRow
	var subjects []string
	for _, row := range rows {
		if row.Kind == "comment" {
			commentRows = append(commentRows, row.commentRow)
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
