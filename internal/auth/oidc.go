package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDC wraps lazy Keycloak discovery, the authorization-code flow and ID-token
// verification.
type OIDC struct {
	issuer       string
	clientID     string
	clientSecret string
	redirectURL  string
	httpClient   *http.Client

	mu       sync.Mutex
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	config   *oauth2.Config
}

// NewOIDC constructs a lazy OIDC client.
func NewOIDC(issuer, clientID, clientSecret, redirectURL string) *OIDC {
	return &OIDC{
		issuer:       issuer,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (o *OIDC) ensure(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.provider != nil {
		return nil
	}
	discoveryCtx := oidc.ClientContext(ctx, o.httpClient)
	provider, err := oidc.NewProvider(discoveryCtx, o.issuer)
	if err != nil {
		return fmt.Errorf("oidc discovery: %w", err)
	}
	o.provider = provider
	o.verifier = provider.Verifier(&oidc.Config{ClientID: o.clientID})
	o.config = &oauth2.Config{
		ClientID:     o.clientID,
		ClientSecret: o.clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  o.redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "roles"},
	}
	return nil
}

// AuthCodeURL builds the Keycloak authorization URL with nonce and PKCE.
func (o *OIDC) AuthCodeURL(ctx context.Context, state, nonce, verifier string) (string, error) {
	if err := o.ensure(ctx); err != nil {
		return "", err
	}
	o.mu.Lock()
	config := *o.config
	o.mu.Unlock()
	return config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.S256ChallengeOption(verifier),
	), nil
}

// Exchange redeems an authorization code and verifies the returned ID token.
func (o *OIDC) Exchange(ctx context.Context, code, verifier string) (*oidc.IDToken, map[string]any, error) {
	if err := o.ensure(ctx); err != nil {
		return nil, nil, err
	}
	o.mu.Lock()
	config := *o.config
	verifierChecker := o.verifier
	o.mu.Unlock()

	exchangeCtx := context.WithValue(ctx, oauth2.HTTPClient, o.httpClient)
	token, err := config.Exchange(exchangeCtx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, nil, fmt.Errorf("token exchange: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, nil, fmt.Errorf("token response contains no id_token")
	}
	idToken, err := verifierChecker.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, nil, fmt.Errorf("verify id_token: %w", err)
	}
	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, nil, fmt.Errorf("decode id_token claims: %w", err)
	}
	return idToken, claims, nil
}
