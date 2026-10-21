// Package users owns the known-keycloak-subject index and read-only user
// display data. Authorization never comes from this package.
package users

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository stores the minimal known-user index.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a known-user repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Upsert records a verified Keycloak subject.
func (r *Repository) Upsert(ctx context.Context, subject string) error {
	if subject == "" {
		return fmt.Errorf("subject must not be empty")
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO users (subject) VALUES ($1) ON CONFLICT (subject) DO NOTHING`, subject)
	return err
}

// ListSubjects returns every known subject ordered lexically.
func (r *Repository) ListSubjects(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT subject FROM users ORDER BY subject`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subjects []string
	for rows.Next() {
		var subject string
		if err := rows.Scan(&subject); err != nil {
			return nil, err
		}
		subjects = append(subjects, subject)
	}
	return subjects, rows.Err()
}

// Exists reports whether a subject is known.
func (r *Repository) Exists(ctx context.Context, subject string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE subject=$1)`, subject).Scan(&exists)
	return exists, err
}

// AllExist reports whether every subject is known.
func (r *Repository) AllExist(ctx context.Context, subjects []string) (bool, error) {
	if len(subjects) == 0 {
		return true, nil
	}
	var count int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE subject = ANY($1)`, subjects).Scan(&count)
	if err != nil {
		return false, err
	}
	unique := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		unique[subject] = struct{}{}
	}
	return count == len(unique), nil
}
