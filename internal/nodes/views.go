package nodes

import (
	"context"

	"git.bitnp.net/BITNP/guild-go/internal/auth"
)

func (s *Service) childViews(ctx context.Context, session auth.Session, rows []NodeRow, leaders, members map[int64]struct{}) ([]Child, error) {
	if len(rows) == 0 {
		return []Child{}, nil
	}
	nodeIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		nodeIDs = append(nodeIDs, row.ID)
	}
	latest, err := s.comments.LatestForNodes(ctx, session, nodeIDs)
	if err != nil {
		return nil, err
	}
	result := make([]Child, 0, len(rows))
	for _, row := range rows {
		summary := Summary{
			ID:           row.ID,
			ParentID:     row.ParentID,
			Title:        row.Title,
			DueAt:        row.DueAt,
			CompletedAt:  row.CompletedAt,
			Tags:         nonNil(row.Tags),
			SortOrder:    row.SortOrder,
			ChildCount:   row.ChildCount,
			CommentCount: row.CommentCount,
			MyRelation:   relationFor(row.ID, leaders, members),
			UpdatedAt:    row.UpdatedAt,
		}
		child := Child{Summary: summary}
		if preview, ok := latest[row.ID]; ok {
			child.LatestComment = &preview
			child.Summary.LatestCommentAt = &preview.CreatedAt
		}
		result = append(result, child)
	}
	return result, nil
}
