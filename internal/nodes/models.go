// Package nodes owns the arbitrary node tree, members, permissions and ordering.
package nodes

import (
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

// Node is a stored tree node.
type Node struct {
	ID          int64      `json:"id"`
	ParentID    *int64     `json:"parentId"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueAt       *time.Time `json:"dueAt"`
	CompletedAt *time.Time `json:"completedAt"`
	Tags        []string   `json:"tags"`
	SortOrder   int64      `json:"sortOrder"`
	CreatedBy   string     `json:"createdBySubject"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// Summary is a compact node row used by the tree, overview and child lists.
type Summary struct {
	ID              int64      `json:"id"`
	ParentID        *int64     `json:"parentId"`
	Title           string     `json:"title"`
	DueAt           *time.Time `json:"dueAt"`
	CompletedAt     *time.Time `json:"completedAt"`
	Tags            []string   `json:"tags"`
	SortOrder       int64      `json:"sortOrder"`
	ChildCount      int        `json:"childCount"`
	CommentCount    int        `json:"commentCount"`
	LatestCommentAt *time.Time `json:"latestCommentAt"`
	MyRelation      *string    `json:"myRelation"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// Breadcrumb is one ancestor in the path from the root.
type Breadcrumb struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// Child is a direct child with its latest comment.
type Child struct {
	Summary
	LatestComment *comments.Preview `json:"latestComment"`
}

// Permissions are the viewer's effective actions on a node.
type Permissions struct {
	CanEdit            bool `json:"canEdit"`
	CanDelete          bool `json:"canDelete"`
	CanComment         bool `json:"canComment"`
	CanManageMembers   bool `json:"canManageMembers"`
	CanAddChild        bool `json:"canAddChild"`
	CanReorderChildren bool `json:"canReorderChildren"`
}

// Detail is the full node view.
type Detail struct {
	Node
	CreatedByUser          users.Summary   `json:"createdBy"`
	Breadcrumb             []Breadcrumb    `json:"breadcrumb"`
	DirectLeaders          []users.Summary `json:"directLeaders"`
	DirectMembers          []users.Summary `json:"directMembers"`
	InheritedLeaderCount   int             `json:"inheritedLeaderCount"`
	InheritedMemberCount   int             `json:"inheritedMemberCount"`
	EffectiveMemberCount   int             `json:"effectiveMemberCount"`
	SubtreeNodeCount       int             `json:"subtreeNodeCount"`
	SubtreeCommentCount    int             `json:"subtreeCommentCount"`
	SubtreeAttachmentCount int             `json:"subtreeAttachmentCount"`
	Permissions            Permissions     `json:"permissions"`
	Children               []Child         `json:"children"`
}

// Source describes where an inherited membership comes from.
type Source struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// EffectiveMember is one user in the effective member/leader list.
type EffectiveMember struct {
	User    users.Summary `json:"user"`
	Direct  bool          `json:"direct"`
	Sources []Source      `json:"sources"`
}

// EffectiveMembers is the expanded member view.
type EffectiveMembers struct {
	Leaders []EffectiveMember `json:"leaders"`
	Members []EffectiveMember `json:"members"`
}

// CreateRequest is the create-node payload.
type CreateRequest struct {
	ParentID       *int64     `json:"parentId"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	DueAt          *time.Time `json:"dueAt"`
	Tags           []string   `json:"tags"`
	LeaderSubjects []string   `json:"leaderSubjects"`
	MemberSubjects []string   `json:"memberSubjects"`
}

// UpdateRequest replaces the editable node fields.
type UpdateRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueAt       *time.Time `json:"dueAt"`
	Tags        []string   `json:"tags"`
	Completed   bool       `json:"completed"`
}

// OrderRequest reorders all direct children of one parent (or all roots).
type OrderRequest struct {
	ParentID *int64  `json:"parentId"`
	IDs      []int64 `json:"ids"`
}

// MembersRequest replaces the direct leaders and members.
type MembersRequest struct {
	Leaders []string `json:"leaders"`
	Members []string `json:"members"`
}

// Overview is the landing page payload.
type Overview struct {
	Roots          []RootSummary     `json:"roots"`
	RecentComments []comments.Recent `json:"recentComments"`
}

// RootSummary is a root node with leaders and latest comment.
type RootSummary struct {
	Summary
	Leaders       []users.Summary   `json:"leaders"`
	LatestComment *comments.Preview `json:"latestComment"`
}
