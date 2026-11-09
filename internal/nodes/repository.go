package nodes

import "github.com/jackc/pgx/v5/pgxpool"

// Repository contains explicit SQL for nodes.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a node repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}
