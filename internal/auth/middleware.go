package auth

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

type SessionHandler func(http.ResponseWriter, *http.Request, Session)

// RequireSession passes the authenticated session to the handler.
func (s *Service) RequireSession(next SessionHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := s.SessionForRequest(r)
		if !ok {
			if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "登录已过期")
				return
			}
			writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "需要登录")
			return
		}
		next(w, r, session)
	})
}

// RequireCSRF validates the session CSRF token on unsafe methods.
func RequireCSRF(next SessionHandler) SessionHandler {
	return func(w http.ResponseWriter, r *http.Request, session Session) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next(w, r, session)
			return
		}
		token := r.Header.Get("X-CSRF-Token")
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(session.CSRF)) != 1 {
			writeAuthError(w, http.StatusForbidden, "CSRF_INVALID", "CSRF 令牌无效")
			return
		}
		next(w, r, session)
	}
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
