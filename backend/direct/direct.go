// Package direct implements a backend wrapper which allows to connect clients to a
// backend directly bypassing grpc servers. This is intended either for testing,
// use at own risk!
package direct

import (
	"context"
	"io"
	"sync/atomic"

	"github.com/bsm/accord/backend"
	"github.com/bsm/accord/internal/service"
	"github.com/bsm/accord/rpc"
	"google.golang.org/grpc"
)

// Connect allows to connect clients directly to backend, bypassing the
// servers. This is not recommended!
func Connect(b backend.Backend) rpc.V1Client {
	return &bypass{b: b, s: service.New(b)}
}

type bypass struct {
	b backend.Backend
	s rpc.V1Server
}

func (b *bypass) Acquire(ctx context.Context, in *rpc.AcquireRequest, _ ...grpc.CallOption) (*rpc.AcquireResponse, error) {
	return b.s.Acquire(ctx, in)
}

func (b *bypass) Renew(ctx context.Context, in *rpc.RenewRequest, _ ...grpc.CallOption) (*rpc.RenewResponse, error) {
	return b.s.Renew(ctx, in)
}

func (b *bypass) Done(ctx context.Context, in *rpc.DoneRequest, _ ...grpc.CallOption) (*rpc.DoneResponse, error) {
	return b.s.Done(ctx, in)
}

func (b *bypass) List(ctx context.Context, in *rpc.ListRequest, _ ...grpc.CallOption) (rpc.V1_ListClient, error) {
	ch := make(chan *rpc.Handle, 10)
	lc := &listClient{ctx: ctx, ch: ch}
	ls := &listServer{ctx: ctx, ch: ch}

	go func() {
		if err := b.s.List(in, ls); err != nil {
			lc.erv.Store(err)
		}

		close(ch)
	}()

	return lc, nil
}

// --------------------------------------------------------------------

type listClient struct {
	grpc.ClientStream

	ctx context.Context
	ch  chan *rpc.Handle

	erv atomic.Value
}

func (s *listClient) Context() context.Context { return s.ctx }
func (s *listClient) Recv() (*rpc.Handle, error) {
	if v := s.erv.Load(); v != nil {
		return nil, v.(error)
	}

	select {
	case h, more := <-s.ch:
		if !more {
			return nil, io.EOF
		}
		return h, nil
	case <-s.ctx.Done():
	}
	return nil, s.ctx.Err()
}

type listServer struct {
	grpc.ServerStream

	ctx context.Context
	ch  chan *rpc.Handle
}

func (s *listServer) Context() context.Context { return s.ctx }
func (s *listServer) Send(h *rpc.Handle) error {
	select {
	case <-s.ctx.Done():
	case s.ch <- h:
	}
	return s.ctx.Err()
}
