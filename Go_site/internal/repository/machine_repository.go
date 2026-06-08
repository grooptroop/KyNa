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
            cluster_ip,
            ingress_host,
            resources_preset,
            access_scope,
			container_port,
            service_port,
            image,
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
			&m.ClusterIP,
			&m.IngressHost,
			&m.ResourcesPreset,
			&m.AccessScope,
			&m.ContainerPort,
			&m.ServicePort,
			&m.Image,
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
            cluster_ip,
            ingress_host,
            resources_preset,
            access_scope,
            container_port,
            service_port,
            image
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
        RETURNING id, created_at, updated_at`,
		m.Username,
		m.Name,
		m.Mode,
		m.ServiceKind,
		m.Status,
		m.ExternalIP,
		m.ClusterIP,
		m.IngressHost,
		m.ResourcesPreset,
		m.AccessScope,
		m.ContainerPort,
		m.ServicePort,
		m.Image,
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

func (r *MachineRepository) UpdateStatusIPAndHost(
	ctx context.Context,
	id int64,
	status model.MachineStatus,
	externalIP *string,
	ingressHost *string,
) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE user_machines
        SET status = $2, external_ip = $3, ingress_host = $4, updated_at = now()
        WHERE id = $1`,
		id, status, externalIP, ingressHost,
	)
	if err != nil {
		return fmt.Errorf("update user_machines status/ip/host: %w", err)
	}
	return nil
}

func (r *MachineRepository) UpdateMetadata(ctx context.Context, m *model.UserMachine) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE user_machines
        SET
            name = $1,
            mode = $2,
            service_kind = $3,
            status = $4,
            external_ip = $5,
            cluster_ip = $6,
            ingress_host = $7,
            resources_preset = $8,
            access_scope = $9,
            container_port = $10,
            service_port = $11,
            image = $12,
            updated_at = now()
        WHERE id = $13 AND username = $14
    `,
		m.Name,
		m.Mode,
		m.ServiceKind,
		m.Status,
		m.ExternalIP,
		m.ClusterIP,
		m.IngressHost,
		m.ResourcesPreset,
		m.AccessScope,
		m.ContainerPort,
		m.ServicePort,
		m.Image,
		m.ID,
		m.Username,
	)
	if err != nil {
		return fmt.Errorf("update user_machines metadata: %w", err)
	}
	return nil
}

func (r *MachineRepository) UpdateImage(ctx context.Context, id int64, image string) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE user_machines
        SET image = $2, updated_at = now()
        WHERE id = $1`,
		id, image,
	)
	if err != nil {
		return fmt.Errorf("update user_machines image: %w", err)
	}
	return nil
}

func (r *MachineRepository) InsertHistory(ctx context.Context, m *model.UserMachine, eventType model.UserMachineEventType) error {
	_, err := r.pool.Exec(ctx, `
        INSERT INTO user_machines_history (
            machine_id,
            username,
            name,
            mode,
            service_kind,
            status,
            external_ip,
            cluster_ip,
            ingress_host,
            resources_preset,
            access_scope,
            container_port,
            service_port,
            image,
            event_type
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
        )`,
		m.ID,
		m.Username,
		m.Name,
		m.Mode,
		m.ServiceKind,
		m.Status,
		m.ExternalIP,
		m.ClusterIP,
		m.IngressHost,
		m.ResourcesPreset,
		m.AccessScope,
		m.ContainerPort,
		m.ServicePort,
		m.Image,
		eventType,
	)
	if err != nil {
		return fmt.Errorf("insert user_machines_history: %w", err)
	}
	return nil
}

func (r *MachineRepository) GetByID(ctx context.Context, id int64, username string) (*model.UserMachine, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT
            id,
            username,
            name,
            mode,
            service_kind,
            status,
            external_ip,
            cluster_ip,
            ingress_host,
            resources_preset,
            access_scope,
            container_port,
            service_port,
            image,
            created_at,
            updated_at
        FROM user_machines
        WHERE id = $1 AND username = $2
    `,
		id, username,
	)

	var m model.UserMachine
	if err := row.Scan(
		&m.ID,
		&m.Username,
		&m.Name,
		&m.Mode,
		&m.ServiceKind,
		&m.Status,
		&m.ExternalIP,
		&m.ClusterIP,
		&m.IngressHost,
		&m.ResourcesPreset,
		&m.AccessScope,
		&m.ContainerPort,
		&m.ServicePort,
		&m.Image,
		&m.CreatedAt,
		&m.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("get machine by id: %w", err)
	}
	return &m, nil
}

func (r *MachineRepository) ListHistoryByUsername(ctx context.Context, username string) ([]model.UserMachineHistory, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT
            h.id,
            h.machine_id,
            h.username,
            h.name,
            h.mode,
            h.service_kind,
            h.status,
            h.external_ip,
            h.cluster_ip,
            h.ingress_host,
            h.resources_preset,
            h.access_scope,
            h.container_port,
            h.service_port,
            h.image,
            h.event_type,
            h.occurred_at
        FROM user_machines_history h
        WHERE h.username = $1
        ORDER BY h.occurred_at DESC
        LIMIT 200
    `,
		username,
	)
	if err != nil {
		return nil, fmt.Errorf("list history by username: %w", err)
	}
	defer rows.Close()

	var hs []model.UserMachineHistory
	for rows.Next() {
		var h model.UserMachineHistory
		if err := rows.Scan(
			&h.ID,
			&h.MachineID,
			&h.Username,
			&h.Name,
			&h.Mode,
			&h.ServiceKind,
			&h.Status,
			&h.ExternalIP,
			&h.ClusterIP,
			&h.IngressHost,
			&h.ResourcesPreset,
			&h.AccessScope,
			&h.ContainerPort,
			&h.ServicePort,
			&h.Image,
			&h.EventType,
			&h.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan history row: %w", err)
		}
		hs = append(hs, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history rows err: %w", err)
	}
	return hs, nil
}
