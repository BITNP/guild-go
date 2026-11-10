package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func waitForBlockedOperation(t *testing.T, ctx context.Context, pool *pgxpool.Pool, blockingPID uint32, result <-chan error) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		var waiting bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity WHERE datname=current_database()
			AND $1::int = ANY(pg_blocking_pids(pid))
		)`, int64(blockingPID)).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}

		select {
		case err := <-result:
			t.Fatalf("operation completed before the blocking transaction: %v", err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
}
