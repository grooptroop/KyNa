package migrations

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const createUserProvisionsTable = `
CREATE TABLE IF NOT EXISTS user_provisions (
    id          BIGSERIAL PRIMARY KEY,
    username    TEXT NOT NULL UNIQUE,
    domain      TEXT NOT NULL,
    mode        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    external_ip TEXT,
    access_scope TEXT,
    cluster_ip TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_provisions_status
  ON user_provisions(status);
`

const createUserMachinesTable = `
CREATE TABLE IF NOT EXISTS user_machines (
    id           BIGSERIAL PRIMARY KEY,
    username     TEXT NOT NULL REFERENCES user_provisions(username) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    mode         TEXT NOT NULL,
    service_kind TEXT NOT NULL DEFAULT 'web',
    version      TEXT,
    status       TEXT NOT NULL DEFAULT 'pending',
    external_ip  TEXT,
    ingress_host text,
    container_port integer,
    service_port integer,
    image text,
    resources_preset TEXT NOT NULL DEFAULT 'small',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_machines_username
  ON user_machines(username);

CREATE INDEX IF NOT EXISTS idx_user_machines_status
  ON user_machines(status);
`

const createAccountsTable = `
CREATE TABLE IF NOT EXISTS accounts (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'user',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

const createUserMachinesHistoryTable = `
CREATE TABLE IF NOT EXISTS user_machines_history (
    id              BIGSERIAL PRIMARY KEY,
    machine_id      BIGINT, 
    username        TEXT        NOT NULL,
    name            TEXT        NOT NULL,
    mode            TEXT        NOT NULL,
    service_kind    TEXT        NOT NULL,
    status          TEXT        NOT NULL,
    external_ip     TEXT,
    cluster_ip      TEXT,
    ingress_host    TEXT,
    resources_preset TEXT       NOT NULL,
    access_scope    TEXT        NOT NULL,
    container_port  INTEGER     NOT NULL,
    service_port    INTEGER     NOT NULL,
    image           TEXT,
    event_type      TEXT        NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_machines_history_machine
  ON user_machines_history (machine_id, occurred_at DESC);
`

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, createUserProvisionsTable); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, createUserMachinesTable); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, createUserMachinesHistoryTable); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, createAccountsTable); err != nil {
		return err
	}
	return nil
}
