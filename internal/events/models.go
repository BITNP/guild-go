// Package events persists append-only domain events and broadcasts wake-up
// notifications to internal service consumers.
package events

import (
	"encoding/json"
	"time"
)

const (
	KindNodeCreated    = "node.created"
	KindNodeUpdated    = "node.updated"
	KindNodeCompleted  = "node.completed"
	KindNodeReopened   = "node.reopened"
	KindNodeMoved      = "node.moved"
	KindNodeDeleted    = "node.deleted"
	KindMemberAssigned = "member.assigned"
	KindMemberRemoved  = "member.removed"
	KindCommentCreated = "comment.created"
)

// Event is a committed domain event stored in PostgreSQL.
type Event struct {
	ID           int64           `json:"id"`
	Kind         string          `json:"kind"`
	NodeID       *int64          `json:"nodeId"`
	ActorSubject string          `json:"actorSubject"`
	Payload      json.RawMessage `json:"payload"`
	CreatedAt    time.Time       `json:"createdAt"`
}

// Input describes one event to append inside an existing database transaction.
type Input struct {
	Kind         string
	NodeID       *int64
	ActorSubject string
	Payload      any
}

// ListResponse is one cursor-paginated batch of events plus the current stream head.
type ListResponse struct {
	Items      []Event   `json:"items"`
	HasMore    bool      `json:"hasMore"`
	NextAfter  int64     `json:"nextAfter"`
	LatestID   int64     `json:"latestId"`
	ServerTime time.Time `json:"serverTime"`
}

// DueNodesQuery configures a due-node scan for reminders and deadline inspections.
type DueNodesQuery struct {
	Since          *time.Time
	Advance        time.Duration
	DueAfter       *time.Time
	DueBefore      *time.Time
	IncludeOverdue bool
	Limit          int
}

// DueNodeItem describes one open node matching a deadline query.
type DueNodeItem struct {
	ID       int64     `json:"id"`
	ParentID *int64    `json:"parentId"`
	Title    string    `json:"title"`
	DueAt    time.Time `json:"dueAt"`
	Leaders  []string  `json:"leaders"`
	Members  []string  `json:"members"`
	Overdue  bool      `json:"overdue"`
}

// DueNodesResponse returns open due nodes paired with the authoritative database time.
type DueNodesResponse struct {
	Items      []DueNodeItem `json:"items"`
	ServerTime time.Time     `json:"serverTime"`
}

// ActivitySummaryNode is a compact node row included in periodic activity summaries.
type ActivitySummaryNode struct {
	ID          int64      `json:"id"`
	ParentID    *int64     `json:"parentId"`
	Title       string     `json:"title"`
	DueAt       *time.Time `json:"dueAt"`
	CompletedAt *time.Time `json:"completedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// ActivitySummaryComment is a compact comment excerpt included in periodic activity summaries.
type ActivitySummaryComment struct {
	ID            int64     `json:"id"`
	NodeID        int64     `json:"nodeId"`
	NodeTitle     string    `json:"nodeTitle"`
	AuthorSubject string    `json:"authorSubject"`
	BodyPreview   string    `json:"bodyPreview"`
	CreatedAt     time.Time `json:"createdAt"`
}

// ActivitySummary aggregates guild activity in [Start, End) plus live open/overdue counts.
type ActivitySummary struct {
	Start          time.Time                `json:"start"`
	End            time.Time                `json:"end"`
	SnapshotAt     time.Time                `json:"snapshotAt"`
	CreatedCount   int                      `json:"createdCount"`
	CompletedCount int                      `json:"completedCount"`
	CommentCount   int                      `json:"commentCount"`
	OpenCount      int                      `json:"openCount"`
	OverdueCount   int                      `json:"overdueCount"`
	CreatedNodes   []ActivitySummaryNode    `json:"createdNodes"`
	CompletedNodes []ActivitySummaryNode    `json:"completedNodes"`
	RecentComments []ActivitySummaryComment `json:"recentComments"`
	OverdueNodes   []ActivitySummaryNode    `json:"overdueNodes"`
}
