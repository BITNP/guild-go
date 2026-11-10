package comments

import (
	"context"
	"strings"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

func (s *Service) LatestForNodes(ctx context.Context, session auth.Session, nodeIDs []int64) (map[int64]Preview, error) {
	result := make(map[int64]Preview, len(nodeIDs))
	if len(nodeIDs) == 0 {
		return result, nil
	}
	rows, err := s.repo.latestForNodes(ctx, nodeIDs)
	if err != nil {
		return nil, err
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
	for _, row := range rows {
		result[row.NodeID] = BuildPreview(row.ID, authors[row.Author], row.Body, row.CreatedAt, reactions[row.ID], row.AttachmentCount)
	}
	return result, nil
}

func (s *Service) Recent(ctx context.Context, session auth.Session, limit int) ([]Recent, error) {
	rows, err := s.repo.recent(ctx, limit)
	if err != nil {
		return nil, err
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
	result := make([]Recent, 0, len(rows))
	for _, row := range rows {
		summary := reactions[row.ID]
		if summary == nil {
			summary = []Reaction{}
		}
		result = append(result, Recent{
			ID: row.ID, NodeID: row.NodeID, NodeTitle: row.NodeTitle, RootTitle: row.RootTitle,
			Author: authors[row.Author], BodyPreview: TextPreview(row.Body, 120),
			CreatedAt: row.CreatedAt, ReactionSummary: summary,
		})
	}
	return result, nil
}

// BuildPreview creates a compact latest-comment view.
func BuildPreview(id int64, author users.Summary, body string, createdAt time.Time, reactions []Reaction, attachmentCount int) Preview {
	if reactions == nil {
		reactions = []Reaction{}
	}
	return Preview{
		ID:              id,
		Author:          author,
		BodyPreview:     TextPreview(body, 200),
		CreatedAt:       createdAt,
		ReactionSummary: reactions,
		AttachmentCount: attachmentCount,
	}
}

// TextPreview truncates text to max runes, preserving line breaks.
func TextPreview(text string, max int) string {
	text = strings.TrimSpace(text)
	if max <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return strings.TrimSpace(string(runes[:max])) + "…"
}
