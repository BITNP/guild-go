package users

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// KeycloakAdmin is a minimal read-only Keycloak Admin API client.
type KeycloakAdmin struct {
	baseURL      string
	clientID     string
	clientSecret string
	tokenURL     string
	httpClient   *http.Client

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

// NewKeycloakAdmin creates a client. An empty baseURL disables it.
func NewKeycloakAdmin(baseURL, clientID, clientSecret, issuer string) *KeycloakAdmin {
	return &KeycloakAdmin{
		baseURL:      strings.TrimRight(baseURL, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     strings.TrimRight(issuer, "/") + "/protocol/openid-connect/token",
		httpClient:   &http.Client{Timeout: 8 * time.Second},
	}
}

// Enabled reports whether Admin API lookups are configured.
func (k *KeycloakAdmin) Enabled() bool {
	return k != nil && k.baseURL != "" && k.clientID != "" && k.clientSecret != ""
}

// User returns a Keycloak user representation.
func (k *KeycloakAdmin) User(ctx context.Context, subject string) (map[string]any, error) {
	if !k.Enabled() {
		return nil, fmt.Errorf("keycloak admin api is disabled")
	}
	var result map[string]any
	if err := k.get(ctx, "/users/"+url.PathEscape(subject), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// RealmRoles returns a user's effective realm role names.
func (k *KeycloakAdmin) RealmRoles(ctx context.Context, subject string) ([]string, error) {
	if !k.Enabled() {
		return nil, fmt.Errorf("keycloak admin api is disabled")
	}
	var result []map[string]any
	if err := k.get(ctx, "/users/"+url.PathEscape(subject)+"/role-mappings/realm/composite", &result); err != nil {
		return nil, err
	}
	roles := make([]string, 0, len(result))
	for _, role := range result {
		if name, ok := role["name"].(string); ok && strings.TrimSpace(name) != "" {
			roles = append(roles, name)
		}
	}
	return roles, nil
}

func (k *KeycloakAdmin) get(ctx context.Context, path string, target any) error {
	token, err := k.accessToken(ctx)
	if err != nil {
		return err
	}
	status, err := k.doGet(ctx, path, token, target)
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized {
		k.invalidate()
		token, err = k.accessToken(ctx)
		if err != nil {
			return err
		}
		status, err = k.doGet(ctx, path, token, target)
		if err != nil {
			return err
		}
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("keycloak admin api returned status %d", status)
	}
	return nil
}

func (k *KeycloakAdmin) doGet(ctx context.Context, path, token string, target any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.baseURL+path, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := k.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return resp.StatusCode, err
	}
	return resp.StatusCode, nil
}

func (k *KeycloakAdmin) accessToken(ctx context.Context) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.token != "" && time.Now().Before(k.expiresAt) {
		return k.token, nil
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, k.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(k.clientID+":"+k.clientSecret)))
	resp, err := k.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("keycloak token endpoint returned status %d", resp.StatusCode)
	}
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("keycloak token response contains no access token")
	}
	expiresIn := body.ExpiresIn
	if expiresIn < 60 {
		expiresIn = 60
	}
	k.token = body.AccessToken
	k.expiresAt = time.Now().Add(time.Duration(expiresIn-30) * time.Second)
	return k.token, nil
}

func (k *KeycloakAdmin) invalidate() {
	k.mu.Lock()
	k.token = ""
	k.expiresAt = time.Time{}
	k.mu.Unlock()
}
