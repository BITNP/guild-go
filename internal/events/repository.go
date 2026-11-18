package events

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const eventsLockKey int64 = 745301982341

// Repository queries and appends rows in the PostgreSQL events table and
// provides service-level deadline/activity read models.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs an events repository backed by pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// RecordTx inserts one domain event inside an active database transaction.
// Callers must run node/member/comment writes before calling RecordTx so the
// advisory lock preserves monotonic commit order without introducing lock inversions.
func RecordTx(ctx context.Context, tx pgx.Tx, input Input) (int64, error) {
	var raw json.RawMessage
	switch v := input.Payload.(type) {
	case nil:
		raw = json.RawMessage(`{}`)
	case json.RawMessage:
		if len(v) == 0 {
			raw = json.RawMessage(`{}`)
		} else {
			raw = v
		}
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return 0, err
		}
		raw = encoded
	}

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, eventsLockKey); err != nil {
		return 0, err
	}

	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO events (kind, node_id, actor_subject, payload)
		VALUES ($1, $2, $3, $4::jsonb)
		RETURNING id`,
		input.Kind, input.NodeID, input.ActorSubject, string(raw),
	).Scan(&id)
	return id, err
}

// LatestID returns the highest committed event ID, or 0 when the log is empty.
func (r *Repository) LatestID(ctx context.Context) (int64, error) {
	var latest int64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(id), 0) FROM events`).Scan(&latest)
	return latest, err
}

// ListAfter returns committed events with id > after in ascending cursor order.
func (r *Repository) ListAfter(ctx context.Context, after int64, limit int) (ListResponse, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, kind, node_id, actor_subject, payload, created_at
		FROM events
		WHERE id > $1
		ORDER BY id ASC
		LIMIT $2`,
		after, limit+1,
	)
	if err != nil {
		return ListResponse{}, err
	}
	defer rows.Close()

	items := make([]Event, 0)
	for rows.Next() {
		var (
			ev      Event
			nodeID  pgtype.Int8
			payload []byte
		)
		if err := rows.Scan(&ev.ID, &ev.Kind, &nodeID, &ev.ActorSubject, &payload, &ev.CreatedAt); err != nil {
			return ListResponse{}, err
		}
		ev.NodeID = int64Ptr(nodeID)
		if len(payload) == 0 {
			ev.Payload = json.RawMessage(`{}`)
		} else {
			ev.Payload = json.RawMessage(payload)
		}
		items = append(items, ev)
	}
	if err := rows.Err(); err != nil {
		return ListResponse{}, err
	}

	hasMore := false
	if len(items) > limit {
		hasMore = true
		items = items[:limit]
	}

	nextAfter := after
	if len(items) > 0 {
		nextAfter = items[len(items)-1].ID
	}

	var latestID int64
	var serverTime time.Time
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(id), 0), clock_timestamp() FROM events`).Scan(&latestID, &serverTime); err != nil {
		return ListResponse{}, err
	}

	return ListResponse{
		Items:      items,
		HasMore:    hasMore,
		NextAfter:  nextAfter,
		LatestID:   latestID,
		ServerTime: serverTime,
	}, nil
}

