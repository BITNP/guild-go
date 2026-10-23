package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/access"
	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/config"
	"git.bitnp.net/BITNP/guild-go/internal/httpapi"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
	"git.bitnp.net/BITNP/guild-go/internal/storage"
)

type failingUploadStore struct {
	*storage.MemoryStore
	failNext bool
}

func TestCommentImageHTTP(t *testing.T) {
	f := setup(t)
	root, err := f.nodes.Create(context.Background(), f.admin, nodes.CreateRequest{Title: "HTTP images"})
	if err != nil {
		t.Fatal(err)
	}
	sessions := auth.NewStore(time.Minute)
	id, csrf, err := sessions.Create("admin", "Admin", "", "", f.admin.Permissions)
	if err != nil {
		t.Fatal(err)
	}
	publicURL, _ := url.Parse("http://guild.example")
	authService := auth.NewService(nil, sessions, nil, publicURL, false, nil)
	handler := httpapi.New(config.Config{}, f.pool, authService, f.users, f.nodes, f.comments, f.store).Handler()
	path := fmt.Sprintf("/api/nodes/%d/comment-images", root.ID)
	var imageID int64
	for _, test := range []struct {
		name, field string
		files, size int
		csrf, login bool
		want        int
	}{
		{"anonymous", "file", 1, 0, true, false, 401},
		{"missing CSRF", "file", 1, 0, false, true, 403},
		{"missing file", "file", 0, 0, true, true, 400},
		{"multiple files", "file", 2, 0, true, true, 400},
		{"old batch field", "files", 1, 0, true, true, 400},
		{"oversized", "file", 1, storage.MaxAttachmentBytes + 1, true, true, 413},
		{"single image", "file", 1, 0, true, true, 201},
	} {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			form := multipart.NewWriter(&body)
			for range test.files {
				part, err := form.CreateFormFile(test.field, "image.png")
				if err != nil {
					t.Fatal(err)
				}
				data := testPNG(t)
				if test.size > 0 {
					data = make([]byte, test.size)
				}
				if _, err := part.Write(data); err != nil {
					t.Fatal(err)
				}
			}
			if err := form.Close(); err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, path, &body)
			request.Header.Set("Content-Type", form.FormDataContentType())
			if test.login {
				request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: id})
			}
			if test.csrf {
				request.Header.Set("X-CSRF-Token", csrf)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status %d, want %d: %s", response.Code, test.want, response.Body)
			}
			if test.want == 201 {
				var upload struct{ ID int64 }
				if err := json.Unmarshal(response.Body.Bytes(), &upload); err != nil || upload.ID == 0 {
					t.Fatalf("missing uploaded image id: %s", response.Body)
				}
				imageID = upload.ID
			}
		})
	}
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/nodes/%d/comments", root.ID),
		strings.NewReader(fmt.Sprintf(`{"body":"One image","attachmentIds":[%d]}`, imageID)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", csrf)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: id})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var comment comments.Comment
	if err := json.Unmarshal(response.Body.Bytes(), &comment); err != nil || response.Code != 201 || len(comment.Attachments) != 1 || comment.Attachments[0].ID != imageID {
		t.Fatalf("publishing uploaded image failed: %d %s", response.Code, response.Body)
	}
}

func (s *failingUploadStore) Put(ctx context.Context, key string, data []byte, contentType string) error {
	if s.failNext {
		s.failNext = false
		return errors.New("object storage unavailable")
	}
	return s.MemoryStore.Put(ctx, key, data, contentType)
}

