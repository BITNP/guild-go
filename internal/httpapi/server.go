// Package httpapi wires the HTTP surface, middleware and static SPA.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"git.bitnp.net/BITNP/guild-go/internal/apperr"
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

type sessionHandler func(http.ResponseWriter, *http.Request, auth.Session)

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

func (s *Server) handleSession(mux *http.ServeMux, pattern string, requireCSRF bool, fn sessionHandler) {
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := auth.SessionFrom(r.Context())
		if !ok {
			writeError(w, r, apperr.Unauthorized("需要登录"))
			return
		}
		fn(w, r, session)
	})
	if requireCSRF {
		handler = s.auth.RequireCSRF(handler)
	}
	mux.Handle(pattern, s.auth.RequireSession(handler))
}

// Handler returns the complete HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", s.health)
	mux.HandleFunc("GET /api/auth/login", s.auth.StartLogin)
	mux.HandleFunc("GET /api/auth/callback", s.auth.Callback)
	mux.HandleFunc("GET /api/auth/status", s.authStatus)

	s.handleSession(mux, "GET /api/auth/csrf", false, s.csrf)
	s.handleSession(mux, "GET /api/auth/me", false, s.me)
	s.handleSession(mux, "POST /api/auth/logout", true, s.logout)

	s.handleSession(mux, "GET /api/users", false, s.listUsers)
	s.handleSession(mux, "GET /api/users/{subject}", false, s.getUser)

	s.handleSession(mux, "GET /api/overview", false, s.overview)
	s.handleSession(mux, "GET /api/nodes", false, s.listNodes)
	s.handleSession(mux, "GET /api/nodes/{id}", false, s.getNode)
	s.handleSession(mux, "POST /api/nodes", true, s.createNode)
	s.handleSession(mux, "PUT /api/nodes/order", true, s.orderNodes)
	s.handleSession(mux, "PUT /api/nodes/{id}", true, s.updateNode)
	s.handleSession(mux, "PUT /api/nodes/{id}/move", true, s.moveNode)
	s.handleSession(mux, "DELETE /api/nodes/{id}", true, s.deleteNode)
	s.handleSession(mux, "GET /api/nodes/{id}/members", false, s.getMembers)
	s.handleSession(mux, "PUT /api/nodes/{id}/members", true, s.replaceMembers)
	s.handleSession(mux, "GET /api/nodes/{id}/comments", false, s.listComments)
	s.handleSession(mux, "GET /api/nodes/{id}/progress", false, s.listProgress)
	s.handleSession(mux, "POST /api/nodes/{id}/comments", true, s.createComment)
	s.handleSession(mux, "POST /api/nodes/{id}/comment-images", true, s.uploadCommentImage)
	s.handleSession(mux, "DELETE /api/comment-images/{id}", true, s.deleteCommentUpload)
	s.handleSession(mux, "DELETE /api/comments/{id}", true, s.deleteComment)
	s.handleSession(mux, "GET /api/comments/{id}/reactions/{emoji}", false, s.reactionUsers)
	s.handleSession(mux, "PUT /api/comments/{id}/reactions/{emoji}", true, s.toggleReaction)
	s.handleSession(mux, "GET /api/attachments/{id}", false, s.getAttachment)

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

func (s *Server) csrf(w http.ResponseWriter, _ *http.Request, session auth.Session) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"token": session.CSRF, "headerName": "X-CSRF-Token"})
}

func (s *Server) me(w http.ResponseWriter, _ *http.Request, session auth.Session) {
	writeJSON(w, http.StatusOK, s.users.Current(session))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request, session auth.Session) {
	s.auth.Logout(w, r, session.ID)
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
