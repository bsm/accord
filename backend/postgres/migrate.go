package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

var migrateV1 = []string{
	`CREATE TABLE meta_info (
			version INT NOT NULL DEFAULT 0
		)`,
	`INSERT INTO meta_info (version) VALUES (0)`,
	`CREATE TABLE resource_handles (
			id UUID PRIMARY KEY,
			namespace VARCHAR(100) NOT NULL,
			name VARCHAR(255) NOT NULL,
			owner VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			done_at TIMESTAMP WITH TIME ZONE,
			num_acquired INT NOT NULL DEFAULT 0,
			metadata JSONB NOT NULL,

			UNIQUE (namespace, name)
		)`,
	`CREATE INDEX resource_handles_owner ON resource_handles USING btree (owner)`,
	`CREATE INDEX resource_handles_expires_at ON resource_handles USING btree (expires_at)`,
	`CREATE INDEX resource_handles_done_at ON resource_handles USING btree (done_at)`,
	`CREATE INDEX resource_handles_metadata ON resource_handles USING gin (metadata)`,
}

var migrateV2 = []string{
	`ALTER TABLE resource_handles ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()`,
	`CREATE INDEX resource_handles_updated_at ON resource_handles USING btree (updated_at)`,
}

func migrateUp(ctx context.Context, db *sql.DB, version int, queries []string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, sql := range queries {
		if _, err := tx.ExecContext(ctx, sql); err != nil {
			return fmt.Errorf("migration (v%d) query %q failed: %v", version, sql, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE meta_info SET version = $1`, version); err != nil {
		return fmt.Errorf("migration (v%d) failed: %v", version, err)
	}
	return tx.Commit()
}
