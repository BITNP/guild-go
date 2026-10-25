// Package comments owns node progress, linear comments, emoji reactions and image attachments.
package comments

import (
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/users"
)

// Reaction is an aggregated emoji reaction for one comment.
type Reaction struct {
	Emoji       string `json:"emoji"`
	Count       int    `json:"count"`
	ReactedByMe bool   `json:"reactedByMe"`
}

// Attachment is a stored comment image.
type Attachment struct {
	ID          int64  `json:"id"`
	URL         string `json:"url"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
}

// Comment is a full comment view.
type Comment struct {
	ID          int64         `json:"id"`
	NodeID      int64         `json:"nodeId"`
	Author      users.Summary `json:"author"`
	Body        string        `json:"body"`
	CreatedAt   time.Time     `json:"createdAt"`
	Attachments []Attachment  `json:"attachments"`
	Reactions   []Reaction    `json:"reactions"`
	CanDelete   bool          `json:"canDelete"`
}

// Preview is the compact latest-comment shape used by child rows and overview.
type Preview struct {
	ID              int64         `json:"id"`
	Author          users.Summary `json:"author"`
	BodyPreview     string        `json:"bodyPreview"`
	CreatedAt       time.Time     `json:"createdAt"`
	ReactionSummary []Reaction    `json:"reactionSummary"`
	AttachmentCount int           `json:"attachmentCount"`
}

// List is a page of comments.
type List struct {
	Items      []Comment `json:"items"`
	HasMore    bool      `json:"hasMore"`
	NextBefore *int64    `json:"nextBefore"`
}

// Recent is one item in the site-wide recent-comment feed.
type Recent struct {
	ID              int64         `json:"id"`
	NodeID          int64         `json:"nodeId"`
	NodeTitle       string        `json:"nodeTitle"`
	RootTitle       string        `json:"rootTitle"`
	Author          users.Summary `json:"author"`
	BodyPreview     string        `json:"bodyPreview"`
	CreatedAt       time.Time     `json:"createdAt"`
	ReactionSummary []Reaction    `json:"reactionSummary"`
}

// Emoji keys accepted by the API.
var EmojiKeys = []string{
	"thumbs_up", "thumbs_down", "laugh", "hooray",
	"confused", "heart", "rocket", "eyes",
}

// ValidEmoji reports whether key is in the fixed reaction set.
func ValidEmoji(key string) bool {
	for _, emoji := range EmojiKeys {
		if emoji == key {
			return true
		}
	}
	return false
}
