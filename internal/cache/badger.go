package cache

import (
	"os"

	"github.com/dgraph-io/badger"
)

type badgerCache struct {
	*badger.DB
}

// OpenBadger opens a badger DB cache.
func OpenBadger(dir string) (Cache, error) {
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return nil, err
	}

	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	opts.Logger = nil
	opts.Truncate = true

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &badgerCache{DB: db}, nil
}

// Add implements Cache interface.
func (c *badgerCache) Add(name string) error {
	return c.Update(func(tx *badger.Txn) error {
		return tx.Set([]byte(name), nil)
	})
}

// AddBatch implements Cache interface.
func (c *badgerCache) AddBatch() (BatchWriter, error) {
	return &badgerBatchWriter{
		WriteBatch: c.DB.NewWriteBatch(),
	}, nil
}

// Contains implements Cache interface.
func (c *badgerCache) Contains(name string) (bool, error) {
	var found bool
	err := c.View(func(tx *badger.Txn) error {
		_, err := tx.Get([]byte(name))
		if err == badger.ErrKeyNotFound {
			return nil
		}

		found = err == nil
		return err
	})
	return found, err
}

// --------------------------------------------------------------------

type badgerBatchWriter struct {
	*badger.WriteBatch
}

// Add implements BatchWriter interface.
func (c *badgerBatchWriter) Add(name string) error {
	return c.Set([]byte(name), nil, 0)
}

// Add implements BatchWriter interface.
func (c *badgerBatchWriter) Discard() error {
	c.Cancel()
	return nil
}
