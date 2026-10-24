package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.bitnp.net/BITNP/guild-go/internal/access"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
	"git.bitnp.net/BITNP/guild-go/internal/comments"
	"git.bitnp.net/BITNP/guild-go/internal/config"
	"git.bitnp.net/BITNP/guild-go/internal/database"
	"git.bitnp.net/BITNP/guild-go/internal/httpapi"
	"git.bitnp.net/BITNP/guild-go/internal/nodes"
	"git.bitnp.net/BITNP/guild-go/internal/storage"
	"git.bitnp.net/BITNP/guild-go/internal/users"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		slog.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	store, err := storage.NewS3(ctx, cfg.S3Endpoint, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey)
	if err != nil {
		slog.Error("S3 configuration failed", "error", err)
		os.Exit(1)
	}

	userRepo := users.NewRepository(pool)
	keycloakAdmin := users.NewKeycloakAdmin(
		cfg.KeycloakAdminURL,
		cfg.KeycloakAdminClientID,
		cfg.KeycloakAdminClientSecret,
		cfg.OIDCIssuer,
	)
	if !keycloakAdmin.Enabled() {
		slog.Warn("Keycloak Admin API is disabled; other users will show subject-only profiles")
	}
	directory := users.NewDirectory(keycloakAdmin)
	userService := users.NewService(userRepo, directory)

	sessions := auth.NewStore(cfg.SessionTimeout)
	pending := auth.NewPendingStore()
	oidcClient := auth.NewOIDC(cfg.OIDCIssuer, cfg.OIDCClientID, cfg.OIDCClientSecret, cfg.RedirectURL())
	authService := auth.NewService(oidcClient, sessions, pending, cfg.PublicURL, cfg.SecureCookie, userService.Provision)
	if cfg.DevAuthBypass {
		if err := userService.Provision(ctx, cfg.DevAuthSubject); err != nil {
			slog.Error("provision development user failed", "error", err)
			os.Exit(1)
		}
		if err := authService.EnableDevAuthBypass(cfg.DevAuthSubject, cfg.DevAuthName); err != nil {
			slog.Error("enable development authentication bypass failed", "error", err)
			os.Exit(1)
		}
		slog.Warn("development authentication bypass enabled; do not use in production", "subject", cfg.DevAuthSubject)
	}

	checker := access.New(pool)
	nodeRepo := nodes.NewRepository(pool)
	nodeService := nodes.NewService(nodeRepo, userService, checker, store)
	commentRepo := comments.NewRepository(pool)
	commentService := comments.NewService(commentRepo, userService, checker, store)

	server := httpapi.New(cfg, pool, authService, userService, nodeService, commentService, store)

	go sessions.StartCleanup(ctx)
	go pending.StartCleanup(ctx)
	go commentService.StartUploadCleanup(ctx)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       120 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		slog.Info("guild-go listening", "addr", cfg.Addr, "public_url", cfg.PublicURL.String())
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Warn("graceful shutdown failed", "error", err)
	}
	slog.Info("guild-go stopped")
}
