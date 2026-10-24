package config

import (
	"strings"
	"testing"
)

func TestDevAuthBypassDoesNotRequireOIDC(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_DEV_AUTH_BYPASS", "true")
	t.Setenv("OIDC_ISSUER", "")
	t.Setenv("OIDC_CLIENT_ID", "")
	t.Setenv("OIDC_CLIENT_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load development config: %v", err)
	}
	if !cfg.DevAuthBypass {
		t.Fatal("development authentication bypass must be enabled")
	}
	if cfg.DevAuthSubject != "dev-user" {
		t.Fatalf("default development subject = %q", cfg.DevAuthSubject)
	}
}

func TestOIDCMustBeConfiguredWithoutDevAuthBypass(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_DEV_AUTH_BYPASS", "false")
	t.Setenv("OIDC_ISSUER", "")
	t.Setenv("OIDC_CLIENT_ID", "")
	t.Setenv("OIDC_CLIENT_SECRET", "")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "OIDC_ISSUER") {
		t.Fatalf("expected a missing OIDC_ISSUER error, got %v", err)
	}
}

func TestDevAuthBypassRejectsEmptyIdentity(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_DEV_AUTH_BYPASS", "true")
	t.Setenv("APP_DEV_AUTH_SUBJECT", "")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "APP_DEV_AUTH_SUBJECT") {
		t.Fatalf("expected an empty development identity error, got %v", err)
	}
}

func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_PUBLIC_URL", "http://localhost:8080")
	t.Setenv("APP_SECURE_COOKIE", "false")
	t.Setenv("DATABASE_URL", "postgres://guild:password@localhost/guild")
	t.Setenv("S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("S3_BUCKET", "guild")
	t.Setenv("S3_ACCESS_KEY", "access-key")
	t.Setenv("S3_SECRET_KEY", "secret-key")
	t.Setenv("KEYCLOAK_ADMIN_URL", "")
	t.Setenv("KEYCLOAK_ADMIN_CLIENT_ID", "")
	t.Setenv("KEYCLOAK_ADMIN_CLIENT_SECRET", "")
}
