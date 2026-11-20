package users

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	authz "git.bitnp.net/BITNP/bitnp-authz"
	"git.bitnp.net/BITNP/guild-go/internal/apperr"
	"git.bitnp.net/BITNP/guild-go/internal/auth"
)

// Position is a display-only administrative position.
type Position struct {
	Code       string  `json:"code"`
	Label      string  `json:"label"`
	Scope      string  `json:"scope"`
	Department *string `json:"department"`
}

// Summary is a compact user representation.
type Summary struct {
	Subject   string     `json:"subject"`
	UserName  string     `json:"userName"`
	Avatar    *string    `json:"avatar"`
	Positions []Position `json:"positions"`
}

// SummaryMap indexes resolved user summaries by subject.
type SummaryMap map[string]Summary

// FallbackSummary returns a subject-only summary when display metadata is unavailable.
func FallbackSummary(subject string) Summary {
	return Summary{Subject: subject, UserName: subject, Positions: []Position{}}
}

// For returns the resolved summary for subject, or a subject-only fallback.
func (m SummaryMap) For(subject string) Summary {
	if summary, ok := m[subject]; ok {
		return summary
	}
	return FallbackSummary(subject)
}

// Permissions is the signed-in user's authorization view.
type Permissions struct {
	SystemAdmin        bool     `json:"systemAdmin"`
	Manager            bool     `json:"manager"`
	AccessDepartments  []string `json:"accessDepartments"`
	ManagedDepartments []string `json:"managedDepartments"`
}

// Profile is a full user profile. Email and Permissions are self-only.
type Profile struct {
	Summary
	Email       *string      `json:"email"`
	Permissions *Permissions `json:"permissions"`
}

// Service assembles user views from the session and the Keycloak directory.
type Service struct {
	repo      *Repository
	directory *Directory
}

// NewService creates a user service.
func NewService(repo *Repository, directory *Directory) *Service {
	return &Service{repo: repo, directory: directory}
}

// Provision records a verified subject.
func (s *Service) Provision(ctx context.Context, subject string) error {
	return s.repo.Upsert(ctx, subject)
}

// ResolveActorSession builds an authorization session for an internal service
// acting on behalf of subject using live Keycloak user and realm role data.
func (s *Service) ResolveActorSession(ctx context.Context, subject string) (auth.Session, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return auth.Session{}, apperr.BadRequest("必须提供有效的用户 subject")
	}
	session, admitted, err := s.lookupKeycloakSession(ctx, subject)
	if err != nil {
		return auth.Session{}, err
	}
	if !admitted {
		return auth.Session{}, apperr.Forbidden("用户不存在、已禁用或没有公会准入权限")
	}
	if err := s.Provision(ctx, subject); err != nil {
		return auth.Session{}, err
	}
	return session, nil
}

// EnsureSubjects verifies that each non-empty subject is either already known
// locally or exists as an enabled, admitted Keycloak user, provisioning new rows.
func (s *Service) EnsureSubjects(ctx context.Context, subjects []string) error {
	seen := make(map[string]struct{}, len(subjects))
	for _, raw := range subjects {
		subject := strings.TrimSpace(raw)
		if subject == "" {
			continue
		}
		if _, ok := seen[subject]; ok {
			continue
		}
		seen[subject] = struct{}{}

		exists, err := s.repo.Exists(ctx, subject)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		_, admitted, err := s.lookupKeycloakSession(ctx, subject)
		if err != nil {
			return err
		}
		if !admitted {
			return apperr.BadRequest("包含不存在或未准入的用户")
		}
		if err := s.Provision(ctx, subject); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) lookupKeycloakSession(ctx context.Context, subject string) (auth.Session, bool, error) {
	if s.directory == nil || s.directory.client == nil || !s.directory.client.Enabled() {
		return auth.Session{}, false, apperr.New(http.StatusServiceUnavailable, "KEYCLOAK_UNAVAILABLE", "Keycloak Admin API 未配置")
	}
	user, err := s.directory.client.User(ctx, subject)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") {
			return auth.Session{}, false, nil
		}
		return auth.Session{}, false, fmt.Errorf("查询 Keycloak 用户失败: %w", err)
	}
	if enabled, ok := user["enabled"].(bool); ok && !enabled {
		return auth.Session{}, false, nil
	}
	roles, err := s.directory.client.RealmRoles(ctx, subject)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") {
			return auth.Session{}, false, nil
		}
		return auth.Session{}, false, fmt.Errorf("查询 Keycloak 角色失败: %w", err)
	}
	perms := authz.FromRoles(roles)
	if !perms.Admitted {
		return auth.Session{}, false, nil
	}
	var picture string
	if avatar := avatarURL(user); avatar != nil {
		picture = *avatar
	}
	return auth.Session{
		Subject:     subject,
		Name:        displayName(user, subject),
		Picture:     picture,
		Permissions: perms,
	}, true, nil
}

