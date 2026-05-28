package repository

import (
	"context"
	"fmt"

	"github.com/grooptroop/KyNa/Go_site/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) Create(ctx context.Context, a *model.Account) error {
	return r.pool.QueryRow(ctx, `
        INSERT INTO accounts (username, email, password_hash, role)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at`,
		a.Username, a.Email, a.PasswordHash, a.Role,
	).Scan(&a.ID, &a.CreatedAt)
}

func (r *AccountRepository) FindByUsername(ctx context.Context, username string) (*model.Account, error) {
	var a model.Account
	err := r.pool.QueryRow(ctx, `
        SELECT id, username, email, password_hash, role, created_at
        FROM accounts
        WHERE username = $1`,
		username,
	).Scan(
		&a.ID,
		&a.Username,
		&a.Email,
		&a.PasswordHash,
		&a.Role,
		&a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find account by username: %w", err)
	}
	return &a, nil
}
