package comments

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/storage"
)

// UploadImage stores one private image for a future comment and returns its id.
func (s *Service) UploadImage(ctx context.Context, session auth.Session, nodeID int64, data []byte) (int64, error) {
	if err := s.requireCommenter(ctx, session, nodeID); err != nil {
		return 0, err
	}
	contentType, err := storage.DetectImageType(data)
	if err != nil {
		return 0, apperr.BadRequest(err.Error())
	}
	key, err := storage.NewCommentKey(time.Now())
	if err != nil {
		return 0, err
	}
	uploadCtx, cancel := context.WithTimeout(ctx, storage.TimeoutForUpload)
	err = s.store.Put(uploadCtx, key, data, contentType)
	cancel()
	if err != nil {
		return 0, fmt.Errorf("store comment image: %w", err)
	}
	id, err := s.repo.AddUpload(ctx, nodeID, session.Subject, key, contentType, int64(len(data)))
	if err != nil {
		s.deleteObjects([]string{key})
		return 0, err
	}
	return id, nil
}

// DeleteUpload discards one unpublished image belonging to the caller.
func (s *Service) DeleteUpload(ctx context.Context, session auth.Session, id int64) error {
	key, err := s.repo.DeleteUpload(ctx, id, session.Subject)
	if err == pgx.ErrNoRows {
		return apperr.NotFound("待发表的图片不存在")
	}
	if err != nil {
		return err
	}
	s.deleteObjects([]string{key})
	return nil
}

// StartUploadCleanup expires unpublished images at startup and hourly thereafter.
func (s *Service) StartUploadCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		keys, err := s.repo.DeleteExpiredUploads(ctx)
		if err != nil {
			if ctx.Err() == nil {
				slog.Warn("clean up unpublished comment images failed", "error", err)
			}
		} else {
			s.deleteObjects(keys)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) deleteObjects(keys []string) {
	for _, key := range keys {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := s.store.Delete(ctx, key); err != nil {
			slog.Warn("delete comment attachment failed", "key", key, "error", err)
		}
		cancel()
	}
}
