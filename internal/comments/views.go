package comments

import (
	"context"
	"fmt"

	"git.bitnp.net/BITNP/guild-go/internal/auth"
)

func (s *Service) assemble(ctx context.Context, session auth.Session, rows []commentRow) ([]Comment, error) {
	if len(rows) == 0 {
		return []Comment{}, nil
	}
	ids := make([]int64, 0, len(rows))
	subjects := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
		subjects = append(subjects, row.Author)
	}
	authors := s.users.Summaries(ctx, session, subjects)
	attachments, err := s.repo.AttachmentsForComments(ctx, ids)
	if err != nil {
		return nil, err
	}
	reactions, err := s.repo.ReactionsForComments(ctx, ids, session.Subject)
	if err != nil {
		return nil, err
	}
	result := make([]Comment, 0, len(rows))
	for _, row := range rows {
		author := authors[row.Author]
		attachmentViews := make([]Attachment, 0, len(attachments[row.ID]))
		for _, attachment := range attachments[row.ID] {
			attachmentViews = append(attachmentViews, Attachment{
				ID:          attachment.ID,
				URL:         fmt.Sprintf("/api/attachments/%d", attachment.ID),
				ContentType: attachment.ContentType,
				SizeBytes:   attachment.SizeBytes,
			})
		}
		commentReactions := reactions[row.ID]
		if commentReactions == nil {
			commentReactions = []Reaction{}
		}
		result = append(result, Comment{
			ID:          row.ID,
			NodeID:      row.NodeID,
			Author:      author,
			Body:        row.Body,
			CreatedAt:   row.CreatedAt,
			Attachments: attachmentViews,
			Reactions:   commentReactions,
			CanDelete:   row.Author == session.Subject || session.Permissions.SystemAdmin,
		})
	}
	return result, nil
}
