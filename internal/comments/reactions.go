package comments

import (
	"context"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

// ToggleReaction toggles one emoji. Every admitted user may react.
func (s *Service) ToggleReaction(ctx context.Context, commentID int64, subject, emoji string) ([]Reaction, error) {
	if !ValidEmoji(emoji) {
		return nil, apperr.BadRequest("不支持的表情")
	}
	exists, err := s.repo.Exists(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperr.NotFound("进展不存在")
	}
	return s.repo.ToggleReaction(ctx, commentID, subject, emoji)
}

// ReactionUsers lists the people who used an emoji. Every admitted user may view them.
func (s *Service) ReactionUsers(ctx context.Context, session auth.Session, commentID int64, emoji string) ([]users.Summary, error) {
	if !ValidEmoji(emoji) {
		return nil, apperr.BadRequest("不支持的表情")
	}
	exists, err := s.repo.Exists(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperr.NotFound("进展不存在")
	}
	subjects, err := s.repo.ReactionSubjects(ctx, commentID, emoji)
	if err != nil {
		return nil, err
	}
	result := make([]users.Summary, 0, len(subjects))
	for _, subject := range subjects {
		result = append(result, s.users.SummaryFor(ctx, session, subject))
	}
	return result, nil
}
