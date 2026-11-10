package comments

import (
	"context"
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
			return apperr.Forbidden("只有节点有效成员可以发布进展")
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
		return Comment{}, apperr.BadRequest("进展正文不能超过 5000 字")
	}
	if len(attachmentIDs) > storage.MaxAttachments {
		return Comment{}, apperr.BadRequest("每条进展最多 4 张图片")
	}
	if body == "" && len(attachmentIDs) == 0 {
		return Comment{}, apperr.BadRequest("进展内容不能为空")
	}
	row, err := s.repo.Create(ctx, nodeID, session.Subject, body, attachmentIDs)
	if err != nil {
		return Comment{}, err
	}
	items, err := s.assemble(ctx, session, []commentRow{row})
	if err != nil {
		return Comment{}, err
	}
	return items[0], nil
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
	if hasMore {
		value := items[len(items)-1].ID
		nextBefore = &value
	}
	return List{Items: items, HasMore: hasMore, NextBefore: nextBefore}, nil
}

// Delete removes a comment authored by the caller, or any comment for a
// system administrator.
func (s *Service) Delete(ctx context.Context, session auth.Session, commentID int64) error {
	row, err := s.repo.Get(ctx, commentID)
	if err == pgx.ErrNoRows {
		return apperr.NotFound("进展不存在")
	}
	if err != nil {
		return err
	}
	if row.Author != session.Subject && !session.Permissions.SystemAdmin {
		return apperr.Forbidden("只能删除自己发布的进展")
	}
	keys, err := s.repo.Delete(ctx, commentID)
	if err != nil {
		return err
	}
	storage.DeleteObjects(s.store, keys)
	return nil
}
