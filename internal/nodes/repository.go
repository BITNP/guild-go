package nodes

import (
	"context"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/database"
	"git.bitnp.net/BITNP/guild-go/internal/events"
)

// Repository contains explicit SQL for nodes.
type Repository struct {
	pool   *pgxpool.Pool
	broker *events.Broker
}

// NewRepository creates a node repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// WithBroker attaches an event broker notified after mutations commit.
func (r *Repository) WithBroker(broker *events.Broker) *Repository {
	r.broker = broker
	return r
}

type nodeScan struct {
	node      Node
	parent    pgtype.Int8
	due       pgtype.Timestamptz
	completed pgtype.Timestamptz
	tags      []string
}

func newScan() *nodeScan {
	return &nodeScan{}
}

func (s *nodeScan) dest() []any {
	return []any{
		&s.node.ID, &s.parent, &s.node.Title, &s.node.Description,
		&s.due, &s.completed, &s.tags, &s.node.SortOrder,
		&s.node.CreatedBy, &s.node.CreatedAt, &s.node.UpdatedAt,
	}
}

func (s *nodeScan) value() Node {
	s.node.ParentID = int64Ptr(s.parent)
	s.node.DueAt = timePtr(s.due)
	s.node.CompletedAt = timePtr(s.completed)
	s.node.Tags = s.tags
	if s.node.Tags == nil {
		s.node.Tags = []string{}
	}
	return s.node
}

func int64Ptr(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

// NodeRow is a node plus list counters.
type NodeRow struct {
	Node
	ChildCount      int
	CommentCount    int
	LatestCommentAt *time.Time
}

// LatestRow is the raw latest-comment data for a child.
type LatestRow struct {
	ID              int64
	Body            string
	CreatedAt       time.Time
	Author          string
	AttachmentCount int
}

// ChildRow is a child node plus its latest comment.
type ChildRow struct {
	Node
	ChildCount   int
	CommentCount int
	Latest       *LatestRow
}

// Get returns one node.
func (r *Repository) Get(ctx context.Context, id int64) (Node, error) {
	scan := newScan()
	err := r.pool.QueryRow(ctx, `
		SELECT id, parent_id, title, description, due_at, completed_at, tags,
		       sort_order, created_by, created_at, updated_at
		FROM nodes WHERE id=$1`, id).Scan(scan.dest()...)
	if err == pgx.ErrNoRows {
		return Node{}, apperr.NotFound("节点不存在")
	}
	if err != nil {
		return Node{}, err
	}
	return scan.value(), nil
}

// List returns every node with list counters.
func (r *Repository) List(ctx context.Context) ([]NodeRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.parent_id, n.title, n.description, n.due_at, n.completed_at, n.tags,
		       n.sort_order, n.created_by, n.created_at, n.updated_at,
		       (SELECT count(*) FROM nodes c WHERE c.parent_id = n.id),
		       (SELECT count(*) FROM comments c WHERE c.node_id = n.id),
		       (SELECT max(c.created_at) FROM comments c WHERE c.node_id = n.id)
		FROM nodes n
		ORDER BY n.parent_id NULLS FIRST, n.sort_order, n.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []NodeRow
	for rows.Next() {
		scan := newScan()
		var childCount, commentCount int
		var latest pgtype.Timestamptz
		args := append(scan.dest(), &childCount, &commentCount, &latest)
		if err := rows.Scan(args...); err != nil {
			return nil, err
		}
		result = append(result, NodeRow{
			Node:            scan.value(),
			ChildCount:      childCount,
			CommentCount:    commentCount,
			LatestCommentAt: timePtr(latest),
		})
	}
	return result, rows.Err()
}

// Children returns direct children (or roots when parentID is nil) with their
// latest comment.
func (r *Repository) Children(ctx context.Context, parentID *int64) ([]ChildRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.parent_id, n.title, n.description, n.due_at, n.completed_at, n.tags,
		       n.sort_order, n.created_by, n.created_at, n.updated_at,
		       (SELECT count(*) FROM nodes c WHERE c.parent_id = n.id),
		       (SELECT count(*) FROM comments c WHERE c.node_id = n.id),
		       lc.id, lc.body, lc.created_at, lc.author_subject,
		       (SELECT count(*) FROM comment_attachments a WHERE a.comment_id = lc.id)
		FROM nodes n
		LEFT JOIN LATERAL (
			SELECT id, body, created_at, author_subject
			FROM comments WHERE node_id = n.id
			ORDER BY id DESC LIMIT 1
		) lc ON true
		WHERE ($1::bigint IS NULL AND n.parent_id IS NULL) OR n.parent_id = $1
		ORDER BY n.sort_order, n.id`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ChildRow
	for rows.Next() {
		scan := newScan()
		var childCount, commentCount int
		var latestID pgtype.Int8
		var latestBody pgtype.Text
		var latestAt pgtype.Timestamptz
		var latestAuthor pgtype.Text
		var attachmentCount int
		args := append(scan.dest(), &childCount, &commentCount,
			&latestID, &latestBody, &latestAt, &latestAuthor, &attachmentCount)
		if err := rows.Scan(args...); err != nil {
			return nil, err
		}
		row := ChildRow{Node: scan.value(), ChildCount: childCount, CommentCount: commentCount}
		if latestID.Valid {
			row.Latest = &LatestRow{
				ID:              latestID.Int64,
				Body:            latestBody.String,
				CreatedAt:       latestAt.Time,
				Author:          latestAuthor.String,
				AttachmentCount: attachmentCount,
			}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// Breadcrumb returns the path from the root to the node.
func (r *Repository) Breadcrumb(ctx context.Context, id int64) ([]Breadcrumb, error) {
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE chain AS (
			SELECT id, parent_id, title, 0 AS depth FROM nodes WHERE id=$1
			UNION ALL
			SELECT n.id, n.parent_id, n.title, c.depth + 1
			FROM nodes n JOIN chain c ON n.id = c.parent_id
		)
		SELECT id, title FROM chain WHERE depth > 0 ORDER BY depth DESC`, id)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Breadcrumb, error) {
		var item Breadcrumb
		err := row.Scan(&item.ID, &item.Title)
		return item, err
	})
}

