package storage

import (
	"context"
	"log/slog"
	"time"
)

func DeleteObjects(store Store, keys []string) {
	for _, key := range keys {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := store.Delete(ctx, key); err != nil {
			slog.Warn("delete stored object failed", "key", key, "error", err)
		}
		cancel()
	}
}
