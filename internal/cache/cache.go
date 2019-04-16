package cache

// Cache interface.
type Cache interface {
	// Contains checks if an entry exists.
	Contains(entry string) (bool, error)
	// Add adds an entry.
	Add(entry string) error
	// AddBatch entry
	AddBatch() (BatchWriter, error)
	// Close closes and wipes the cache
	Close() error
}

// BatchWriter adds batches of entries.
type BatchWriter interface {
	// Add adds an entry to the batch.
	Add(entry string) error
	// Flush flushes the batch.
	Flush() error
	// Discard discards the batch.
	Discard() error
}
