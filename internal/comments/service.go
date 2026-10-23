package comments

import (
	"context"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"git.bitnp.net/BITNP/guild-go/internal/access"
	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/storage"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

const (
	defaultPageSize = 30
	maxPageSize     = 100
	maxBodyRunes    = 5000
)

// Service implements comment, reaction and attachment use cases.
type Service struct {
	repo   *Repository
	users  *users.Service
	access *access.Checker
	store  storage.Store
}

// NewService creates a comment service.
func NewService(repo *Repository, userService *users.Service, checker *access.Checker, store storage.Store) *Service {
	return &Service{repo: repo, users: userService, access: checker, store: store}
}

func (s *Service) requireCommenter(ctx context.Context, session auth.Session, nodeID int64) error {
	exists, err := s.access.NodeExists(ctx, nodeID)
	if err != nil {
		return err
	}
	if !exists {
		return apperr.NotFound("节点不存在")
	}
	if !session.Permissions.SystemAdmin {
		member, err := s.access.IsEffectiveMember(ctx, nodeID, session.Subject)
		if err != nil {
			return err
		}
		if !member {
			return apperr.Forbidden("只有节点有效成员可以评论")
		}
	}
	return nil
}

// Create publishes a comment using previously uploaded images.
func (s *Service) Create(ctx context.Context, session auth.Session, nodeID int64, body string, attachmentIDs []int64) (Comment, error) {
	if err := s.requireCommenter(ctx, session, nodeID); err != nil {
		return Comment{}, err
	}
	body = strings.TrimSpace(body)
	if utf8.RuneCountInString(body) > maxBodyRunes {
		return Comment{}, apperr.BadRequest("评论不能超过 5000 字")
	}
	if len(attachmentIDs) > storage.MaxAttachments {
		return Comment{}, apperr.BadRequest("每条评论最多 4 张图片")
	}
	if body == "" && len(attachmentIDs) == 0 {
		return Comment{}, apperr.BadRequest("评论内容不能为空")
	}
	commentID, err := s.repo.Create(ctx, nodeID, session.Subject, body, attachmentIDs)
	if err != nil {
		return Comment{}, err
	}
	return s.one(ctx, session, commentID)
}

// List returns a page of comments for a node.
func (s *Service) List(ctx context.Context, session auth.Session, nodeID int64, before *int64, limit int) (List, error) {
	exists, err := s.access.NodeExists(ctx, nodeID)
	if err != nil {
		return List{}, err
	}
	if !exists {
		return List{}, apperr.NotFound("节点不存在")
	}
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	rows, hasMore, err := s.repo.List(ctx, nodeID, before, limit)
	if err != nil {
		return List{}, err
	}
	items, err := s.assemble(ctx, session, rows)
	if err != nil {
		return List{}, err
	}
	var nextBefore *int64
	if hasMore && len(items) > 0 {
		value := items[len(items)-1].ID
		nextBefore = &value
	}
	if items == nil {
		items = []Comment{}
	}
	return List{Items: items, HasMore: hasMore, NextBefore: nextBefore}, nil
}

// Delete removes a comment authored by the caller, or any comment for a
// system administrator.
func (s *Service) Delete(ctx context.Context, session auth.Session, commentID int64) error {
	row, err := s.repo.Get(ctx, commentID)
	if err == pgx.ErrNoRows {
		return apperr.NotFound("评论不存在")
	}
	if err != nil {
		return err
	}
	if row.Author != session.Subject && !session.Permissions.SystemAdmin {
		return apperr.Forbidden("只能删除自己的评论")
	}
	keys, err := s.repo.Delete(ctx, commentID)
	if err != nil {
		return err
	}
	s.deleteObjects(keys)
	return nil
}

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
		return nil, apperr.NotFound("评论不存在")
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
		return nil, apperr.NotFound("评论不存在")
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

// GetAttachment returns the stream for one image.
func (s *Service) GetAttachment(ctx context.Context, id int64) (io.ReadCloser, string, int64, error) {
	row, err := s.repo.AttachmentByID(ctx, id)
	if err == pgx.ErrNoRows {
		return nil, "", 0, apperr.NotFound("附件不存在")
	}
	if err != nil {
		return nil, "", 0, err
	}
	return s.store.Get(ctx, row.ObjectKey)
}

func (s *Service) one(ctx context.Context, session auth.Session, commentID int64) (Comment, error) {
	row, err := s.repo.Get(ctx, commentID)
	if err != nil {
		return Comment{}, err
	}
	items, err := s.assemble(ctx, session, []CommentRow{row})
	if err != nil {
		return Comment{}, err
	}
	if len(items) == 0 {
		return Comment{}, apperr.NotFound("评论不存在")
	}
	return items[0], nil
}

func (s *Service) assemble(ctx context.Context, session auth.Session, rows []CommentRow) ([]Comment, error) {
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
		author, ok := authors[row.Author]
		if !ok {
			author = users.Summary{Subject: row.Author, UserName: row.Author, Positions: []users.Position{}}
		}
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