// Create inserts a node and its direct members in one transaction.
func (r *Repository) Create(ctx context.Context, req CreateRequest, createdBy string, leaders, members []string) (int64, error) {
	var (
		id       int64
		eventIDs []int64
	)
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

		evID, err := events.RecordTx(ctx, tx, events.Input{
			Kind:         events.KindNodeCreated,
			NodeID:       &id,
			ActorSubject: createdBy,
			Payload: map[string]any{
				"title":    req.Title,
				"parentId": req.ParentID,
				"dueAt":    req.DueAt,
				"tags":     nonNilStrings(req.Tags),
				"leaders":  nonNilStrings(leaders),
				"members":  nonNilStrings(members),
			},
		})
		if err != nil {
			return err
		}
		eventIDs = append(eventIDs, evID)
		for _, subject := range leaders {
			evID, err := events.RecordTx(ctx, tx, events.Input{
				Kind:         events.KindMemberAssigned,
				NodeID:       &id,
				ActorSubject: createdBy,
				Payload: map[string]any{
					"subject":   subject,
					"role":      "leader",
					"nodeTitle": req.Title,
					"onCreate":  true,
				},
			})
			if err != nil {
				return err
			}
			eventIDs = append(eventIDs, evID)
		}
		for _, subject := range members {
			evID, err := events.RecordTx(ctx, tx, events.Input{
				Kind:         events.KindMemberAssigned,
				NodeID:       &id,
				ActorSubject: createdBy,
				Payload: map[string]any{
					"subject":   subject,
					"role":      "member",
					"nodeTitle": req.Title,
					"onCreate":  true,
				},
			})
			if err != nil {
				return err
			}
			eventIDs = append(eventIDs, evID)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	r.broker.Publish(eventIDs...)
	return id, nil
}

// Update replaces editable fields and toggles real completion time.
func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest, subject string) (Node, error) {
	scan := newScan()
	var eventIDs []int64
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		var (
			oldTitle     string
			previous     string
			oldDue       pgtype.Timestamptz
			oldCompleted pgtype.Timestamptz
			oldTags      []string
		)
		err := tx.QueryRow(ctx, `SELECT title, description, due_at, completed_at, tags FROM nodes WHERE id=$1 FOR UPDATE`, id).
			Scan(&oldTitle, &previous, &oldDue, &oldCompleted, &oldTags)
		if err == pgx.ErrNoRows {
			return apperr.NotFound("节点不存在")
		}
		if err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `
		UPDATE nodes SET
			title=$2,
			description=$3,
			due_at=$4,
			tags=$5,
			completed_at = CASE WHEN $6 THEN COALESCE(completed_at, now()) ELSE NULL END,
			updated_at=clock_timestamp()
		WHERE id=$1
		RETURNING id, parent_id, title, description, due_at, completed_at, tags,
		          sort_order, created_by, created_at, updated_at`,
			id, req.Title, req.Description, req.DueAt, req.Tags, req.Completed).Scan(scan.dest()...); err != nil {
			return err
		}
		updated := scan.value()
		if previous != req.Description {
			if _, err := tx.Exec(ctx, `
				INSERT INTO node_description_changes (node_id, author_subject, old_description, new_description, created_at)
				VALUES ($1,$2,$3,$4,$5)`, id, subject, previous, req.Description, updated.UpdatedAt); err != nil {
				return err
			}
		}

		changedFields := make([]string, 0, 4)
		if oldTitle != updated.Title {
			changedFields = append(changedFields, "title")
		}
		if previous != updated.Description {
			changedFields = append(changedFields, "description")
		}
		if !sameTimePtr(timePtr(oldDue), updated.DueAt) {
			changedFields = append(changedFields, "dueAt")
		}
		if !slices.Equal(nonNilStrings(oldTags), updated.Tags) {
			changedFields = append(changedFields, "tags")
		}
		if len(changedFields) > 0 {
			evID, err := events.RecordTx(ctx, tx, events.Input{
				Kind:         events.KindNodeUpdated,
				NodeID:       &id,
				ActorSubject: subject,
				Payload: map[string]any{
					"title":         updated.Title,
					"changedFields": changedFields,
				},
			})
			if err != nil {
				return err
			}
			eventIDs = append(eventIDs, evID)
		}
		if !oldCompleted.Valid && updated.CompletedAt != nil {
			evID, err := events.RecordTx(ctx, tx, events.Input{
				Kind:         events.KindNodeCompleted,
				NodeID:       &id,
				ActorSubject: subject,
				Payload: map[string]any{
					"title":       updated.Title,
					"completedAt": updated.CompletedAt,
				},
			})
			if err != nil {
				return err
			}
			eventIDs = append(eventIDs, evID)
		} else if oldCompleted.Valid && updated.CompletedAt == nil {
			evID, err := events.RecordTx(ctx, tx, events.Input{
				Kind:         events.KindNodeReopened,
				NodeID:       &id,
				ActorSubject: subject,
				Payload: map[string]any{
					"title": updated.Title,
				},
			})
			if err != nil {
				return err
			}
			eventIDs = append(eventIDs, evID)
		}
		return nil
	})
	if err != nil {
		return Node{}, err
	}
	r.broker.Publish(eventIDs...)
	return scan.value(), nil
}

