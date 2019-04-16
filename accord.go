package accord

import (
	"errors"
	"sync"
)

var (
	// ErrAcquired error is returned if the resource is already acquired.
	ErrAcquired = errors.New("accord: acquired")
	// ErrDone error is returned if the resource is already marked done.
	ErrDone = errors.New("accord: done")
	// ErrClosed error is returned by the handle if closed.
	ErrClosed = errors.New("accord: closed")
)

type metadata struct {
	kv map[string]string
	mu sync.RWMutex
}

// Snap returns a snapshot of metadata.
func (m *metadata) Snap() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snap := make(map[string]string, len(m.kv))
	for k, v := range m.kv {
		snap[k] = v
	}
	return snap
}

// Set sets key.
func (m *metadata) Set(k, v string) {
	m.mu.Lock()
	if m.kv == nil {
		m.kv = make(map[string]string)
	}
	m.kv[k] = v
	m.mu.Unlock()
}

// Merge updates metadata from kv.
func (m *metadata) Update(kv map[string]string) {
	m.mu.Lock()
	if m.kv == nil {
		m.kv = make(map[string]string)
	}
	for k, v := range kv {
		m.kv[k] = v
	}
	m.mu.Unlock()
}
