// Package mock implements an in-memory mock backend for testing.
package mock

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/bsm/accord"
	"github.com/bsm/accord/backend"
	"github.com/bsm/accord/rpc"
	"github.com/google/uuid"
)

var _ backend.Backend = (*Backend)(nil)

type fullName struct {
	Namespace, Name string
}

// Backend implements a mock backend.
type Backend struct {
	byName map[fullName]*backend.HandleData
	byID   map[uuid.UUID]*backend.HandleData
	asList []*backend.HandleData
	mu     sync.RWMutex
}

// New opens a mock backend
func New() *Backend {
	return &Backend{
		byName: make(map[fullName]*backend.HandleData),
		byID:   make(map[uuid.UUID]*backend.HandleData),
	}
}

// Get returns the stored handle data.
func (b *Backend) Get(_ context.Context, handleID uuid.UUID) (*backend.HandleData, error) {
	b.mu.RLock()
	stored := b.byID[handleID]
	b.mu.RUnlock()
	return stored, nil
}

// Acquire implements the backend.Backend interface.
func (b *Backend) Acquire(_ context.Context, owner, namespace, name string, exp time.Time, metadata map[string]string) (*backend.HandleData, error) {
	key := fullName{Namespace: namespace, Name: name}
	now := time.Now()

	b.mu.RLock()
	stored, ok := b.byName[key]
	b.mu.RUnlock()

	if ok && stored.IsDone() {
		return nil, accord.ErrDone
	} else if ok && !stored.ExpTime.Before(now) {
		return nil, accord.ErrAcquired
	}

	handle := &backend.HandleData{
		ID:          uuid.New(),
		Namespace:   namespace,
		Name:        name,
		ExpTime:     exp,
		NumAcquired: 1,
		Owner:       owner,
		Metadata:    metadata,
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if stored, ok := b.byName[key]; ok && stored.IsDone() {
		return nil, accord.ErrDone
	} else if ok && !stored.ExpTime.Before(now) {
		return nil, accord.ErrAcquired
	} else if ok {
		handle.NumAcquired = stored.NumAcquired + 1
		handle.UpdateMetadata(stored.Metadata)
	}

	b.byID[handle.ID] = handle
	b.byName[key] = handle
	b.asList = append(b.asList, handle)

	return handle, nil
}

// Renew implements the backend.Backend interface.
func (b *Backend) Renew(_ context.Context, owner string, handleID uuid.UUID, exp time.Time, metadata map[string]string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if stored, ok := b.byID[handleID]; !ok || stored.IsDone() || stored.Owner != owner {
		return backend.ErrInvalidHandle
	} else {
		stored.UpdateMetadata(metadata)
		stored.ExpTime = exp
	}
	return nil
}

// Done implements the backend.Backend interface.
func (b *Backend) Done(_ context.Context, owner string, handleID uuid.UUID, metadata map[string]string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if stored, ok := b.byID[handleID]; !ok || stored.IsDone() || stored.Owner != owner {
		return backend.ErrInvalidHandle
	} else {
		stored.UpdateMetadata(metadata)
		stored.DoneTime = time.Now()
	}
	return nil
}

// List implements the backend.Backend interface.
func (b *Backend) List(_ context.Context, req *rpc.ListRequest, iter backend.Iterator) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	filter := req.GetFilter()
	for i := len(b.asList) - 1 - int(req.GetOffset()); i >= 0; i-- {
		if handle := b.asList[i]; filter == nil || isSelected(filter, handle) {
			if err := iter(handle); err == backend.ErrIteratorDone {
				break
			} else if err != nil {
				return err
			}
		}
	}
	return nil
}

// Ping implements the backend.Backend interface.
func (*Backend) Ping() error { return nil }

// Close implements the backend.Backend interface.
func (*Backend) Close() error { return nil }

func isSelected(filter *rpc.ListRequest_Filter, handle *backend.HandleData) bool {
	if filter.Status == rpc.ListRequest_Filter_DONE && !handle.IsDone() {
		return false
	} else if filter.Status == rpc.ListRequest_Filter_PENDING && handle.IsDone() {
		return false
	}

	if filter.Prefix != "" && !strings.HasPrefix(handle.Namespace, filter.Prefix) {
		return false
	}

	if len(filter.Metadata) != 0 {
		if len(handle.Metadata) == 0 {
			return false
		}
		for k, v := range filter.Metadata {
			if handle.Metadata[k] != v {
				return false
			}
		}
	}

	return true
}
