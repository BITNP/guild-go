package storage

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestDetectImageType(t *testing.T) {
	contentType, err := DetectImageType(testPNG(t))
	if err != nil || contentType != "image/png" {
		t.Fatalf("PNG rejected: %s %v", contentType, err)
	}
	for name, data := range map[string][]byte{
		"empty": {},
		"text":  []byte("<svg></svg>"),
	} {
		if _, err := DetectImageType(data); err == nil {
			t.Fatalf("%s should be rejected", name)
		}
	}
}

func TestMemoryStore(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	if err := store.Put(ctx, "key", []byte("data"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	reader, contentType, size, err := store.Get(ctx, "key")
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "text/plain" || size != 4 {
		t.Fatalf("metadata = %s %d", contentType, size)
	}
	var buffer bytes.Buffer
	_, _ = buffer.ReadFrom(reader)
	_ = reader.Close()
	if buffer.String() != "data" {
		t.Fatalf("data = %q", buffer.String())
	}
	if err := store.Delete(ctx, "key"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := store.Get(ctx, "key"); err != ErrNotFound {
		t.Fatalf("deleted object error = %v", err)
	}
}
