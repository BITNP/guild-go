package users

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	authz "git.bitnp.net/BITNP/bitnp-authz"
)

const (
	positiveCacheTTL = 10 * time.Minute
	negativeCacheTTL = 30 * time.Second
	maxCacheEntries  = 5000
)

// Identity is display-only Keycloak data for another user.
type Identity struct {
	UserName  string
	Avatar    *string
	Positions []Position
}

type cachedIdentity struct {
	identity  Identity
	found     bool
	expiresAt time.Time
}

// Directory resolves other users' names, avatars and positions. It never
// participates in authorization.
type Directory struct {
	client *KeycloakAdmin
	mu     sync.Mutex
	cache  map[string]cachedIdentity
}

// NewDirectory creates a display-only directory.
func NewDirectory(client *KeycloakAdmin) *Directory {
	return &Directory{client: client, cache: make(map[string]cachedIdentity)}
}

// Lookup resolves a subject. Failures are negatively cached briefly.
func (d *Directory) Lookup(ctx context.Context, subject string) (Identity, bool) {
	if d == nil || !d.client.Enabled() {
		return Identity{}, false
	}
	now := time.Now()
	d.mu.Lock()
	entry, ok := d.cache[subject]
	d.mu.Unlock()
	if ok && now.Before(entry.expiresAt) {
		if entry.found {
			return entry.identity, true
		}
		return Identity{}, false
	}

	user, err := d.client.User(ctx, subject)
	if err != nil {
		slog.Warn("keycloak user lookup failed", "subject", subject, "error", err)
		d.store(subject, Identity{}, false, negativeCacheTTL)
		return Identity{}, false
	}
	roles, err := d.client.RealmRoles(ctx, subject)
	if err != nil {
		slog.Warn("keycloak role lookup failed", "subject", subject, "error", err)
		d.store(subject, Identity{}, false, negativeCacheTTL)
		return Identity{}, false
	}
	identity := Identity{
		UserName:  displayName(user, subject),
		Avatar:    avatarURL(user),
		Positions: positionViews(authz.PositionsFromRoles(roles)),
	}
	d.store(subject, identity, true, positiveCacheTTL)
	return identity, true
}

// LookupOrFallback returns a subject-only identity when Keycloak is unavailable.
func (d *Directory) LookupOrFallback(ctx context.Context, subject string) Identity {
	if identity, ok := d.Lookup(ctx, subject); ok {
		return identity
	}
	return Identity{UserName: subject, Positions: []Position{}}
}

func (d *Directory) store(subject string, identity Identity, found bool, ttl time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.cache) >= maxCacheEntries {
		now := time.Now()
		for key, entry := range d.cache {
			if now.After(entry.expiresAt) {
				delete(d.cache, key)
			}
		}
		if len(d.cache) >= maxCacheEntries {
			d.cache = make(map[string]cachedIdentity)
		}
	}
	d.cache[subject] = cachedIdentity{identity: identity, found: found, expiresAt: time.Now().Add(ttl)}
}

func displayName(user map[string]any, subject string) string {
	first := stringValue(user["firstName"])
	last := stringValue(user["lastName"])
	combined := strings.TrimSpace(first + " " + last)
	if combined != "" {
		return combined
	}
	if username := stringValue(user["username"]); username != "" {
		return username
	}
	return subject
}

func avatarURL(user map[string]any) *string {
	attributes, ok := user["attributes"].(map[string]any)
	if !ok {
		return nil
	}
	value := attributes["picture"]
	if list, ok := value.([]any); ok && len(list) > 0 {
		value = list[0]
	}
	text, _ := value.(string)
	normalized := normalizePicture(text)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func normalizePicture(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 2048 {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	return parsed.String()
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}
