package nodes

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/database"
)

// Create inserts a node and its direct members in one transaction.
func (r *Repository) Create(ctx context.Context, req CreateRequest, createdBy string, leaders, members []string) (int64, error) {
	var id int64

	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO nodes (parent_id, title, description, due_at, tags, sort_order, created_by)
			VALUES ($1, $2, $3, $4, $5,
			        COALESCE((SELECT max(sort_order) + 1 FROM nodes WHERE parent_id IS NOT DISTINCT FROM $1), 0),
			        $6)
			RETURNING id`,
			req.ParentID, req.Title, req.Description, req.DueAt, req.Tags, createdBy).Scan(&id)
		if err != nil {
			return err
		}

		if req.Description != "" {
			if _, err := tx.Exec(ctx, `
				INSERT INTO node_description_changes (node_id, author_subject, old_description, new_description, created_at)
				SELECT id, created_by, '', description, created_at FROM nodes WHERE id=$1`, id); err != nil {
				return err
			}
		}

		for _, subject := range leaders {
			if _, err := tx.Exec(ctx, `INSERT INTO node_members (node_id, subject, role) VALUES ($1,$2,'leader')`, id, subject); err != nil {
				return err
			}
		}

		for _, subject := range members {
			if _, err := tx.Exec(ctx, `INSERT INTO node_members (node_id, subject, role) VALUES ($1,$2,'member')`, id, subject); err != nil {
				return err
			}
		}

		if req.ParentID != nil {
			if _, err := tx.Exec(ctx, `UPDATE nodes SET updated_at=now() WHERE id=$1`, *req.ParentID); err != nil {
				return err
			}
		}

		return nil
	})

	return id, err
}

// Update replaces editable fields and toggles real completion time.
func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest, subject string) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		var previous string
		err := tx.QueryRow(ctx, `SELECT description FROM nodes WHERE id=$1 FOR UPDATE`, id).Scan(&previous)
		if err == pgx.ErrNoRows {
			return apperr.NotFound("节点不存在")
		}
		if err != nil {
			return err
		}

		var updatedAt time.Time
		if err := tx.QueryRow(ctx, `
		UPDATE nodes SET
			title=$2,
			description=$3,
			due_at=$4,
			tags=$5,
			completed_at = CASE WHEN $6 THEN COALESCE(completed_at, now()) ELSE NULL END,
			updated_at=clock_timestamp()
		WHERE id=$1
		RETURNING updated_at`,
			id, req.Title, req.Description, req.DueAt, req.Tags, req.Completed).Scan(&updatedAt); err != nil {
			return err
		}

		if previous != req.Description {
			_, err := tx.Exec(ctx, `
				INSERT INTO node_description_changes (node_id, author_subject, old_description, new_description, created_at)
				VALUES ($1,$2,$3,$4,$5)`, id, subject, previous, req.Description, updatedAt)

			return err
		}

		return nil
	})
}

func sameParent(a, b *int64) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

// Move changes only the subtree root's parent and appends it to the new siblings.
func (r *Repository) Move(ctx context.Context, id int64, parentID *int64, authorize func(pgx.Tx) error) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		// Serialize tree writes so concurrent moves cannot create a cycle between
		// validation and update, and destination ordering stays consistent.
		if _, err := tx.Exec(ctx, `LOCK TABLE nodes IN EXCLUSIVE MODE`); err != nil {
			return err
		}

		var previous pgtype.Int8
		if err := tx.QueryRow(ctx, `SELECT parent_id FROM nodes WHERE id=$1`, id).Scan(&previous); err != nil {
			if err == pgx.ErrNoRows {
				return apperr.NotFound("节点不存在")
			}

			return err
		}

		if err := authorize(tx); err != nil {
			return err
		}

		if sameParent(int64Ptr(previous), parentID) {
			return nil
		}

		if parentID != nil {
			var exists, cycle bool
			if err := tx.QueryRow(ctx, `
				WITH RECURSIVE ancestors AS (
					SELECT id, parent_id FROM nodes WHERE id=$1
					UNION ALL
					SELECT n.id, n.parent_id FROM nodes n JOIN ancestors a ON n.id=a.parent_id
				)
				SELECT EXISTS (SELECT 1 FROM ancestors), EXISTS (SELECT 1 FROM ancestors WHERE id=$2)`,
				*parentID, id).Scan(&exists, &cycle); err != nil {
				return err
			}

			if !exists {
				return apperr.NotFound("目标父节点不存在")
			}

			if cycle {
				return apperr.BadRequest("不能将节点移动到自身或其下级节点")
			}
		}

		if _, err := tx.Exec(ctx, `
			UPDATE nodes SET parent_id=$2,
				sort_order=COALESCE((SELECT max(sort_order)+1 FROM nodes WHERE parent_id IS NOT DISTINCT FROM $2), 0),
				updated_at=clock_timestamp()
			WHERE id=$1`, id, parentID); err != nil {
			return err
		}

		_, err := tx.Exec(ctx, `UPDATE nodes SET updated_at=clock_timestamp() WHERE id=$1 OR id=$2`, int64Ptr(previous), parentID)

		return err
	})
}

// Delete removes a subtree and returns its published and pending image keys.
func (r *Repository) Delete(ctx context.Context, id int64, authorize func(pgx.Tx) error) ([]string, error) {
	var keys []string
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `LOCK TABLE nodes IN EXCLUSIVE MODE`); err != nil {
			return err
		}

		rows, err := tx.Query(ctx, `
			WITH RECURSIVE subtree AS (
				SELECT id FROM nodes WHERE id=$1
				UNION ALL
				SELECT n.id FROM nodes n JOIN subtree s ON n.parent_id=s.id
			)
			SELECT id FROM subtree`, id)
		if err != nil {
			return err
		}
		nodeIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
			return err
		}
		if len(nodeIDs) == 0 {
			return apperr.NotFound("节点不存在")
		}
		if err := authorize(tx); err != nil {
			return err
		}

		rows, err = tx.Query(ctx, `
			SELECT a.object_key FROM comment_attachments a
			LEFT JOIN comments c ON c.id=a.comment_id
			WHERE COALESCE(c.node_id, a.pending_node_id) = ANY($1)
			ORDER BY a.id`, nodeIDs)
		if err != nil {
			return err
		}
		keys, err = pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `DELETE FROM nodes WHERE id=$1`, id)
		return err
	})
	return keys, err
}

// ReplaceMembers updates the direct membership set without recreating unchanged rows.
func (r *Repository) ReplaceMembers(ctx context.Context, id int64, leaders, members []string) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		var lockedID int64
		if err := tx.QueryRow(ctx, `SELECT id FROM nodes WHERE id=$1 FOR UPDATE`, id).Scan(&lockedID); err != nil {
			if err == pgx.ErrNoRows {
				return apperr.NotFound("节点不存在")
			}

			return err
		}

		// Keep unchanged (subject, role) pairs; recreate only removed/changed roles.
		if _, err := tx.Exec(ctx, `DELETE FROM node_members
            WHERE node_id=$1 AND NOT (
                (role='leader' AND subject = ANY(COALESCE($2::text[], '{}'))) OR
                (role='member' AND subject = ANY(COALESCE($3::text[], '{}')))
            )`, id, leaders, members); err != nil {
			return err
		}

		for role, subjects := range map[string][]string{"leader": leaders, "member": members} {
			for _, subject := range subjects {
				if _, err := tx.Exec(ctx, `INSERT INTO node_members (node_id, subject, role)
                    VALUES ($1,$2,$3) ON CONFLICT (node_id, subject) DO NOTHING`, id, subject, role); err != nil {
					return err
				}
			}
		}

		_, err := tx.Exec(ctx, `UPDATE nodes SET updated_at=now() WHERE id=$1`, id)

		return err
	})
}

// Reorder rewrites sort_order for exactly the direct children of parentID.
// A nil parentID means root nodes.
func (r *Repository) Reorder(ctx context.Context, parentID *int64, ids []int64) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id FROM nodes
			WHERE ($1::bigint IS NULL AND parent_id IS NULL) OR parent_id=$1
			ORDER BY id FOR UPDATE`, parentID)
		if err != nil {
			return err
		}

		var current []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()

				return err
			}

			current = append(current, id)
		}
		rows.Close()

		if err := rows.Err(); err != nil {
			return err
		}

		if !sameIDSet(current, ids) {
			return apperr.BadRequest("排序列表必须完整覆盖该父节点的直接子节点")
		}

		for index, id := range ids {
			if _, err := tx.Exec(ctx, `UPDATE nodes SET sort_order=$2 WHERE id=$1`, id, index); err != nil {
				return err
			}
		}

		if parentID != nil {
			if _, err := tx.Exec(ctx, `UPDATE nodes SET updated_at=now() WHERE id=$1`, *parentID); err != nil {
				return err
			}
		}

		return nil
	})
}

func sameIDSet(current, requested []int64) bool {
	if len(current) != len(requested) {
		return false
	}

	seen := make(map[int64]struct{}, len(requested))
	for _, id := range requested {
		if _, ok := seen[id]; ok {
			return false
		}

		seen[id] = struct{}{}
	}

	for _, id := range current {
		if _, ok := seen[id]; !ok {
			return false
		}
	}

	return true
}
