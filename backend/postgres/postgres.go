// Package postgres implements a postgres-based backend for storing state.
package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/bsm/accord"
	"github.com/bsm/accord/backend"
	"github.com/bsm/accord/rpc"
	"github.com/google/uuid"
)

type postgres struct {
	*sql.DB
	stmt  sq.StatementBuilderType
	ownDB bool
}

// Open connects to the backend
func Open(ctx context.Context, driver, dsn string) (backend.Backend, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	b, err := OpenDB(ctx, db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	b.(*postgres).ownDB = true
	return b, nil
}

// OpenDB connects to the backend
func OpenDB(ctx context.Context, db *sql.DB) (backend.Backend, error) {
	b := &postgres{
		DB:   db,
		stmt: sq.StatementBuilder.RunWith(db).PlaceholderFormat(sq.Dollar),
	}
	if err := b.migrate(ctx); err != nil {
		return nil, err
	}
	return b, nil
}

// Acquire implements the backend.Backend interface.
func (b *postgres) Acquire(ctx context.Context, owner, namespace, name string, exp time.Time, metadata map[string]string) (*backend.HandleData, error) {
	handleID := uuid.New()
	now := time.Now().UTC()
	stmt := b.stmt.Insert("resource_handles").
		Columns(
			"id",
			"namespace",
			"name",
			"owner",
			"expires_at",
			"num_acquired",
			"created_at",
			"updated_at",
			"metadata",
		).
		Values(
			handleID,
			namespace,
			name,
			owner,
			exp,
			1,
			now,
			now,
			metaJSONb(metadata),
		).
		Suffix(`
			ON CONFLICT (namespace, name) DO UPDATE SET
				id           = CASE WHEN resource_handles.expires_at < ? AND resource_handles.done_at IS NULL THEN ? ELSE resource_handles.id END,
				owner        = CASE WHEN resource_handles.expires_at < ? AND resource_handles.done_at IS NULL THEN ? ELSE resource_handles.owner END,
				expires_at   = CASE WHEN resource_handles.expires_at < ? AND resource_handles.done_at IS NULL THEN ? ELSE resource_handles.expires_at END,
				num_acquired = CASE WHEN resource_handles.expires_at < ? AND resource_handles.done_at IS NULL THEN resource_handles.num_acquired + 1 ELSE resource_handles.num_acquired END,
				updated_at   = ?
			RETURNING
				id,
				namespace,
				name,
				owner,
				expires_at,
				num_acquired,
				done_at,
				metadata
		`, now, handleID, now, owner, now, exp.UTC(), now, now)

	handle, err := scanHandle(stmt.QueryRowContext(ctx))
	if err != nil {
		return nil, err
	}
	if handle.IsDone() {
		return nil, accord.ErrDone
	}
	if handle.Owner != owner || !bytes.Equal(handle.ID[:], handleID[:]) {
		return nil, accord.ErrAcquired
	}
	return handle, nil
}

// Get implements the backend.Backend interface.
func (b *postgres) Get(ctx context.Context, handleID uuid.UUID) (*backend.HandleData, error) {
	stmt := b.stmt.
		Select(
			"id",
			"namespace",
			"name",
			"owner",
			"expires_at",
			"num_acquired",
			"done_at",
			"metadata",
		).
		From("resource_handles").
		Where(sq.Eq{"id": handleID})

	handle, err := scanHandle(stmt.QueryRowContext(ctx))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return handle, nil
}

// List implements the backend.Backend interface.
func (b *postgres) List(ctx context.Context, req *rpc.ListRequest, iter backend.Iterator) error {
	stmt := b.stmt.
		Select(
			"id",
			"namespace",
			"name",
			"owner",
			"expires_at",
			"num_acquired",
			"done_at",
			"metadata",
		).
		From("resource_handles").
		OrderBy("created_at DESC")

	if o := req.GetOffset(); o != 0 {
		stmt = stmt.Offset(o)
	}

	if f := req.GetFilter(); f != nil {
		if f.Status == rpc.ListRequest_Filter_DONE {
			stmt = stmt.Where(sq.NotEq{"done_at": nil})
		} else if f.Status == rpc.ListRequest_Filter_PENDING {
			stmt = stmt.Where(sq.Eq{"done_at": nil})
		}
		if f.Prefix != "" {
			stmt = stmt.Where(sq.Like{"namespace": f.Prefix + "%"})
		}
		if len(f.Metadata) != 0 {
			metadata, _ := json.Marshal(f.Metadata)
			stmt = stmt.Where("metadata @> ?", metadata)
		}
	}

	rows, err := stmt.QueryContext(ctx)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		handle, err := scanHandle(rows)
		if err != nil {
			return err
		}

		if err := iter(handle); err == backend.ErrIteratorDone {
			break
		} else if err != nil {
			return err
		}
	}
	return rows.Err()
}

// Renew implements the backend.Backend interface.
func (b *postgres) Renew(ctx context.Context, owner string, handleID uuid.UUID, exp time.Time, metadata map[string]string) error {
	now := time.Now().UTC()
	stmt := b.stmt.Update("resource_handles").
		Set("expires_at", exp.UTC()).
		Set("updated_at", now).
		Set("metadata", sq.Expr(`(metadata || ?)`, metaJSONb(metadata))).
		Where(sq.Eq{
			"id":      handleID,
			"owner":   owner,
			"done_at": nil,
		})
	return performUpdate(ctx, stmt)
}

// Done implements the backend.Backend interface.
func (b *postgres) Done(ctx context.Context, owner string, handleID uuid.UUID, metadata map[string]string) error {
	now := time.Now().UTC()
	stmt := b.stmt.Update("resource_handles").
		Set("done_at", now).
		Set("updated_at", now).
		Set("metadata", sq.Expr(`(metadata || ?)`, metaJSONb(metadata))).
		Where(sq.Eq{
			"id":      handleID,
			"owner":   owner,
			"done_at": nil,
		})
	return performUpdate(ctx, stmt)
}

// Ping implements the backend.Backend interface.
func (b *postgres) Ping() error { return b.DB.Ping() }

// Close implements the backend.Backend interface.
func (b *postgres) Close() error {
	if b.ownDB {
		return b.DB.Close()
	}
	return nil
}

func (b *postgres) migrate(ctx context.Context) error {
	var version int64
	_ = b.stmt.Select("version").From("meta_info").ScanContext(ctx, &version)

	if version < 1 {
		if err := migrateUp(ctx, b.DB, 1, migrateV1); err != nil {
			return err
		}
	}
	if version < 2 {
		if err := migrateUp(ctx, b.DB, 2, migrateV2); err != nil {
			return err
		}
	}
	return nil
}
