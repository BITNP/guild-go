package comments

import (
	"context"
	"time"
)

type progressRow struct {
	commentRow
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
