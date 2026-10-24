package config

import "testing"

func TestDevAuthBypassDoesNotRequireOIDC(t *testing.T) {
	t.Setenv("APP_DEV_AUTH_BYPASS", "true")
	t.Setenv("DATABASE_URL", "postgres://guild:password@localhost/guild")
	t.Setenv("S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("S3_BUCKET", "guild")
	t.Setenv("S3_ACCESS_KEY", "access-key")
	t.Setenv("S3_SECRET_KEY", "secret-key")
	t.Setenv("OIDC_ISSUER", "")
	t.Setenv("OIDC_CLIENT_ID", "")
	t.Setenv("OIDC_CLIENT_SECRET", "")
	t.Setenv("KEYCLOAK_ADMIN_URL", "")
	t.Setenv("KEYCLOAK_ADMIN_CLIENT_ID", "")
	t.Setenv("KEYCLOAK_ADMIN_CLIENT_SECRET", "")

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
