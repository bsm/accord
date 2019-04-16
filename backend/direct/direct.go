// Package direct implements a backend wrapper which allows to connect clients to a
// backend directly bypassing grpc servers. This is intended either for testing,
// use at own risk!
package direct

import (
	"context"
	"io"
	"sync/atomic"

	"github.com/bsm/accord/backend"
	"github.com/bsm/accord/internal/proto"
	"github.com/bsm/accord/internal/service"
	"google.golang.org/grpc"
)

// Connect allows to connect clients directly to backend, bypassing the
// servers. This is not recommended!
func Connect(b backend.Backend) proto.V1Client {
	return &bypass{b: b, s: service.New(b)}
}

type bypass struct {
	b backend.Backend
	s proto.V1Server
}

func (b *bypass) Acquire(ctx context.Context, in *proto.AcquireRequest, _ ...grpc.CallOption) (*proto.AcquireResponse, error) {
	return b.s.Acquire(ctx, in)
}

func (b *bypass) Renew(ctx context.Context, in *proto.RenewRequest, _ ...grpc.CallOption) (*proto.RenewResponse, error) {
	return b.s.Renew(ctx, in)
}

func (b *bypass) Done(ctx context.Context, in *proto.DoneRequest, _ ...grpc.CallOption) (*proto.DoneResponse, error) {
	return b.s.Done(ctx, in)
}

func (b *bypass) List(ctx context.Context, in *proto.ListRequest, _ ...grpc.CallOption) (proto.V1_ListClient, error) {
	ch := make(chan *proto.Handle, 10)
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
	ch  chan *proto.Handle

	erv atomic.Value
}

func (s *listClient) Context() context.Context { return s.ctx }
func (s *listClient) Recv() (*proto.Handle, error) {
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
	ch  chan *proto.Handle
}

func (s *listServer) Context() context.Context { return s.ctx }
func (s *listServer) Send(h *proto.Handle) error {
	select {
	case <-s.ctx.Done():
	case s.ch <- h:
	}
	return s.ctx.Err()
}
