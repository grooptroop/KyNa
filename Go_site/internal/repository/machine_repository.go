package repository

import (
	"context"
	"fmt"

	"github.com/grooptroop/KyNa/Go_site/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MachineRepository struct {
	pool *pgxpool.Pool
}

func NewMachineRepository(pool *pgxpool.Pool) *MachineRepository {
	return &MachineRepository{pool: pool}
}
func (r *MachineRepository) ListByUsername(ctx context.Context, username string) ([]model.UserMachine, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT
            id,
            username,
            name,
            mode,
            service_kind,
            status,
            external_ip,
            resources_preset,
            created_at,
            updated_at
        FROM user_machines
        WHERE username = $1
        ORDER BY id DESC`,
		username,
	)
	if err != nil {
		return nil, fmt.Errorf("list machines query: %w", err)
	}
	defer rows.Close()

	var ms []model.UserMachine
	for rows.Next() {
		var m model.UserMachine
		if err := rows.Scan(
			&m.ID,
			&m.Username,
			&m.Name,
			&m.Mode,
			&m.ServiceKind,
			&m.Status,
			&m.ExternalIP,
			&m.ResourcesPreset,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan machine row: %w", err)
		}
		ms = append(ms, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("machines rows err: %w", err)
	}
	return ms, nil
}

func (r *MachineRepository) Create(ctx context.Context, m *model.UserMachine) error {
	if m.Username == "" {
		return fmt.Errorf("empty username in MachineRepository.Create")
	}
	if m.Name == "" {
		return fmt.Errorf("empty name in MachineRepository.Create")
	}

	err := r.pool.QueryRow(ctx, `
    INSERT INTO user_machines (
        username,
        name,
        mode,
        service_kind,
        status,
        external_ip,
        resources_preset
    )
    VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING id, created_at, updated_at`,
		m.Username,
		m.Name,
		m.Mode,
		m.ServiceKind,
		m.Status,
		m.ExternalIP,
		m.ResourcesPreset, // вот это важное поле
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert user_machines: %w", err)
	}
	return nil
}

func (r *MachineRepository) UpdateStatusAndIP(
	ctx context.Context,
	id int64,
	status model.MachineStatus,
	externalIP *string,
) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE user_machines
        SET status = $2, external_ip = $3, updated_at = now()
        WHERE id = $1`,
		id, status, externalIP,
	)
	if err != nil {
		return fmt.Errorf("update user_machines status/ip: %w", err)
	}
	return nil
}

// DeleteByID удаляет машинку по id и username (чтобы не удалить чужую)
func (r *MachineRepository) DeleteByID(ctx context.Context, id int64, username string) error {
	cmdTag, err := r.pool.Exec(ctx, `
        DELETE FROM user_machines
        WHERE id = $1 AND username = $2`,
		id, username,
	)
	if err != nil {
		return fmt.Errorf("delete machine: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("delete machine: no rows affected")
	}
	return nil
}
