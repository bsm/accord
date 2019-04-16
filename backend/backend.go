package backend

import (
	"context"
	"errors"
	"time"

	"github.com/bsm/accord/internal/proto"
	"github.com/google/uuid"
)

var (
	// ErrIteratorDone may be thrown by list iterators to stop iteration.
	ErrIteratorDone = errors.New("accord: iterator done")
	// ErrInvalidHandle returned by the backend if the handle cannot be used or has expired.
	ErrInvalidHandle = errors.New("accord: invalid handle")
)

// Iterator function. Return ErrIteratorDone to cancel gracefully.
type Iterator func(*HandleData) error

// Backend represents a storage/persistent backend for handle information.
type Backend interface {
	// Acquire acquires a new named resource handle within namespace until exp time.
	Acquire(ctx context.Context, owner, namespace, name string, exp time.Time, metadata map[string]string) (*HandleData, error)

	// Renew renews a handle with a specific exp time and returns the updated handle.
	Renew(ctx context.Context, owner string, handleID uuid.UUID, exp time.Time, metadata map[string]string) error

	// Done marks the resource as done.
	Done(ctx context.Context, owner string, handleID uuid.UUID, metadata map[string]string) error

	// Get retrieves handle data by ID.
	Get(ctx context.Context, handleID uuid.UUID) (*HandleData, error)

	// List iterates over done resources within a namespace
	List(ctx context.Context, filter *proto.ListRequest_Filter, iter Iterator) error

	// Close closes the backend connection.
	Close() error
}

// --------------------------------------------------------------------

// HandleData is retrieved by the backend.
type HandleData struct {
	ID          uuid.UUID         // a unique identifier
	Namespace   string            // the namespace of the handle
	Name        string            // the name of the handle
	Owner       string            // last/current owner identifier
	ExpTime     time.Time         // expiration time
	DoneTime    time.Time         // done time
	NumAcquired int               // number of times acquired
	Metadata    map[string]string // custom metadata
}

// IsDone indicates when a resource is marked as done.
func (h *HandleData) IsDone() bool {
	return h.DoneTime.After(zeroTime)
}

// UpdateMetadata merged metadata.
func (h *HandleData) UpdateMetadata(meta map[string]string) {
	if h.Metadata == nil {
		h.Metadata = make(map[string]string, len(meta))
	}

	for key, value := range meta {
		h.Metadata[key] = value
	}
}

var zeroTime = time.Unix(0, 0)
