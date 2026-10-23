// Package httpapi wires the HTTP surface, middleware and static SPA.
package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/config"
	"git.bitnp.net/BITNP/guild-go/internal/database"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
	"git.bitnp.net/BITNP/guild-go/internal/storage"
	"git.bitnp.net/BITNP/guild-go/internal/users"
	"git.bitnp.net/BITNP/guild-go/internal/web"
)

// Server owns the HTTP dependencies.
type Server struct {
	cfg      config.Config
	pool     *pgxpool.Pool
	auth     *auth.Service
	users    *users.Service
	nodes    *nodes.Service
	comments *comments.Service
	store    storage.Store
}

// New creates an HTTP server.
func New(cfg config.Config, pool *pgxpool.Pool, authService *auth.Service, userService *users.Service, nodeService *nodes.Service, commentService *comments.Service, store storage.Store) *Server {
	return &Server{
		cfg:      cfg,
		pool:     pool,
		auth:     authService,
		users:    userService,
		nodes:    nodeService,
		comments: commentService,
		store:    store,
	}
}

// Handler returns the complete HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", s.health)
	mux.HandleFunc("GET /api/auth/login", s.auth.StartLogin)
	mux.HandleFunc("GET /api/auth/callback", s.auth.Callback)
	mux.HandleFunc("GET /api/auth/status", s.authStatus)
	mux.Handle("GET /api/auth/csrf", s.auth.RequireSession(http.HandlerFunc(s.csrf)))
	mux.Handle("GET /api/auth/me", s.auth.RequireSession(http.HandlerFunc(s.me)))
	mux.Handle("POST /api/auth/logout", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.logout))))

	mux.Handle("GET /api/users", s.auth.RequireSession(http.HandlerFunc(s.listUsers)))
	mux.Handle("GET /api/users/{subject}", s.auth.RequireSession(http.HandlerFunc(s.getUser)))

	mux.Handle("GET /api/overview", s.auth.RequireSession(http.HandlerFunc(s.overview)))
	mux.Handle("GET /api/nodes", s.auth.RequireSession(http.HandlerFunc(s.listNodes)))
	mux.Handle("GET /api/nodes/{id}", s.auth.RequireSession(http.HandlerFunc(s.getNode)))
	mux.Handle("POST /api/nodes", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.createNode))))
	mux.Handle("PUT /api/nodes/order", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.orderNodes))))
	mux.Handle("PUT /api/nodes/{id}", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.updateNode))))
	mux.Handle("DELETE /api/nodes/{id}", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.deleteNode))))
	mux.Handle("GET /api/nodes/{id}/members", s.auth.RequireSession(http.HandlerFunc(s.getMembers)))
	mux.Handle("PUT /api/nodes/{id}/members", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.replaceMembers))))
	mux.Handle("GET /api/nodes/{id}/comments", s.auth.RequireSession(http.HandlerFunc(s.listComments)))
	mux.Handle("POST /api/nodes/{id}/comments", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.createComment))))
	mux.Handle("POST /api/nodes/{id}/comment-images", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.uploadCommentImage))))
	mux.Handle("DELETE /api/comment-images/{id}", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.deleteCommentUpload))))
	mux.Handle("DELETE /api/comments/{id}", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.deleteComment))))
	mux.Handle("GET /api/comments/{id}/reactions/{emoji}", s.auth.RequireSession(http.HandlerFunc(s.reactionUsers)))
	mux.Handle("PUT /api/comments/{id}/reactions/{emoji}", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.toggleReaction))))
	mux.Handle("GET /api/attachments/{id}", s.auth.RequireSession(http.HandlerFunc(s.getAttachment)))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, errorResponse{Status: 404, Error: "NOT_FOUND", Message: "接口不存在"})
	})
	mux.Handle("/", web.Handler())

	return s.recoverer(s.logger(s.securityHeaders(mux)))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 3*time.Second)
	defer cancel()
	if err := database.Ping(ctx, s.pool); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	session, ok := s.currentSession(r)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"state": "anonymous", "user": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": "active", "user": s.users.Current(session)})
}

func (s *Server) csrf(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"token": session.CSRF, "headerName": "X-CSRF-Token"})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	writeJSON(w, http.StatusOK, s.users.Current(session))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Status: 401, Error: "UNAUTHORIZED", Message: "需要登录"})
		return
	}
	s.auth.Logout(w, r, session.ID)
}

func (s *Server) currentSession(r *http.Request) (auth.Session, bool) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil || cookie.Value == "" {
		return auth.Session{}, false
	}
	return s.auth.Sessions.Get(cookie.Value)
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' https: data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; font-src 'self' data:; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (s *Server) logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		if !strings.HasPrefix(r.URL.Path, "/assets/") {
			slog.Info("http request", "method", r.Method, "path", r.URL.Path, "status", recorder.status, "duration", time.Since(start).Round(time.Millisecond))
		}
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("panic recovered", "method", r.Method, "path", r.URL.Path, "panic", recovered)
				writeJSON(w, http.StatusInternalServerError, errorResponse{Status: 500, Error: "INTERNAL", Message: "服务器内部错误"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
