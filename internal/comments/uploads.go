package comments

import (
	"context"
	"errors"
	"fmt"
	"io"
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
		storage.DeleteObjects(s.store, []string{key})
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
	storage.DeleteObjects(s.store, []string{key})
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
			storage.DeleteObjects(s.store, keys)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
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
	reader, contentType, size, err := s.store.Get(ctx, row.ObjectKey)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, "", 0, apperr.NotFound("附件不存在")
	}
	return reader, contentType, size, err
}
