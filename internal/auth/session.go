package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	authz "git.bitnp.net/BITNP/bitnp-authz"
)

// Session is the server-side state attached to a browser cookie.
type Session struct {
	ID          string
	Subject     string
	Name        string
	Email       string
	Picture     string
	Permissions authz.SitePermissions
	CSRF        string
	ExpiresAt   time.Time
}

// Store keeps sessions in memory. The application is deliberately single-instance.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]Session
	timeout  time.Duration
}

// NewStore creates an empty session store.
func NewStore(timeout time.Duration) *Store {
	return &Store{sessions: make(map[string]Session), timeout: timeout}
}

// Timeout returns the sliding session lifetime.
func (s *Store) Timeout() time.Duration {
	return s.timeout
}

// Create stores a new session and returns its identifier and CSRF token.
func (s *Store) Create(subject, name, email, picture string, permissions authz.SitePermissions) (string, string, error) {
	id, err := RandomToken(32)
	if err != nil {
		return "", "", err
	}
	csrf, err := RandomToken(32)
	if err != nil {
		return "", "", err
	}
	session := Session{
		ID:          id,
		Subject:     subject,
		Name:        name,
		Email:       email,
		Picture:     picture,
		Permissions: permissions,
		CSRF:        csrf,
		ExpiresAt:   time.Now().Add(s.timeout),
	}
	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()
	return id, csrf, nil
}

// Get returns a session and slides its expiry. Expired sessions are removed.
func (s *Store) Get(id string) (Session, bool) {
	if id == "" {
		return Session{}, false
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return Session{}, false
	}
	if now.After(session.ExpiresAt) {
		delete(s.sessions, id)
		return Session{}, false
	}
	session.ExpiresAt = now.Add(s.timeout)
	s.sessions[id] = session
	return session, true
}

// Delete removes a session.
func (s *Store) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// Cleanup removes expired sessions.
func (s *Store) Cleanup() {
	now := time.Now()
	s.mu.Lock()
	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
	s.mu.Unlock()
}

// StartCleanup runs periodic cleanup until ctx is cancelled.
func (s *Store) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Cleanup()
		}
	}
}

// RandomToken returns a URL-safe random token.
func RandomToken(bytes int) (string, error) {
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
