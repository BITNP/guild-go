package nodes

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

// Activity is a compact event in the site-wide recent feed. IDs are unique
// within a kind; clients use (kind, id) as the item key.
type Activity struct {
	ID              int64               `json:"id"`
	Kind            string              `json:"kind"`
	NodeID          int64               `json:"nodeId"`
	NodeTitle       string              `json:"nodeTitle"`
	RootTitle       string              `json:"rootTitle"`
	Author          *users.Summary      `json:"author"`
	BodyPreview     string              `json:"bodyPreview"`
	CreatedAt       time.Time           `json:"createdAt"`
	DueAt           *time.Time          `json:"dueAt"`
	CompletedAt     *time.Time          `json:"completedAt"`
	ReactionSummary []comments.Reaction `json:"reactionSummary"`
	AttachmentCount int                 `json:"attachmentCount"`
}

type activityRow struct {
	Activity
	Subject string
}

// RecentActivity reads existing comments and each node's latest activity.
// Node updates have no stored actor; do not attribute them to the creator.
func (r *Repository) RecentActivity(ctx context.Context, subject string, relatedOnly bool, relatedIDs []int64, limit int) ([]activityRow, error) {
	rows, err := r.pool.Query(ctx, `
		WITH events AS (
			SELECT c.id, 'comment' AS kind, c.node_id, c.author_subject, c.body,
			       c.created_at,
			       (SELECT count(*)::int FROM comment_attachments a WHERE a.comment_id=c.id) AS attachment_count
			FROM comments c
			UNION ALL
			SELECT id,
			       CASE WHEN GREATEST(updated_at, completed_at) > created_at THEN 'node_updated' ELSE 'node_created' END,
			       id,
			       CASE WHEN GREATEST(updated_at, completed_at) > created_at THEN '' ELSE created_by END,
			       '', GREATEST(created_at, updated_at, completed_at), 0
			FROM nodes
		), recent AS (
			SELECT * FROM events
			WHERE NOT $2::boolean OR author_subject=$1 OR node_id=ANY($3::bigint[])
			   OR node_id IN (SELECT id FROM nodes WHERE created_by=$1)
			ORDER BY created_at DESC, kind DESC, id DESC LIMIT $4
		)
		SELECT e.id, e.kind, e.node_id, n.title, root.title, e.author_subject,
		       e.body, e.created_at, e.attachment_count, n.due_at, n.completed_at
		FROM recent e JOIN nodes n ON n.id=e.node_id
		CROSS JOIN LATERAL (
			WITH RECURSIVE chain AS (
				SELECT id, parent_id, title FROM nodes WHERE id=n.id
				UNION ALL
				SELECT p.id, p.parent_id, p.title FROM nodes p JOIN chain c ON p.id=c.parent_id
			)
			SELECT title FROM chain WHERE parent_id IS NULL
		) root
		ORDER BY e.created_at DESC, e.kind DESC, e.id DESC`, subject, relatedOnly, relatedIDs, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (activityRow, error) {
		var row activityRow
		err := r.Scan(&row.ID, &row.Kind, &row.NodeID, &row.NodeTitle, &row.RootTitle,
			&row.Subject, &row.BodyPreview, &row.CreatedAt, &row.AttachmentCount, &row.DueAt, &row.CompletedAt)
		return row, err
	})
}

func (s *Service) activityViews(ctx context.Context, session auth.Session, rows []activityRow) ([]Activity, error) {
	ids := make([]int64, 0, len(rows))
	subjects := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Kind == "comment" {
			ids = append(ids, row.ID)
		}
		if row.Subject != "" {
			subjects = append(subjects, row.Subject)
		}
	}
	authors := s.users.Summaries(ctx, session, subjects)
	reactions, err := s.comments.ReactionsForComments(ctx, ids, session.Subject)
	if err != nil {
		return nil, err
	}
	result := make([]Activity, 0, len(rows))
	for _, row := range rows {
		view := row.Activity
		if row.Subject != "" {
			author := authors.For(row.Subject)
			view.Author = &author
		}
		view.BodyPreview = comments.TextPreview(view.BodyPreview, 120)
		view.ReactionSummary = []comments.Reaction{}
		if row.Kind == "comment" && reactions[row.ID] != nil {
			view.ReactionSummary = reactions[row.ID]
		}
		result = append(result, view)
	}
	return result, nil
}
