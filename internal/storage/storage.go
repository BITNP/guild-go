// Package storage stores comment images in a private S3-compatible bucket.
package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// MaxAttachments is the maximum number of images per comment.
	MaxAttachments = 4
	// MaxAttachmentBytes is the maximum size of one image.
	MaxAttachmentBytes = 5 << 20
	// S3Region and path-style addressing are fixed for the shared RustFS/S3 service.
	s3Region         = "us-east-1"
	s3ForcePathStyle = true
)

// ErrNotFound is returned when an object does not exist.
var ErrNotFound = errors.New("object not found")

// Store is the object storage abstraction used by comments.
type Store interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, string, int64, error)
	Delete(ctx context.Context, key string) error
}

// NewCommentKey returns an unguessable object key under the guild-go prefix.
func NewCommentKey(now time.Time) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return fmt.Sprintf("guild-go/comments/%s/%x", now.UTC().Format("2006/01"), raw), nil
}

// DetectImageType validates an allowed image and returns its media type.
func DetectImageType(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("图片为空")
	}
	if len(data) > MaxAttachmentBytes {
		return "", fmt.Errorf("单张图片不能超过 5MB")
	}
	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return contentType, nil
	default:
		return "", fmt.Errorf("只支持 JPEG、PNG、WebP、GIF 图片")
	}
}

// MemoryStore is an in-memory Store used by tests.
type MemoryStore struct {
	mu      sync.RWMutex
	objects map[string]memoryObject
}

type memoryObject struct {
	data        []byte
	contentType string
}

// NewMemoryStore creates an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{objects: make(map[string]memoryObject)}
}

// Put stores an object.
func (m *MemoryStore) Put(_ context.Context, key string, data []byte, contentType string) error {
	m.mu.Lock()
	m.objects[key] = memoryObject{data: append([]byte(nil), data...), contentType: contentType}
	m.mu.Unlock()
	return nil
}

// Get returns an object.
func (m *MemoryStore) Get(_ context.Context, key string) (io.ReadCloser, string, int64, error) {
	m.mu.RLock()
	object, ok := m.objects[key]
	m.mu.RUnlock()
	if !ok {
		return nil, "", 0, ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(object.data)), object.contentType, int64(len(object.data)), nil
}

// Delete removes an object.
func (m *MemoryStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.objects, key)
	m.mu.Unlock()
	return nil
}

// Keys returns stored keys for assertions in tests.
func (m *MemoryStore) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.objects))
	for key := range m.objects {
		keys = append(keys, key)
	}
	return keys
}

func normalizeContentType(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
}
