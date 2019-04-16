package postgres

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/bsm/accord/backend"
	"github.com/lib/pq"
)

func performUpdate(ctx context.Context, stmt sq.UpdateBuilder) error {
	res, err := stmt.ExecContext(ctx)
	if err != nil {
		return err
	}

	num, err := res.RowsAffected()
	if err != nil {
		return err
	} else if num == 0 {
		return backend.ErrInvalidHandle
	}
	return nil
}

func scanHandle(row sq.RowScanner) (*backend.HandleData, error) {
	var (
		maybeDone pq.NullTime
		handle    backend.HandleData
	)
	if err := row.Scan(
		&handle.ID,
		&handle.Namespace,
		&handle.Name,
		&handle.Owner,
		&handle.ExpTime,
		&handle.NumAcquired,
		&maybeDone,
		(*metaJSONb)(&handle.Metadata),
	); err != nil {
		return nil, err
	}

	if maybeDone.Valid {
		handle.DoneTime = maybeDone.Time
	}
	return &handle, nil
}

// --------------------------------------------------------------------

type metaJSONb map[string]string

func (m *metaJSONb) Scan(v interface{}) error {
	switch vv := v.(type) {
	case string:
		return m.Scan([]byte(vv))
	case []byte:
		return json.Unmarshal(vv, m)
	default:
		return errors.New("accord: invalid metadata type")
	}
}

// Value implements the driver Valuer interface.
func (m metaJSONb) Value() (driver.Value, error) {
	if len(m) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}
