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
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_provisions_status
  ON user_provisions(status);
`

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, createUserProvisionsTable)
	return err
}
