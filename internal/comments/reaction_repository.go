package comments

import (
	"context"
	"slices"

	"github.com/jackc/pgx/v5"

	"git.bitnp.net/BITNP/guild-go/internal/database"
)

// Exists reports whether a comment exists.
func (r *Repository) Exists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM comments WHERE id=$1)`, id).Scan(&exists)
	return exists, err
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

func sortReactions(reactions []Reaction) {
	slices.SortFunc(reactions, func(a, b Reaction) int {
		return slices.Index(EmojiKeys, a.Emoji) - slices.Index(EmojiKeys, b.Emoji)
	})
}