// AllExist reports whether every subject is a known user.
func (s *Service) AllExist(ctx context.Context, subjects []string) (bool, error) {
	return s.repo.AllExist(ctx, subjects)
}

// Current builds the signed-in user's profile.
func (s *Service) Current(session auth.Session) Profile {
	positions := positionViews(session.Permissions.Positions)
	avatar := optionalString(session.Picture)
	email := optionalString(session.Email)
	return Profile{
		Summary: Summary{
			Subject:   session.Subject,
			UserName:  session.Name,
			Avatar:    avatar,
			Positions: positions,
		},
		Email:       email,
		Permissions: permissionsView(session.Permissions),
	}
}

// SummaryFor resolves one subject for display.
func (s *Service) SummaryFor(ctx context.Context, session auth.Session, subject string) Summary {
	if subject == session.Subject {
		return s.Current(session).Summary
	}
	identity := s.directory.LookupOrFallback(ctx, subject)
	return Summary{
		Subject:   subject,
		UserName:  identity.UserName,
		Avatar:    identity.Avatar,
		Positions: identity.Positions,
	}
}

// Summaries resolves many subjects, deduplicating them.
func (s *Service) Summaries(ctx context.Context, session auth.Session, subjects []string) SummaryMap {
	result := make(SummaryMap, len(subjects))
	for _, subject := range subjects {
		if subject == "" {
			continue
		}
		if _, ok := result[subject]; ok {
			continue
		}
		result[subject] = s.SummaryFor(ctx, session, subject)
	}
	return result
}

// Get returns one user profile. Non-self profiles omit permissions and email.
func (s *Service) Get(ctx context.Context, session auth.Session, subject string) (Profile, error) {
	exists, err := s.repo.Exists(ctx, subject)
	if err != nil {
		return Profile{}, err
	}
	if !exists {
		return Profile{}, apperr.NotFound("用户不存在")
	}
	if subject == session.Subject {
		return s.Current(session), nil
	}
	identity := s.directory.LookupOrFallback(ctx, subject)
	return Profile{
		Summary: Summary{
			Subject:   subject,
			UserName:  identity.UserName,
			Avatar:    identity.Avatar,
			Positions: identity.Positions,
		},
	}, nil
}

// List returns the known-user directory.
func (s *Service) List(ctx context.Context, session auth.Session) ([]Summary, error) {
	subjects, err := s.repo.ListSubjects(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Summary, 0, len(subjects))
	for _, subject := range subjects {
		result = append(result, s.SummaryFor(ctx, session, subject))
	}
	return result, nil
}

func positionViews(views []authz.PositionView) []Position {
	result := make([]Position, 0, len(views))
	for _, view := range views {
		var department *string
		if view.Department != nil {
			value := string(*view.Department)
			department = &value
		}
		result = append(result, Position{
			Code:       view.Code,
			Label:      view.Label,
			Scope:      string(view.Scope),
			Department: department,
		})
	}
	return result
}

func permissionsView(permissions authz.SitePermissions) *Permissions {
	access := make([]string, 0, len(permissions.AccessDepartments))
	for _, department := range permissions.AccessDepartments {
		access = append(access, string(department))
	}
	managed := make([]string, 0, len(permissions.ManagedDepartments))
	for _, department := range permissions.ManagedDepartments {
		managed = append(managed, string(department))
	}
	return &Permissions{
		SystemAdmin:        permissions.SystemAdmin,
		Manager:            permissions.Manager,
		AccessDepartments:  access,
		ManagedDepartments: managed,
	}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
