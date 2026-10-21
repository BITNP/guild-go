package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
)

func contextWithTimeout(r *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), timeout)
}

func parseID(r *http.Request, name string) (int64, error) {
	value := r.PathValue(name)
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.NotFound("资源不存在")
	}
	return id, nil
}

func parseOptionalInt64(r *http.Request, name string) (*int64, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return nil, apperr.BadRequest(name + " 参数不正确")
	}
	return &parsed, nil
}

func parseLimit(r *http.Request) (int, error) {
	value := r.URL.Query().Get("limit")
	if value == "" {
		return 30, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, apperr.BadRequest("limit 参数不正确")
	}
	if parsed > 100 {
		parsed = 100
	}
	return parsed, nil
}

func currentSession(r *http.Request) (auth.Session, bool) {
	return auth.SessionFrom(r.Context())
}
