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
	"git.bitnp.net/BITNP/guild-go/internal/config"
	"git.bitnp.net/BITNP/guild-go/internal/database"
	"git.bitnp.net/BITNP/guild-go/internal/events"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
	"git.bitnp.net/BITNP/guild-go/internal/storage"
	"git.bitnp.net/BITNP/guild-go/internal/users"
	"git.bitnp.net/BITNP/guild-go/internal/web"
)

// InternalAuthorizer resolves an acting user session and provisions known
// subjects on behalf of an authenticated internal service.
type InternalAuthorizer interface {
	ResolveActorSession(ctx context.Context, subject string) (auth.Session, error)
	EnsureSubjects(ctx context.Context, subjects []string) error
}

// InternalFeed exposes committed domain events and service-level read projections.
type InternalFeed interface {
	LatestID(ctx context.Context) (int64, error)
	ListAfter(ctx context.Context, after int64, limit int) (events.ListResponse, error)
	ListDueNodes(ctx context.Context, query events.DueNodesQuery) (events.DueNodesResponse, error)
	ActivitySummary(ctx context.Context, start, end time.Time) (events.ActivitySummary, error)
}

// Server owns the HTTP dependencies.
type Server struct {
	cfg           config.Config
	pool          *pgxpool.Pool
	auth          *auth.Service
	users         *users.Service
	nodes         *nodes.Service
	comments      *comments.Service
	store         storage.Store
	internalToken string
	internalAuthz InternalAuthorizer
	internalFeed  InternalFeed
	eventBroker   *events.Broker
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

// WithInternalAPI enables actor-delegated /api/internal/* service endpoints using
// a shared bearer token and Keycloak-backed actor resolution.
func (s *Server) WithInternalAPI(token string, authorizer InternalAuthorizer) *Server {
	s.internalToken = strings.TrimSpace(token)
	s.internalAuthz = authorizer

	return s
}

// WithInternalFeed enables service-level /api/internal/* event, deadline and
// activity summary endpoints authenticated solely by the shared bearer token.
func (s *Server) WithInternalFeed(token string, feed InternalFeed, broker *events.Broker) *Server {
	if trimmed := strings.TrimSpace(token); trimmed != "" {
		s.internalToken = trimmed
	}
	s.internalFeed = feed
	s.eventBroker = broker

	return s
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
	mux.Handle("PUT /api/nodes/{id}/move", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.moveNode))))
	mux.Handle("DELETE /api/nodes/{id}", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.deleteNode))))
	mux.Handle("GET /api/nodes/{id}/members", s.auth.RequireSession(http.HandlerFunc(s.getMembers)))
	mux.Handle("PUT /api/nodes/{id}/members", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.replaceMembers))))
	mux.Handle("GET /api/nodes/{id}/comments", s.auth.RequireSession(http.HandlerFunc(s.listComments)))
	mux.Handle("GET /api/nodes/{id}/progress", s.auth.RequireSession(http.HandlerFunc(s.listProgress)))
	mux.Handle("POST /api/nodes/{id}/comments", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.createComment))))
	mux.Handle("POST /api/nodes/{id}/comment-images", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.uploadCommentImage))))
	mux.Handle("DELETE /api/comment-images/{id}", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.deleteCommentUpload))))
	mux.Handle("DELETE /api/comments/{id}", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.deleteComment))))
	mux.Handle("GET /api/comments/{id}/reactions/{emoji}", s.auth.RequireSession(http.HandlerFunc(s.reactionUsers)))
	mux.Handle("PUT /api/comments/{id}/reactions/{emoji}", s.auth.RequireSession(s.auth.RequireCSRF(http.HandlerFunc(s.toggleReaction))))
	mux.Handle("GET /api/attachments/{id}", s.auth.RequireSession(http.HandlerFunc(s.getAttachment)))

	mux.Handle("GET /api/internal/users", s.internalAuth(http.HandlerFunc(s.listUsers)))
	mux.Handle("GET /api/internal/overview", s.internalAuth(http.HandlerFunc(s.overview)))
	mux.Handle("GET /api/internal/nodes", s.internalAuth(http.HandlerFunc(s.listNodes)))
	mux.Handle("GET /api/internal/nodes/{id}", s.internalAuth(http.HandlerFunc(s.getNode)))
	mux.Handle("POST /api/internal/nodes", s.internalAuth(http.HandlerFunc(s.createNodeInternal)))
	mux.Handle("PUT /api/internal/nodes/{id}", s.internalAuth(http.HandlerFunc(s.updateNode)))
	mux.Handle("PUT /api/internal/nodes/{id}/move", s.internalAuth(http.HandlerFunc(s.moveNode)))
	mux.Handle("DELETE /api/internal/nodes/{id}", s.internalAuth(http.HandlerFunc(s.deleteNode)))
	mux.Handle("PUT /api/internal/nodes/{id}/members", s.internalAuth(http.HandlerFunc(s.replaceMembersInternal)))
	mux.Handle("GET /api/internal/nodes/{id}/progress", s.internalAuth(http.HandlerFunc(s.listProgress)))
	mux.Handle("POST /api/internal/nodes/{id}/comments", s.internalAuth(http.HandlerFunc(s.createComment)))

	mux.Handle("GET /api/internal/events", s.internalServiceAuth(http.HandlerFunc(s.listEventsInternal)))
	mux.Handle("GET /api/internal/events/stream", s.internalServiceAuth(http.HandlerFunc(s.streamEventsInternal)))
	mux.Handle("GET /api/internal/due-nodes", s.internalServiceAuth(http.HandlerFunc(s.dueNodesInternal)))
	mux.Handle("GET /api/internal/activity-summary", s.internalServiceAuth(http.HandlerFunc(s.activitySummaryInternal)))

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
	return s.auth.SessionForRequest(r)
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

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
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
