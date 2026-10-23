package auth

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	authz "git.bitnp.net/BITNP/bitnp-authz"
)

// SessionCookieName is the browser session cookie.
const SessionCookieName = "guild_session"

// Service implements login, callback and logout.
type Service struct {
	OIDC      *OIDC
	Sessions  *Store
	Pending   *PendingStore
	PublicURL *url.URL
	Provision func(ctx context.Context, subject string) error
	Secure    bool
}

// NewService wires the authentication service.
func NewService(oidcClient *OIDC, sessions *Store, pending *PendingStore, publicURL *url.URL, secure bool, provision func(context.Context, string) error) *Service {
	return &Service{
		OIDC:      oidcClient,
		Sessions:  sessions,
		Pending:   pending,
		PublicURL: publicURL,
		Provision: provision,
		Secure:    secure,
	}
}

// StartLogin begins an authorization-code flow.
func (s *Service) StartLogin(w http.ResponseWriter, r *http.Request) {
	returnTo := SafeReturnTo(r.URL.Query().Get("returnTo"))
	state, err := RandomToken(32)
	if err != nil {
		s.redirectError(w, r, "oidc_failed")
		return
	}
	nonce, err := RandomToken(32)
	if err != nil {
		s.redirectError(w, r, "oidc_failed")
		return
	}
	verifier, err := RandomToken(48)
	if err != nil {
		s.redirectError(w, r, "oidc_failed")
		return
	}
	authURL, err := s.OIDC.AuthCodeURL(r.Context(), state, nonce, verifier)
	if err != nil {
		slog.Warn("OIDC discovery failed", "error", err)
		s.redirectError(w, r, "provider_unavailable")
		return
	}
	s.Pending.Put(state, PendingLogin{Nonce: nonce, Verifier: verifier, ReturnTo: returnTo})
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback finishes the authorization-code flow.
func (s *Service) Callback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("error") != "" {
		s.redirectError(w, r, "oidc_failed")
		return
	}
	state := query.Get("state")
	code := query.Get("code")
	if state == "" || code == "" {
		s.redirectError(w, r, "oidc_failed")
		return
	}
	pending, ok := s.Pending.Take(state)
	if !ok {
		s.redirectError(w, r, "oidc_failed")
		return
	}
	idToken, claims, err := s.OIDC.Exchange(r.Context(), code, pending.Verifier)
	if err != nil {
		slog.Warn("OIDC callback failed", "error", err)
		s.redirectError(w, r, "oidc_failed")
		return
	}
	if idToken.Nonce != pending.Nonce {
		slog.Warn("OIDC nonce mismatch")
		s.redirectError(w, r, "oidc_failed")
		return
	}
	roles := authz.RealmRolesFromClaims(claims)
	if len(roles) == 0 {
		s.redirectError(w, r, "no_roles")
		return
	}
	permissions := authz.FromClaims(claims)
	if !permissions.Admitted {
		s.redirectError(w, r, "not_member")
		return
	}
	subject := idToken.Subject
	if subject == "" {
		s.redirectError(w, r, "oidc_failed")
		return
	}
	if err := s.Provision(r.Context(), subject); err != nil {
		slog.Error("provision known user failed", "subject", subject, "error", err)
		s.redirectError(w, r, "oidc_failed")
		return
	}
	name := displayName(claims, subject)
	email, _ := claims["email"].(string)
	email = strings.TrimSpace(email)
	picture := safePicture(claims)
	sessionID, _, err := s.Sessions.Create(subject, name, email, picture, permissions)
	if err != nil {
		slog.Error("create session failed", "error", err)
		s.redirectError(w, r, "oidc_failed")
		return
	}
	s.setSessionCookie(w, sessionID)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, SafeReturnTo(pending.ReturnTo), http.StatusFound)
}

// Logout deletes the current session and clears the cookie.
func (s *Service) Logout(w http.ResponseWriter, r *http.Request, sessionID string) {
	s.Sessions.Delete(sessionID)
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) setSessionCookie(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Service) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func (s *Service) redirectError(w http.ResponseWriter, r *http.Request, code string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, "/login?error="+url.QueryEscape(code), http.StatusFound)
}

func displayName(claims map[string]any, subject string) string {
	for _, key := range []string{"name", "preferred_username"} {
		if value, ok := claims[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return subject
}

func safePicture(claims map[string]any) string {
	picture, _ := claims["picture"].(string)
	return SafePictureURL(picture)
}

// SafePictureURL accepts only absolute http(s) avatar URLs.
func SafePictureURL(value string) string {
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
