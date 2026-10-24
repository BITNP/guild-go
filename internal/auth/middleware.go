package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

type contextKey int

const sessionContextKey contextKey = 1

// RequireSession attaches the current session to the request context.
func (s *Service) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := s.SessionForRequest(r)
		if !ok {
			// A present cookie means the session was dropped or expired; keep
			// telling the browser that, as before the development bypass existed.
			if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "登录已过期")
				return
			}
			writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "需要登录")
			return
		}
		ctx := context.WithValue(r.Context(), sessionContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireCSRF validates the session CSRF token on unsafe methods.
func (s *Service) RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		session, ok := SessionFrom(r.Context())
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "需要登录")
			return
		}
		token := r.Header.Get("X-CSRF-Token")
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(session.CSRF)) != 1 {
			writeAuthError(w, http.StatusForbidden, "CSRF_INVALID", "CSRF 令牌无效")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireSystemAdmin rejects callers without the system-admin permission.
func RequireSystemAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := SessionFrom(r.Context())
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "需要登录")
			return
		}
		if !session.Permissions.SystemAdmin {
			writeAuthError(w, http.StatusForbidden, "FORBIDDEN", "需要系统管理员权限")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SessionFrom returns the session stored by RequireSession.
func SessionFrom(ctx context.Context) (Session, bool) {
	session, ok := ctx.Value(sessionContextKey).(Session)
	return session, ok
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  status,
		"error":   code,
		"message": message,
	})
}
