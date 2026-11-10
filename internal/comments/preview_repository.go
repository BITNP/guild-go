package comments

import "context"

type previewRow struct {
	commentRow
	AttachmentCount int
}

type recentRow struct {
	commentRow
	NodeTitle string
	RootTitle string
}

func (r *Repository) latestForNodes(ctx context.Context, nodeIDs []int64) ([]previewRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (node_id) id, node_id, author_subject, body, created_at,
		       (SELECT count(*) FROM comment_attachments a WHERE a.comment_id=c.id)
		FROM comments c
		WHERE node_id = ANY($1)
		ORDER BY node_id, id DESC`, nodeIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []previewRow
	for rows.Next() {
		var row previewRow
		if err := rows.Scan(&row.ID, &row.NodeID, &row.Author, &row.Body, &row.CreatedAt, &row.AttachmentCount); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *Repository) recent(ctx context.Context, limit int) ([]recentRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.node_id, c.author_subject, c.body, c.created_at, n.title,
		       COALESCE(root.title, n.title)
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
	defer rows.Close()

	var result []recentRow
	for rows.Next() {
		var row recentRow
		if err := rows.Scan(&row.ID, &row.NodeID, &row.Author, &row.Body, &row.CreatedAt, &row.NodeTitle, &row.RootTitle); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