func sameParent(a, b *int64) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func sameTimePtr(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// Move changes only the subtree root's parent and appends it to the new siblings.
func (r *Repository) Move(ctx context.Context, id int64, parentID *int64, actor string, authorize func(pgx.Tx) error) error {
	var eventID int64
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		// Serialize tree writes so concurrent moves cannot create a cycle between
		// validation and update, and destination ordering stays consistent.
		if _, err := tx.Exec(ctx, `LOCK TABLE nodes IN SHARE ROW EXCLUSIVE MODE`); err != nil {
			return err
		}
		var (
			title    string
			previous pgtype.Int8
		)
		if err := tx.QueryRow(ctx, `SELECT title, parent_id FROM nodes WHERE id=$1`, id).Scan(&title, &previous); err != nil {
			if err == pgx.ErrNoRows {
				return apperr.NotFound("节点不存在")
			}
			return err
		}
		if err := authorize(tx); err != nil {
			return err
		}
		fromParent := int64Ptr(previous)
		if sameParent(fromParent, parentID) {
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
		if _, err := tx.Exec(ctx, `UPDATE nodes SET updated_at=clock_timestamp() WHERE id=$1 OR id=$2`, fromParent, parentID); err != nil {
			return err
		}
		evID, err := events.RecordTx(ctx, tx, events.Input{
			Kind:         events.KindNodeMoved,
			NodeID:       &id,
			ActorSubject: actor,
			Payload: map[string]any{
				"title":        title,
				"fromParentId": fromParent,
				"toParentId":   parentID,
			},
		})
		if err != nil {
			return err
		}
		eventID = evID
		return nil
	})
	if err != nil {
		return err
	}
	if eventID > 0 {
		r.broker.Publish(eventID)
	}
	return nil
}

// Delete removes a node; PostgreSQL cascades to the whole subtree.
func (r *Repository) Delete(ctx context.Context, id int64, actor string) error {
	var eventID int64
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		var (
			title    string
			parentID pgtype.Int8
		)
		if err := tx.QueryRow(ctx, `SELECT title, parent_id FROM nodes WHERE id=$1`, id).Scan(&title, &parentID); err != nil {
			if err == pgx.ErrNoRows {
				return apperr.NotFound("节点不存在")
			}
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM nodes WHERE id=$1`, id); err != nil {
			return err
		}
		evID, err := events.RecordTx(ctx, tx, events.Input{
			Kind:         events.KindNodeDeleted,
			NodeID:       nil,
			ActorSubject: actor,
			Payload: map[string]any{
				"deletedNodeId": id,
				"title":         title,
				"parentId":      int64Ptr(parentID),
			},
		})
		if err != nil {
			return err
		}
		eventID = evID
		return nil
	})
	if err != nil {
		return err
	}
	r.broker.Publish(eventID)
	return nil
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
		current, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
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
