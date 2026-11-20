package comments

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"git.bitnp.net/BITNP/guild-go/internal/auth"
)

// RecentRow is one recent comment across the whole site.
type RecentRow struct {
	ID        int64
	NodeID    int64
	NodeTitle string
	RootTitle string
	Body      string
	CreatedAt time.Time
	Author    string
}

// Recent returns the newest comments across all nodes.
func (r *Repository) Recent(ctx context.Context, limit int) ([]RecentRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.node_id, n.title, c.body, c.created_at, c.author_subject,
		       COALESCE(root.title, n.title) AS root_title
		FROM comments c
		JOIN nodes n ON n.id = c.node_id
		LEFT JOIN LATERAL (
			WITH RECURSIVE chain AS (
				SELECT id, parent_id, title FROM nodes WHERE id = n.id
				UNION ALL
				SELECT p.id, p.parent_id, p.title
				FROM nodes p JOIN chain ch ON p.id = ch.parent_id
			)
			SELECT title FROM chain WHERE parent_id IS NULL LIMIT 1
		) root ON true
		ORDER BY c.created_at DESC, c.id DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (RecentRow, error) {
		var item RecentRow
		err := row.Scan(&item.ID, &item.NodeID, &item.NodeTitle, &item.Body, &item.CreatedAt, &item.Author, &item.RootTitle)
		return item, err
	})
}

// Recent returns the site-wide recent-comment feed.
func (s *Service) Recent(ctx context.Context, session auth.Session, limit int) ([]Recent, error) {
	rows, err := s.repo.Recent(ctx, limit)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []Recent{}, nil
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
			ID:              row.ID,
			NodeID:          row.NodeID,
			NodeTitle:       row.NodeTitle,
			RootTitle:       row.RootTitle,
			Author:          authors.For(row.Author),
			BodyPreview:     TextPreview(row.Body, 120),
			CreatedAt:       row.CreatedAt,
			ReactionSummary: summary,
		})
	}
	return result, nil
}
