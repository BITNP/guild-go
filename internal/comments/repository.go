package comments

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/database"
)

// Repository contains explicit SQL for comments, reactions and attachments.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a comment repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create publishes a comment and claims its images in the requested order atomically.
func (r *Repository) Create(ctx context.Context, nodeID int64, author, body string, attachmentIDs []int64) (commentRow, error) {
	row := commentRow{NodeID: nodeID, Author: author, Body: body}
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO comments (node_id, author_subject, body) VALUES ($1,$2,$3) RETURNING id, created_at`,
			nodeID, author, body).Scan(&row.ID, &row.CreatedAt); err != nil {
			return err
		}
		for position, attachmentID := range attachmentIDs {
			tag, err := tx.Exec(ctx, `
				UPDATE comment_attachments
				SET comment_id=$1, position=$2, pending_node_id=NULL, uploader_subject=NULL
				WHERE id=$3 AND comment_id IS NULL AND pending_node_id=$4 AND uploader_subject=$5
				  AND created_at > now() - interval '24 hours'`, row.ID, position, attachmentID, nodeID, author)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return apperr.BadRequest("图片不可用或已过期，请移除后重新添加")
			}
		}
		return nil
	})
	return row, err
}

// Get returns one comment row.
func (r *Repository) Get(ctx context.Context, id int64) (commentRow, error) {
	var row commentRow
	err := r.pool.QueryRow(ctx,
		`SELECT id, node_id, author_subject, body, created_at FROM comments WHERE id=$1`, id).
		Scan(&row.ID, &row.NodeID, &row.Author, &row.Body, &row.CreatedAt)
	if err == pgx.ErrNoRows {
		return commentRow{}, pgx.ErrNoRows
	}
	return row, err
}

// List returns comments newest first.
func (r *Repository) List(ctx context.Context, nodeID int64, before *int64, limit int) ([]commentRow, bool, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, node_id, author_subject, body, created_at
		FROM comments
		WHERE node_id=$1 AND ($2::bigint IS NULL OR id < $2)
		ORDER BY id DESC
		LIMIT $3`, nodeID, before, limit+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	var result []commentRow
	for rows.Next() {
		var row commentRow
		if err := rows.Scan(&row.ID, &row.NodeID, &row.Author, &row.Body, &row.CreatedAt); err != nil {
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

// Delete removes a comment and returns its attachment keys.
func (r *Repository) Delete(ctx context.Context, id int64) ([]string, error) {
	var keys []string
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT object_key FROM comment_attachments WHERE comment_id=$1`, id)
		if err != nil {
			return err
		}
		for rows.Next() {
			var key string
			if err := rows.Scan(&key); err != nil {
				rows.Close()
				return err
			}
			keys = append(keys, key)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `DELETE FROM comments WHERE id=$1`, id)
		return err
	})
	return keys, err
}
