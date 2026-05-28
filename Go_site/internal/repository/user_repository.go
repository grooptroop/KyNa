package repository

import (
	"context"
	"fmt"

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
        SELECT id, username, domain, mode, status, external_ip, created_at, updated_at
        FROM user_provisions
        ORDER BY id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list users query: %w", err)
	}
	defer rows.Close()

	var users []model.UserProvision
	for rows.Next() {
		var u model.UserProvision
		if err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Domain,
			&u.Mode,
			&u.Status,
			&u.ExternalIP,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}

	return users, nil
}

func (r *UserRepository) Create(ctx context.Context, u *model.UserProvision) error {
	return r.pool.QueryRow(ctx, `
        INSERT INTO user_provisions (username, domain, mode, status, external_ip)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at, updated_at`,
		u.Username, u.Domain, u.Mode, u.Status, u.ExternalIP,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *UserRepository) DeleteByUsername(ctx context.Context, username string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM user_provisions WHERE username = $1`, username)
	return err
}

func (r *UserRepository) UpdateStatusAndIP(
	ctx context.Context,
	username string,
	status model.ProvisionStatus,
	externalIP *string,
) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE user_provisions
        SET status = $2, external_ip = $3, updated_at = now()
        WHERE username = $1`,
		username, status, externalIP,
	)
	return err
}

func (r *UserRepository) ListAdminUsers(ctx context.Context) ([]model.AdminUserView, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT
            a.username,
            a.email,
            a.role,
            up.domain,
            up.mode,
            up.status,
            up.external_ip
        FROM accounts a
        LEFT JOIN user_provisions up ON up.username = a.username
        ORDER BY a.id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list admin users query: %w", err)
	}
	defer rows.Close()

	var res []model.AdminUserView

	for rows.Next() {
		var u model.AdminUserView
		if err := rows.Scan(
			&u.Username,
			&u.Email,
			&u.Role,
			&u.Domain,
			&u.Mode,
			&u.Status,
			&u.ExternalIP,
		); err != nil {
			return nil, fmt.Errorf("scan admin user row: %w", err)
		}
		res = append(res, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("admin users rows err: %w", err)
	}

	return res, nil
}
