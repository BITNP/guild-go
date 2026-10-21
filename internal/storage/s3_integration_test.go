package storage

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestS3RoundTrip(t *testing.T) {
	endpoint := os.Getenv("S3_ENDPOINT")
	bucket := os.Getenv("S3_BUCKET")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		t.Skip("S3_* environment variables are not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, err := NewS3(ctx, endpoint, bucket, accessKey, secretKey)
	if err != nil {
		t.Fatal(err)
	}
	key, err := NewCommentKey(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	key = "guild-go-test/" + key
	data := []byte("guild-go s3 test")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_ = store.Delete(cleanupCtx, key)
	})
	if err := store.Put(ctx, key, data, "text/plain"); err != nil {
		t.Fatal(err)
	}
	reader, contentType, size, err := store.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	_, _ = buffer.ReadFrom(reader)
	_ = reader.Close()
	if !bytes.Equal(buffer.Bytes(), data) || contentType != "text/plain" || size != int64(len(data)) {
		t.Fatalf("roundtrip mismatch: %q %s %d", buffer.String(), contentType, size)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := store.Get(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted object error = %v", err)
	}
}