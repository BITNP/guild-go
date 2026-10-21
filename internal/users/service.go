package users

import (
	"context"
	"sort"
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
func (s *Service) Summaries(ctx context.Context, session auth.Session, subjects []string) map[string]Summary {
	result := make(map[string]Summary, len(subjects))
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
	sort.Strings(subjects)
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
