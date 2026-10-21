// Package config loads and validates environment configuration.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

// Config is the complete runtime configuration.
type Config struct {
	Addr           string
	PublicURL      *url.URL
	SecureCookie   bool
	SessionTimeout time.Duration

	DatabaseURL string

	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string

	KeycloakAdminURL          string
	KeycloakAdminClientID     string
	KeycloakAdminClientSecret string

	S3Endpoint  string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
}

// Load reads configuration from the environment.
func Load() (Config, error) {
	cfg := Config{
		Addr:                      env("APP_ADDR", ":8080"),
		SecureCookie:              envBool("APP_SECURE_COOKIE", false),
		SessionTimeout:            envDuration("APP_SESSION_TIMEOUT", 30*time.Minute),
		DatabaseURL:               env("DATABASE_URL", ""),
		OIDCIssuer:                strings.TrimRight(env("OIDC_ISSUER", ""), "/"),
		OIDCClientID:              env("OIDC_CLIENT_ID", ""),
		OIDCClientSecret:          env("OIDC_CLIENT_SECRET", ""),
		KeycloakAdminURL:          strings.TrimRight(env("KEYCLOAK_ADMIN_URL", ""), "/"),
		KeycloakAdminClientID:     env("KEYCLOAK_ADMIN_CLIENT_ID", ""),
		KeycloakAdminClientSecret: env("KEYCLOAK_ADMIN_CLIENT_SECRET", ""),
		S3Endpoint:                strings.TrimRight(env("S3_ENDPOINT", ""), "/"),
		S3Bucket:                  env("S3_BUCKET", ""),
		S3AccessKey:               env("S3_ACCESS_KEY", ""),
		S3SecretKey:               env("S3_SECRET_KEY", ""),
	}

	public, err := url.Parse(env("APP_PUBLIC_URL", "http://localhost:8080"))
	if err != nil || public.Scheme == "" || public.Host == "" {
		return Config{}, fmt.Errorf("APP_PUBLIC_URL must be an absolute http(s) URL")
	}
	if public.Scheme != "http" && public.Scheme != "https" {
		return Config{}, fmt.Errorf("APP_PUBLIC_URL must use http or https")
	}
	public.Path = strings.TrimRight(public.Path, "/")
	cfg.PublicURL = public

	var missing []string
	for name, value := range map[string]string{
		"DATABASE_URL":       cfg.DatabaseURL,
		"OIDC_ISSUER":        cfg.OIDCIssuer,
		"OIDC_CLIENT_ID":     cfg.OIDCClientID,
		"OIDC_CLIENT_SECRET": cfg.OIDCClientSecret,
		"S3_ENDPOINT":        cfg.S3Endpoint,
		"S3_BUCKET":          cfg.S3Bucket,
		"S3_ACCESS_KEY":      cfg.S3AccessKey,
		"S3_SECRET_KEY":      cfg.S3SecretKey,
	} {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	if (cfg.KeycloakAdminURL != "" || cfg.KeycloakAdminClientID != "" || cfg.KeycloakAdminClientSecret != "") &&
		(cfg.KeycloakAdminURL == "" || cfg.KeycloakAdminClientID == "" || cfg.KeycloakAdminClientSecret == "") {
		return Config{}, fmt.Errorf("KEYCLOAK_ADMIN_URL, KEYCLOAK_ADMIN_CLIENT_ID and KEYCLOAK_ADMIN_CLIENT_SECRET must be set together")
	}
	if cfg.SessionTimeout < time.Minute {
		return Config{}, fmt.Errorf("APP_SESSION_TIMEOUT must be at least 1m")
	}
	return cfg, nil
}

// RedirectURL returns the fixed OIDC callback URL.
func (c Config) RedirectURL() string {
	return c.PublicURL.String() + "/api/auth/callback"
}

func env(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func envBool(name string, fallback bool) bool {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}
