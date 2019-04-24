package accord

import (
	"context"
	"sync"
	"time"

	"github.com/bsm/accord/internal/proto"
	"github.com/google/uuid"
)

// Handle holds temporary ownership of a resource. It will automatically renew
// its ownership in the background until either Done or Discard is called (first one wins).
// After a call to Done or Discard, all operations on the handle fail with ErrClosed.
type Handle struct {
	id   uuid.UUID
	rpc  proto.V1Client
	meta *metadata
	opt  *ClientOptions
	mu   sync.Mutex

	ctx   context.Context
	close context.CancelFunc
}

func newHandle(id uuid.UUID, rpc proto.V1Client, meta map[string]string, opt *ClientOptions) *Handle {
	ctx, close := context.WithCancel(context.Background())
	h := &Handle{
		id:    id,
		rpc:   rpc,
		meta:  &metadata{kv: meta},
		opt:   opt,
		ctx:   ctx,
		close: close,
	}
	go h.renewLoop()
	return h
}

// ID returns the handle ID.
func (h *Handle) ID() uuid.UUID {
	return h.id
}

// Metadata returns metadata.
func (h *Handle) Metadata() map[string]string {
	return h.meta.Snap()
}

// SetMeta sets a metadata key.
// It will only be persisted with the next (background) Renew or Done call.
func (h *Handle) SetMeta(key, value string) {
	h.meta.Set(key, value)
}

// Renew manually renews the ownership of the resource with custom metadata.
func (h *Handle) Renew(ctx context.Context, meta map[string]string) error {
	if h.isClosed() {
		return ErrClosed
	}

	h.meta.Update(meta)
	return h.renew(ctx, h.opt.ttlSeconds())
}

// Done marks the resource as done and invalidates the handle.
func (h *Handle) Done(ctx context.Context, meta map[string]string) error {
	if h.isClosed() {
		return ErrClosed
	}

	h.meta.Update(meta)

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := h.rpc.Done(ctx, &proto.DoneRequest{
		Owner:    h.opt.Owner,
		HandleId: h.id[:],
		Metadata: h.meta.Snap(),
	})
	if err == nil {
		h.close()
	}
	return err
}

// Discard discards the handle.
func (h *Handle) Discard() error {
	if h.isClosed() {
		return ErrClosed
	}

	defer h.close()
	return h.renew(h.ctx, 0)
}

func (h *Handle) isClosed() bool {
	select {
	case <-h.ctx.Done():
		return true
	default:
		return false
	}
}

func (h *Handle) renew(ctx context.Context, seconds uint32) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := h.rpc.Renew(ctx, &proto.RenewRequest{
		Owner:    h.opt.Owner,
		HandleId: h.id[:],
		Ttl:      seconds,
		Metadata: h.meta.Snap(),
	})
	return err
}

func (h *Handle) renewLoop() {
	ticker := time.NewTicker(h.opt.TTL * 3 / 10)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			if err := h.Renew(h.ctx, nil); err != nil && err != context.Canceled && err != ErrClosed {
				h.opt.handleError(err)
			}
		}
	}
}
