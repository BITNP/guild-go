package nodes

import (
	"context"

	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

func (s *Service) childViews(ctx context.Context, session auth.Session, rows []ChildRow, leaders, members map[int64]struct{}) ([]Child, error) {
	if len(rows) == 0 {
		return []Child{}, nil
	}
	latestIDs := make([]int64, 0)
	authorSubjects := make([]string, 0)
	for _, row := range rows {
		if row.Latest != nil {
			latestIDs = append(latestIDs, row.Latest.ID)
			authorSubjects = append(authorSubjects, row.Latest.Author)
		}
	}
	authors := s.users.Summaries(ctx, session, authorSubjects)
	reactions, err := s.repo.ReactionsForComments(ctx, latestIDs, session.Subject)
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
		if row.Latest != nil {
			author, ok := authors[row.Latest.Author]
			if !ok {
				author = users.Summary{Subject: row.Latest.Author, UserName: row.Latest.Author, Positions: []users.Position{}}
			}
			preview := comments.BuildPreview(row.Latest.ID, author, row.Latest.Body, row.Latest.CreatedAt, reactions[row.Latest.ID], row.Latest.AttachmentCount)
			child.LatestComment = &preview
			latestAt := row.Latest.CreatedAt
			child.Summary.LatestCommentAt = &latestAt
		}
		result = append(result, child)
	}
	return result, nil
}

func (s *Service) recentViews(ctx context.Context, session auth.Session, rows []RecentRow) ([]comments.Recent, error) {
	if len(rows) == 0 {
		return []comments.Recent{}, nil
	}
	ids := make([]int64, 0, len(rows))
	subjects := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
		subjects = append(subjects, row.Author)
	}
	authors := s.users.Summaries(ctx, session, subjects)
	reactions, err := s.repo.ReactionsForComments(ctx, ids, session.Subject)
	if err != nil {
		return nil, err
	}
	result := make([]comments.Recent, 0, len(rows))
	for _, row := range rows {
		author, ok := authors[row.Author]
		if !ok {
			author = users.Summary{Subject: row.Author, UserName: row.Author, Positions: []users.Position{}}
		}
		summary := reactions[row.ID]
		if summary == nil {
			summary = []comments.Reaction{}
		}
		result = append(result, comments.Recent{
			ID:              row.ID,
			NodeID:          row.NodeID,
			NodeTitle:       row.NodeTitle,
			RootTitle:       row.RootTitle,
			Author:          author,
			BodyPreview:     comments.TextPreview(row.Body, 120),
			CreatedAt:       row.CreatedAt,
			ReactionSummary: summary,
		})
	}
	return result, nil
}
