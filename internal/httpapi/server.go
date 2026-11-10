// Package httpapi wires the HTTP surface, middleware and static SPA.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/database"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
	"git.bitnp.net/BITNP/guild-go/internal/users"
	"git.bitnp.net/BITNP/guild-go/internal/web"
)

// Server owns the HTTP dependencies.
type Server struct {
	pool     *pgxpool.Pool
	auth     *auth.Service
	users    *users.Service
	nodes    *nodes.Service
	comments *comments.Service
}

// New creates an HTTP server.
func New(pool *pgxpool.Pool, authService *auth.Service, userService *users.Service, nodeService *nodes.Service, commentService *comments.Service) *Server {
	return &Server{
		pool:     pool,
		auth:     authService,
		users:    userService,
		nodes:    nodeService,
		comments: commentService,
	}
}

// Handler returns the complete HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", s.health)
	mux.HandleFunc("GET /api/auth/login", s.auth.StartLogin)
	mux.HandleFunc("GET /api/auth/callback", s.auth.Callback)
	mux.HandleFunc("GET /api/auth/status", s.authStatus)
	mux.Handle("GET /api/auth/csrf", s.protected(s.csrf))
	mux.Handle("GET /api/auth/me", s.protected(s.me))
	mux.Handle("POST /api/auth/logout", s.protected(s.logout))

	mux.Handle("GET /api/users", s.protected(s.listUsers))
	mux.Handle("GET /api/users/{subject}", s.protected(s.getUser))

	mux.Handle("GET /api/overview", s.protected(s.overview))
	mux.Handle("GET /api/nodes", s.protected(s.listNodes))
	mux.Handle("GET /api/nodes/{id}", s.protected(s.getNode))
	mux.Handle("POST /api/nodes", s.protected(s.createNode))
	mux.Handle("PUT /api/nodes/order", s.protected(s.orderNodes))
	mux.Handle("PUT /api/nodes/{id}", s.protected(s.updateNode))
	mux.Handle("PUT /api/nodes/{id}/move", s.protected(s.moveNode))
	mux.Handle("DELETE /api/nodes/{id}", s.protected(s.deleteNode))
	mux.Handle("GET /api/nodes/{id}/members", s.protected(s.getMembers))
	mux.Handle("PUT /api/nodes/{id}/members", s.protected(s.replaceMembers))
	mux.Handle("GET /api/nodes/{id}/comments", s.protected(s.listComments))
	mux.Handle("GET /api/nodes/{id}/progress", s.protected(s.listProgress))
	mux.Handle("POST /api/nodes/{id}/comments", s.protected(s.createComment))
	mux.Handle("POST /api/nodes/{id}/comment-images", s.protected(s.uploadCommentImage))
	mux.Handle("DELETE /api/comment-images/{id}", s.protected(s.deleteCommentUpload))
	mux.Handle("DELETE /api/comments/{id}", s.protected(s.deleteComment))
	mux.Handle("GET /api/comments/{id}/reactions/{emoji}", s.protected(s.reactionUsers))
	mux.Handle("PUT /api/comments/{id}/reactions/{emoji}", s.protected(s.toggleReaction))
	mux.Handle("GET /api/attachments/{id}", s.protected(s.getAttachment))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, errorResponse{Status: 404, Error: "NOT_FOUND", Message: "接口不存在"})
	})
	mux.Handle("/", web.Handler())

	return s.recoverer(s.logger(s.securityHeaders(mux)))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
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
	session, ok := s.auth.SessionForRequest(r)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"state": "anonymous", "user": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": "active", "user": s.users.Current(session)})
}

func (s *Server) csrf(w http.ResponseWriter, r *http.Request, session auth.Session) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"token": session.CSRF, "headerName": "X-CSRF-Token"})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, session auth.Session) {
	writeJSON(w, http.StatusOK, s.users.Current(session))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request, session auth.Session) {
	s.auth.Logout(w, r, session.ID)
}

func (s *Server) protected(handler auth.SessionHandler) http.Handler {
	return s.auth.RequireSession(auth.RequireCSRF(handler))
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
