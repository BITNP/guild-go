package comments

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// attachmentRow is a stored attachment.
type attachmentRow struct {
	ID          int64
	CommentID   int64
	ObjectKey   string
	ContentType string
	SizeBytes   int64
}

// AddUpload records one image awaiting publication.
func (r *Repository) AddUpload(ctx context.Context, nodeID int64, subject, key, contentType string, size int64) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO comment_attachments (pending_node_id, uploader_subject, object_key, content_type, size_bytes)
		VALUES ($1,$2,$3,$4,$5) RETURNING id`, nodeID, subject, key, contentType, size).Scan(&id)
	return id, err
}

// DeleteUpload removes an unpublished image owned by the caller.
func (r *Repository) DeleteUpload(ctx context.Context, id int64, subject string) (string, error) {
	var key string
	err := r.pool.QueryRow(ctx, `DELETE FROM comment_attachments
		WHERE id=$1 AND comment_id IS NULL AND uploader_subject=$2 RETURNING object_key`, id, subject).Scan(&key)
	return key, err
}

// DeleteExpiredUploads removes images abandoned for at least a day.
func (r *Repository) DeleteExpiredUploads(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `DELETE FROM comment_attachments
		WHERE comment_id IS NULL AND created_at <= now() - interval '24 hours' RETURNING object_key`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// AttachmentsForComments returns attachments grouped by comment id.
func (r *Repository) AttachmentsForComments(ctx context.Context, commentIDs []int64) (map[int64][]attachmentRow, error) {
	result := make(map[int64][]attachmentRow)
	if len(commentIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, comment_id, object_key, content_type, size_bytes
		FROM comment_attachments
		WHERE comment_id = ANY($1)
		ORDER BY comment_id, position, id`, commentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var row attachmentRow
		if err := rows.Scan(&row.ID, &row.CommentID, &row.ObjectKey, &row.ContentType, &row.SizeBytes); err != nil {
			return nil, err
		}
		result[row.CommentID] = append(result[row.CommentID], row)
	}
	return result, rows.Err()
}

// AttachmentByID returns one attachment row.
func (r *Repository) AttachmentByID(ctx context.Context, id int64) (attachmentRow, error) {
	var row attachmentRow
	err := r.pool.QueryRow(ctx, `
		SELECT id, comment_id, object_key, content_type, size_bytes
		FROM comment_attachments WHERE id=$1 AND comment_id IS NOT NULL`, id).
		Scan(&row.ID, &row.CommentID, &row.ObjectKey, &row.ContentType, &row.SizeBytes)
	return row, err
}