func TestIndependentCommentUploads(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	store := &failingUploadStore{MemoryStore: f.store}
	f.comments = comments.NewService(comments.NewRepository(f.pool), f.users, access.New(f.pool), store)
	root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Images", LeaderSubjects: []string{"alice", "bob"}})
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Other", LeaderSubjects: []string{"alice"}})
	if err != nil {
		t.Fatal(err)
	}
	data := testPNG(t)
	if _, err := f.comments.UploadImage(ctx, f.carol, root.ID, data); apperr.From(err).Status != 403 {
		t.Fatalf("nonmember upload: %v", err)
	}
	if _, err := f.comments.UploadImage(ctx, f.alice, root.ID, []byte("not an image")); apperr.From(err).Status != 400 {
		t.Fatalf("invalid image: %v", err)
	}
	first, err := f.comments.UploadImage(ctx, f.alice, root.ID, data)
	if err != nil {
		t.Fatal(err)
	}
	store.failNext = true
	if _, err := f.comments.UploadImage(ctx, f.alice, root.ID, data); err == nil {
		t.Fatal("expected second file to fail")
	}
	if len(f.store.Keys()) != 1 {
		t.Fatal("failed second upload must preserve the first image")
	}
	if _, _, _, err := f.comments.GetAttachment(ctx, first); apperr.From(err).Status != 404 {
		t.Fatal("unpublished image must not be readable")
	}
	second, err := f.comments.UploadImage(ctx, f.alice, root.ID, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.store.Keys()) != 2 {
		t.Fatal("retry should add only the failed image")
	}
	if _, err := f.comments.Create(ctx, f.bob, root.ID, "stolen", []int64{first}); apperr.From(err).Status != 400 {
		t.Fatalf("another user's upload was accepted: %v", err)
	}
	if _, err := f.comments.Create(ctx, f.alice, other.ID, "wrong node", []int64{first}); apperr.From(err).Status != 400 {
		t.Fatalf("another node's upload was accepted: %v", err)
	}
	for _, ids := range [][]int64{{first, first}, {first, 99999}, {first, second, first, second, first}} {
		if _, err := f.comments.Create(ctx, f.alice, root.ID, "invalid", ids); apperr.From(err).Status != 400 {
			t.Fatalf("invalid attachments %v: %v", ids, err)
		}
	}
	list, err := f.comments.List(ctx, f.alice, root.ID, nil, 30)
	if err != nil || len(list.Items) != 0 {
		t.Fatalf("uploads and failed publication must not create comments: %+v, %v", list, err)
	}
	comment, err := f.comments.Create(ctx, f.alice, root.ID, "", []int64{second, first})
	if err != nil {
		t.Fatal(err)
	}
	if len(comment.Attachments) != 2 || comment.Attachments[0].ID != second || comment.Attachments[1].ID != first {
		t.Fatalf("publication must retain selection order: %+v", comment.Attachments)
	}
	if len(f.store.Keys()) != 2 {
		t.Fatal("publication must reuse the existing objects")
	}
	if _, err := f.comments.Create(ctx, f.alice, root.ID, "reuse", []int64{first}); apperr.From(err).Status != 400 {
		t.Fatalf("published image was reused: %v", err)
	}
	if err := f.comments.DeleteUpload(ctx, f.alice, first); apperr.From(err).Status != 404 {
		t.Fatalf("discard must not delete a published image: %v", err)
	}

	discarded, err := f.comments.UploadImage(ctx, f.alice, root.ID, data)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.comments.DeleteUpload(ctx, f.bob, discarded); apperr.From(err).Status != 404 {
		t.Fatalf("another user discarded an image: %v", err)
	}
	if err := f.comments.DeleteUpload(ctx, f.alice, discarded); err != nil {
		t.Fatal(err)
	}
	if len(f.store.Keys()) != 2 {
		t.Fatal("discarded image remains in storage")
	}
	expired, err := f.comments.UploadImage(ctx, f.alice, root.ID, data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE comment_attachments SET created_at=now()-interval '25 hours'`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.comments.Create(ctx, f.alice, root.ID, "expired", []int64{expired}); apperr.From(err).Status != 400 {
		t.Fatalf("expired image was accepted: %v", err)
	}
	cleanupCtx, stopCleanup := context.WithCancel(ctx)
	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		f.comments.StartUploadCleanup(cleanupCtx)
	}()
	deadline := time.Now().Add(5 * time.Second)
	for len(f.store.Keys()) != 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	stopCleanup()
	<-cleanupDone
	if len(f.store.Keys()) != 2 {
		t.Fatal("expiry must delete only unpublished images")
	}
	if _, err := f.comments.UploadImage(ctx, f.alice, root.ID, data); err != nil {
		t.Fatal(err)
	}
	if err := f.nodes.Delete(ctx, f.admin, root.ID); err != nil {
		t.Fatal(err)
	}
	if len(f.store.Keys()) != 0 {
		t.Fatal("node deletion must remove both published and pending images")
	}
}
