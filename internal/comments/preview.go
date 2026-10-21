package comments

import (
	"strings"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/users"
)

// BuildPreview creates a compact latest-comment view.
func BuildPreview(id int64, author users.Summary, body string, createdAt time.Time, reactions []Reaction, attachmentCount int) Preview {
	if reactions == nil {
		reactions = []Reaction{}
	}
	return Preview{
		ID:              id,
		Author:          author,
		BodyPreview:     TextPreview(body, 200),
		CreatedAt:       createdAt,
		ReactionSummary: reactions,
		AttachmentCount: attachmentCount,
	}
}

// TextPreview truncates text to max runes, preserving line breaks.
func TextPreview(text string, max int) string {
	text = strings.TrimSpace(text)
	if max <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return strings.TrimSpace(string(runes[:max])) + "…"
}
