package auth

import (
	"context"
	"sync"
	"time"
)

// PendingLogin is the short-lived state of an authorization-code flow.
type PendingLogin struct {
	Nonce     string
	Verifier  string
	ReturnTo  string
	ExpiresAt time.Time
}

// PendingStore holds in-flight OIDC logins.
type PendingStore struct {
	mu      sync.Mutex
	entries map[string]PendingLogin
	ttl     time.Duration
}

// NewPendingStore creates an empty pending-login store.
func NewPendingStore() *PendingStore {
	return &PendingStore{entries: make(map[string]PendingLogin), ttl: 10 * time.Minute}
}

// Put stores a pending login keyed by OIDC state.
func (p *PendingStore) Put(state string, entry PendingLogin) {
	entry.ExpiresAt = time.Now().Add(p.ttl)
	p.mu.Lock()
	p.entries[state] = entry
	p.mu.Unlock()
}

// Take removes and returns a pending login; entries are single-use.
func (p *PendingStore) Take(state string) (PendingLogin, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.entries[state]
	if !ok {
		return PendingLogin{}, false
	}
	delete(p.entries, state)
	if time.Now().After(entry.ExpiresAt) {
		return PendingLogin{}, false
	}
	return entry, true
}

// Cleanup removes expired pending logins.
func (p *PendingStore) Cleanup() {
	now := time.Now()
	p.mu.Lock()
	for state, entry := range p.entries {
		if now.After(entry.ExpiresAt) {
			delete(p.entries, state)
		}
	}
	p.mu.Unlock()
}

// StartCleanup periodically removes expired pending logins.
func (p *PendingStore) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.Cleanup()
		}
	}
}
