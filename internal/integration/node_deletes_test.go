package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
	"git.bitnp.net/BITNP/guild-go/internal/storage"
)

func TestNodeDeleteIncludesConcurrentCommentImages(t *testing.T) {
	f := setup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Root"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{ParentID: &root.ID, Title: "Child"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Other"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.comments.UploadImage(ctx, f.admin, other.ID, testPNG(t)); err != nil {
		t.Fatal(err)
	}

	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	var commentID int64
	if err := tx.QueryRow(ctx, `INSERT INTO comments (node_id, author_subject, body)
		VALUES ($1,'admin','Concurrent comment') RETURNING id`, child.ID).Scan(&commentID); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- f.nodes.Delete(ctx, f.admin, root.ID) }()
	waitForBlockedOperation(t, ctx, f.pool, tx.Conn().PgConn().PID(), result)

	const key = "concurrent-comment-image"
	data := testPNG(t)
	if err := f.store.Put(ctx, key, data, "image/png"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO comment_attachments (comment_id, object_key, content_type, size_bytes)
		VALUES ($1,$2,'image/png',$3)`, commentID, key, len(data)); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := f.store.Get(ctx, key); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("concurrently committed image survived subtree deletion: %v", err)
	}
	if len(f.store.Keys()) != 1 {
		t.Fatalf("another node's pending image must survive: %v", f.store.Keys())
	}
	if _, err := f.nodes.Get(ctx, f.admin, child.ID); apperr.From(err).Status != 404 {
		t.Fatalf("child survived subtree deletion: %v", err)
	}
	if _, err := f.nodes.Get(ctx, f.admin, other.ID); err != nil {
		t.Fatalf("unrelated node was removed: %v", err)
	}
}

func TestQueuedDeleteRechecksInheritedLeadership(t *testing.T) {
	f := setup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Root", LeaderSubjects: []string{"alice"}})
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Other", LeaderSubjects: []string{"carol"}})
	if err != nil {
		t.Fatal(err)
	}
	child, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{ParentID: &root.ID, Title: "Child", LeaderSubjects: []string{"dave"}})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `LOCK TABLE nodes IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- f.nodes.Delete(ctx, f.alice, child.ID) }()
	waitForBlockedOperation(t, ctx, f.pool, tx.Conn().PgConn().PID(), result)

	if _, err := tx.Exec(ctx, `UPDATE nodes SET parent_id=$2 WHERE id=$1`, child.ID, other.ID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-result; apperr.From(err).Status != 403 {
		t.Fatalf("queued deletion must lose permission after ancestry changes: %v", err)
	}
	after, err := f.nodes.Get(ctx, f.admin, child.ID)
	if err != nil || after.ParentID == nil || *after.ParentID != other.ID {
		t.Fatalf("rejected deletion changed the node: %+v, %v", after, err)
	}
}

func TestTreeWritesWaitForNodeEdits(t *testing.T) {
	for _, operation := range []string{"move", "delete"} {
		t.Run(operation, func(t *testing.T) {
			f := setup(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			root, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{Title: "Root"})
			if err != nil {
				t.Fatal(err)
			}
			child, err := f.nodes.Create(ctx, f.admin, nodes.CreateRequest{ParentID: &root.ID, Title: "Child"})
			if err != nil {
				t.Fatal(err)
			}
			tx, err := f.pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(context.Background())
			var title string
			if err := tx.QueryRow(ctx, `SELECT title FROM nodes WHERE id=$1 FOR UPDATE`, child.ID).Scan(&title); err != nil {
				t.Fatal(err)
			}
			result := make(chan error, 1)
			go func() {
				if operation == "delete" {
					result <- f.nodes.Delete(ctx, f.admin, child.ID)
					return
				}
				_, err := f.nodes.Move(ctx, f.admin, child.ID, nodes.MoveRequest{})
				result <- err
			}()
			waitForBlockedOperation(t, ctx, f.pool, tx.Conn().PgConn().PID(), result)

			if _, err := tx.Exec(ctx, `UPDATE nodes SET title='Edited' WHERE id=$1`, child.ID); err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			if err := <-result; err != nil {
				t.Fatalf("tree write conflicted with a preceding edit: %v", err)
			}
			after, err := f.nodes.Get(ctx, f.admin, child.ID)
			if operation == "delete" {
				if apperr.From(err).Status != 404 {
					t.Fatalf("node survived deletion: %+v, %v", after, err)
				}
			} else if err != nil || after.ParentID != nil || after.Title != "Edited" {
				t.Fatalf("move lost the preceding edit: %+v, %v", after, err)
			}
		})
	}
}
