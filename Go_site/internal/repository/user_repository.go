package repository

import (
	"context"

	"github.com/grooptroop/KyNa/Go_site/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) List(ctx context.Context) ([]model.UserProvision, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, username, domain, mode, status, created_at, updated_at
        FROM user_provisions
        ORDER BY id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.UserProvision
	for rows.Next() {
		var u model.UserProvision
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Domain, &u.Mode, &u.Status,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

func (r *UserRepository) Create(ctx context.Context, u *model.UserProvision) error {
	return r.pool.QueryRow(ctx, `
        INSERT INTO user_provisions (username, domain, mode, status)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at, updated_at`,
		u.Username, u.Domain, u.Mode, u.Status,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}
