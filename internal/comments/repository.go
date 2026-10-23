package comments

import (
	"context"
	"time"

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

// CommentRow is a stored comment.
type CommentRow struct {
	ID        int64
	NodeID    int64
	Author    string
	Body      string
	CreatedAt time.Time
}

// AttachmentRow is a stored attachment.
type AttachmentRow struct {
	ID          int64
	CommentID   int64
	ObjectKey   string
	ContentType string
	SizeBytes   int64
}

// Create publishes a comment and claims its images in the requested order atomically.
func (r *Repository) Create(ctx context.Context, nodeID int64, author, body string, attachmentIDs []int64) (int64, error) {
	var id int64
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO comments (node_id, author_subject, body) VALUES ($1,$2,$3) RETURNING id`,
			nodeID, author, body).Scan(&id); err != nil {
			return err
		}
		for position, attachmentID := range attachmentIDs {
			tag, err := tx.Exec(ctx, `
				UPDATE comment_attachments
				SET comment_id=$1, position=$2, pending_node_id=NULL, uploader_subject=NULL
				WHERE id=$3 AND comment_id IS NULL AND pending_node_id=$4 AND uploader_subject=$5
				  AND created_at > now() - interval '24 hours'`, id, position, attachmentID, nodeID, author)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return apperr.BadRequest("图片不可用或已过期，请移除后重新添加")
			}
		}
		return nil
	})
	return id, err
}

// Get returns one comment row.
func (r *Repository) Get(ctx context.Context, id int64) (CommentRow, error) {
	var row CommentRow
	err := r.pool.QueryRow(ctx,
		`SELECT id, node_id, author_subject, body, created_at FROM comments WHERE id=$1`, id).
		Scan(&row.ID, &row.NodeID, &row.Author, &row.Body, &row.CreatedAt)
	if err == pgx.ErrNoRows {
		return CommentRow{}, pgx.ErrNoRows
	}
	return row, err
}

// Exists reports whether a comment exists.
func (r *Repository) Exists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM comments WHERE id=$1)`, id).Scan(&exists)
	return exists, err
}

// List returns comments newest first.
func (r *Repository) List(ctx context.Context, nodeID int64, before *int64, limit int) ([]CommentRow, bool, error) {
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
	var result []CommentRow
	for rows.Next() {
		var row CommentRow
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
func (r *Repository) AttachmentsForComments(ctx context.Context, commentIDs []int64) (map[int64][]AttachmentRow, error) {
	result := make(map[int64][]AttachmentRow)
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
		var row AttachmentRow
		if err := rows.Scan(&row.ID, &row.CommentID, &row.ObjectKey, &row.ContentType, &row.SizeBytes); err != nil {
			return nil, err
		}
		result[row.CommentID] = append(result[row.CommentID], row)
	}
	return result, rows.Err()
}

// AttachmentByID returns one attachment row.
func (r *Repository) AttachmentByID(ctx context.Context, id int64) (AttachmentRow, error) {
	var row AttachmentRow
	err := r.pool.QueryRow(ctx, `
		SELECT id, comment_id, object_key, content_type, size_bytes
		FROM comment_attachments WHERE id=$1 AND comment_id IS NOT NULL`, id).
		Scan(&row.ID, &row.CommentID, &row.ObjectKey, &row.ContentType, &row.SizeBytes)
	return row, err
}

// ReactionsForComments returns aggregated reactions grouped by comment id.
func (r *Repository) ReactionsForComments(ctx context.Context, commentIDs []int64, viewer string) (map[int64][]Reaction, error) {
	result := make(map[int64][]Reaction)
	if len(commentIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT comment_id, emoji, count(*)::int, bool_or(subject=$2)
		FROM comment_reactions
		WHERE comment_id = ANY($1)
		GROUP BY comment_id, emoji`, commentIDs, viewer)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var commentID int64
		var reaction Reaction
		if err := rows.Scan(&commentID, &reaction.Emoji, &reaction.Count, &reaction.ReactedByMe); err != nil {
			return nil, err
		}
		result[commentID] = append(result[commentID], reaction)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for id := range result {
		sortReactions(result[id])
	}
	return result, nil
}

// ReactionSubjects returns the people who used an emoji, in reaction order.
func (r *Repository) ReactionSubjects(ctx context.Context, commentID int64, emoji string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT subject FROM comment_reactions
		WHERE comment_id=$1 AND emoji=$2
		ORDER BY created_at, subject`, commentID, emoji)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// ToggleReaction toggles one emoji for one user and returns the new summary.
func (r *Repository) ToggleReaction(ctx context.Context, commentID int64, subject, emoji string) ([]Reaction, error) {
	result := make([]Reaction, 0)
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			DELETE FROM comment_reactions
			WHERE comment_id=$1 AND subject=$2 AND emoji=$3`, commentID, subject, emoji)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			if _, err := tx.Exec(ctx, `
				INSERT INTO comment_reactions (comment_id, subject, emoji)
				VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, commentID, subject, emoji); err != nil {
				return err
			}
		}
		rows, err := tx.Query(ctx, `
			SELECT emoji, count(*)::int, bool_or(subject=$2)
			FROM comment_reactions WHERE comment_id=$1
			GROUP BY emoji`, commentID, subject)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var reaction Reaction
			if err := rows.Scan(&reaction.Emoji, &reaction.Count, &reaction.ReactedByMe); err != nil {
				return err
			}
			result = append(result, reaction)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		sortReactions(result)
		return nil
	})
	return result, err
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

func sortReactions(reactions []Reaction) {
	order := make(map[string]int, len(EmojiKeys))
	for index, key := range EmojiKeys {
		order[key] = index
	}
	for i := 1; i < len(reactions); i++ {
		for j := i; j > 0 && order[reactions[j].Emoji] < order[reactions[j-1].Emoji]; j-- {
			reactions[j], reactions[j-1] = reactions[j-1], reactions[j]
		}
	}
}
