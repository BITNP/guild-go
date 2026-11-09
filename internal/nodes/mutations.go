package nodes

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"git.bitnp.net/BITNP/guild-go/internal/access"
	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
)

// Create inserts a root or child node.
func (s *Service) Create(ctx context.Context, session auth.Session, req CreateRequest) (Detail, error) {
	title, description, tags, err := normalizeNodeFields(req.Title, req.Description, req.Tags)
	if err != nil {
		return Detail{}, err
	}

	if req.ParentID == nil {
		if !session.Permissions.Manager {
			return Detail{}, apperr.Forbidden("只有 manager 可以创建根节点")
		}
	} else {
		if _, err := s.repo.Get(ctx, *req.ParentID); err != nil {
			return Detail{}, err
		}
		if !session.Permissions.SystemAdmin {
			allowed, err := s.access.IsEffectiveLeader(ctx, *req.ParentID, session.Subject)
			if err != nil {
				return Detail{}, err
			}
			if !allowed {
				return Detail{}, apperr.Forbidden("只有父节点负责人可以创建子节点")
			}
		}
	}

	leaders, members, err := s.normalizeMembers(ctx, req.LeaderSubjects, req.MemberSubjects, session.Subject)
	if err != nil {
		return Detail{}, err
	}

	req.Title = title
	req.Description = description
	req.Tags = tags

	id, err := s.repo.Create(ctx, req, session.Subject, leaders, members)
	if err != nil {
		return Detail{}, err
	}

	return s.Get(ctx, session, id)
}

// Update replaces editable fields.
func (s *Service) Update(ctx context.Context, session auth.Session, id int64, req UpdateRequest) (Detail, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return Detail{}, err
	}
	if err := s.requireStructural(ctx, session, id, "只有节点负责人可以编辑节点"); err != nil {
		return Detail{}, err
	}

	title, description, tags, err := normalizeNodeFields(req.Title, req.Description, req.Tags)
	if err != nil {
		return Detail{}, err
	}

	req.Title = title
	req.Description = description
	req.Tags = tags

	if _, err := s.repo.Update(ctx, id, req, session.Subject); err != nil {
		return Detail{}, err
	}

	return s.Get(ctx, session, id)
}

// Move relocates a subtree without changing its content or direct memberships.
func (s *Service) Move(ctx context.Context, session auth.Session, id int64, req MoveRequest) (Detail, error) {
	node, err := s.repo.Get(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	if err := s.requireStructural(ctx, session, id, "只有节点负责人可以移动节点"); err != nil {
		return Detail{}, err
	}

	if req.ParentID != nil && *req.ParentID <= 0 {
		return Detail{}, apperr.BadRequest("目标父节点不正确")
	}

	if sameParent(node.ParentID, req.ParentID) {
		return s.Get(ctx, session, id)
	}

	if req.ParentID == nil {
		if !session.Permissions.Manager && !session.Permissions.SystemAdmin {
			return Detail{}, apperr.Forbidden("只有 manager 或系统管理员可以移动为根节点")
		}
	} else {
		if _, err := s.repo.Get(ctx, *req.ParentID); err != nil {
			return Detail{}, err
		}
		if err := s.requireStructural(ctx, session, *req.ParentID, "只有目标父节点负责人可以移入节点"); err != nil {
			return Detail{}, err
		}
	}

	if err := s.repo.Move(ctx, id, req.ParentID, func(tx pgx.Tx) error {
		if session.Permissions.SystemAdmin {
			return nil
		}
		// A queued move may have lost inherited leadership while another move
		// committed. Recheck after acquiring the tree lock, in this transaction.
		checker := access.InTx(tx)
		ids := []int64{id}
		if req.ParentID != nil {
			ids = append(ids, *req.ParentID)
		}
		for _, nodeID := range ids {
			allowed, err := checker.IsEffectiveLeader(ctx, nodeID, session.Subject)
			if err != nil {
				return err
			}
			if !allowed {
				return apperr.Forbidden("移动权限已变化，请刷新后重试")
			}
		}
		return nil
	}); err != nil {
		return Detail{}, err
	}

	return s.Get(ctx, session, id)
}

// Delete removes a node and its whole subtree.
func (s *Service) Delete(ctx context.Context, session auth.Session, id int64) error {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return err
	}
	if err := s.requireStructural(ctx, session, id, "只有节点负责人可以删除节点"); err != nil {
		return err
	}

	keys, err := s.access.AttachmentKeysForNode(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	for _, key := range keys {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := s.store.Delete(deleteCtx, key); err != nil {
			slog.Warn("delete node attachment failed", "key", key, "error", err)
		}
		cancel()
	}

	return nil
}

// Reorder rewrites sibling order.
func (s *Service) Reorder(ctx context.Context, session auth.Session, parentID *int64, ids []int64) error {
	if parentID == nil {
		if !session.Permissions.Manager {
			return apperr.Forbidden("只有 manager 可以调整根节点顺序")
		}
		return s.repo.Reorder(ctx, nil, ids)
	}

	if _, err := s.repo.Get(ctx, *parentID); err != nil {
		return err
	}

	if err := s.requireStructural(ctx, session, *parentID, "只有父节点负责人可以调整子节点顺序"); err != nil {
		return err
	}

	return s.repo.Reorder(ctx, parentID, ids)
}