// ListDueNodes returns open nodes whose deadlines fall inside the requested window.
func (r *Repository) ListDueNodes(ctx context.Context, query DueNodesQuery) (DueNodesResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	var serverTime time.Time
	if err := r.pool.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&serverTime); err != nil {
		return DueNodesResponse{}, err
	}

	after := query.DueAfter
	before := query.DueBefore
	advance := max(query.Advance, 0)

	if query.Since != nil {
		windowStart := query.Since.Add(advance)
		windowEnd := serverTime.Add(advance)
		if !windowEnd.After(windowStart) {
			return DueNodesResponse{
				Items:      []DueNodeItem{},
				ServerTime: serverTime,
			}, nil
		}
		after = &windowStart
		before = &windowEnd
	} else if before == nil && advance > 0 {
		windowEnd := serverTime.Add(advance)
		before = &windowEnd
	}

	var afterArg, beforeArg pgtype.Timestamptz
	if after != nil {
		afterArg = pgtype.Timestamptz{Time: *after, Valid: true}
	}
	if before != nil {
		beforeArg = pgtype.Timestamptz{Time: *before, Valid: true}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.parent_id, n.title, n.due_at,
		       (n.due_at <= $4) AS overdue,
		       COALESCE(ARRAY(
		           SELECT m.subject FROM node_members m
		           WHERE m.node_id = n.id AND m.role = 'leader'
		           ORDER BY m.created_at, m.subject
		       ), '{}') AS leaders,
		       COALESCE(ARRAY(
		           SELECT m.subject FROM node_members m
		           WHERE m.node_id = n.id AND m.role = 'member'
		           ORDER BY m.created_at, m.subject
		       ), '{}') AS members
		FROM nodes n
		WHERE n.completed_at IS NULL
		  AND n.due_at IS NOT NULL
		  AND ($1::timestamptz IS NULL OR n.due_at >= $1)
		  AND ($2::timestamptz IS NULL OR n.due_at < $2)
		  AND ($3::boolean OR n.due_at > $4)
		ORDER BY n.due_at ASC, n.id ASC
		LIMIT $5`,
		afterArg, beforeArg, query.IncludeOverdue, serverTime, limit,
	)
	if err != nil {
		return DueNodesResponse{}, err
	}
	defer rows.Close()

	items := make([]DueNodeItem, 0)
	for rows.Next() {
		var (
			item     DueNodeItem
			parentID pgtype.Int8
		)
		if err := rows.Scan(&item.ID, &parentID, &item.Title, &item.DueAt, &item.Overdue, &item.Leaders, &item.Members); err != nil {
			return DueNodesResponse{}, err
		}
		item.ParentID = int64Ptr(parentID)
		if item.Leaders == nil {
			item.Leaders = []string{}
		}
		if item.Members == nil {
			item.Members = []string{}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return DueNodesResponse{}, err
	}

	return DueNodesResponse{
		Items:      items,
		ServerTime: serverTime,
	}, nil
}

// ActivitySummary returns aggregated counts and highlights for [start, end) plus live backlog state.
func (r *Repository) ActivitySummary(ctx context.Context, start, end time.Time) (ActivitySummary, error) {
	summary := ActivitySummary{
		Start:          start,
		End:            end,
		CreatedNodes:   []ActivitySummaryNode{},
		CompletedNodes: []ActivitySummaryNode{},
		RecentComments: []ActivitySummaryComment{},
		OverdueNodes:   []ActivitySummaryNode{},
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT
			clock_timestamp(),
			(SELECT count(*)::int FROM nodes WHERE created_at >= $1 AND created_at < $2),
			(SELECT count(*)::int FROM nodes WHERE completed_at IS NOT NULL AND completed_at >= $1 AND completed_at < $2),
			(SELECT count(*)::int FROM comments WHERE created_at >= $1 AND created_at < $2),
			(SELECT count(*)::int FROM nodes WHERE completed_at IS NULL),
			(SELECT count(*)::int FROM nodes WHERE completed_at IS NULL AND due_at IS NOT NULL AND due_at < clock_timestamp())`,
		start, end,
	).Scan(
		&summary.SnapshotAt,
		&summary.CreatedCount,
		&summary.CompletedCount,
		&summary.CommentCount,
		&summary.OpenCount,
		&summary.OverdueCount,
	); err != nil {
		return ActivitySummary{}, err
	}

	createdNodes, err := r.querySummaryNodes(ctx, `
		SELECT id, parent_id, title, due_at, completed_at, created_at
		FROM nodes
		WHERE created_at >= $1 AND created_at < $2
		ORDER BY id ASC
		LIMIT 200`, start, end)
	if err != nil {
		return ActivitySummary{}, err
	}
	summary.CreatedNodes = createdNodes

	completedNodes, err := r.querySummaryNodes(ctx, `
		SELECT id, parent_id, title, due_at, completed_at, created_at
		FROM nodes
		WHERE completed_at IS NOT NULL AND completed_at >= $1 AND completed_at < $2
		ORDER BY completed_at ASC, id ASC
		LIMIT 200`, start, end)
	if err != nil {
		return ActivitySummary{}, err
	}
	summary.CompletedNodes = completedNodes

	overdueNodes, err := r.querySummaryNodes(ctx, `
		SELECT id, parent_id, title, due_at, completed_at, created_at
		FROM nodes
		WHERE completed_at IS NULL AND due_at IS NOT NULL AND due_at < $1
		ORDER BY due_at ASC, id ASC
		LIMIT 200`, summary.SnapshotAt)
	if err != nil {
		return ActivitySummary{}, err
	}
	summary.OverdueNodes = overdueNodes

	commentRows, err := r.pool.Query(ctx, `
		SELECT c.id, c.node_id, n.title, c.author_subject, c.body, c.created_at
		FROM comments c
		JOIN nodes n ON n.id = c.node_id
		WHERE c.created_at >= $1 AND c.created_at < $2
		ORDER BY c.created_at ASC, c.id ASC
		LIMIT 100`, start, end)
	if err != nil {
		return ActivitySummary{}, err
	}
	defer commentRows.Close()

	for commentRows.Next() {
		var (
			item ActivitySummaryComment
			body string
		)
		if err := commentRows.Scan(&item.ID, &item.NodeID, &item.NodeTitle, &item.AuthorSubject, &body, &item.CreatedAt); err != nil {
			return ActivitySummary{}, err
		}
		item.BodyPreview = truncateText(body, 200)
		summary.RecentComments = append(summary.RecentComments, item)
	}
	if err := commentRows.Err(); err != nil {
		return ActivitySummary{}, err
	}

	return summary, nil
}

func (r *Repository) querySummaryNodes(ctx context.Context, sql string, args ...any) ([]ActivitySummaryNode, error) {
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make([]ActivitySummaryNode, 0)
	for rows.Next() {
		var (
			item        ActivitySummaryNode
			parentID    pgtype.Int8
			dueAt       pgtype.Timestamptz
			completedAt pgtype.Timestamptz
		)
		if err := rows.Scan(&item.ID, &parentID, &item.Title, &dueAt, &completedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.ParentID = int64Ptr(parentID)
		item.DueAt = timePtr(dueAt)
		item.CompletedAt = timePtr(completedAt)
		nodes = append(nodes, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}

func int64Ptr(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func truncateText(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "…"
}
