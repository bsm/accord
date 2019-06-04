package service

import (
	"context"
	"time"

	"github.com/bsm/accord"
	"github.com/bsm/accord/backend"
	"github.com/bsm/accord/rpc"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service instances serve GRPC requests.
type Service struct {
	b backend.Backend
}

// New initalizes a new service
func New(b backend.Backend) *Service {
	return &Service{b: b}
}

// Ping implements rpc.Pinger.
func (s *Service) Ping() error {
	return s.b.Ping()
}

// Acquire implements rpc.V1Server.
func (s *Service) Acquire(ctx context.Context, req *rpc.AcquireRequest) (*rpc.AcquireResponse, error) {
	if req.Owner == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid owner")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid name")
	}

	data, err := s.b.Acquire(ctx, req.Owner, req.Namespace, req.Name, expTime(req.Ttl), req.Metadata)
	if err == accord.ErrDone {
		return &rpc.AcquireResponse{Status: rpc.Status_DONE}, nil
	} else if err == accord.ErrAcquired {
		return &rpc.AcquireResponse{Status: rpc.Status_HELD}, nil
	} else if err != nil {
		return nil, err
	}

	return &rpc.AcquireResponse{
		Status: rpc.Status_OK,
		Handle: convertHandle(data),
	}, nil
}

// Renew implements rpc.V1Server.
func (s *Service) Renew(ctx context.Context, req *rpc.RenewRequest) (*rpc.RenewResponse, error) {
	if req.Owner == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid owner")
	}

	handleID, err := uuid.FromBytes(req.HandleId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid handle ID")
	}

	if err := s.b.Renew(ctx, req.Owner, handleID, expTime(req.Ttl), req.Metadata); err != nil {
		return nil, err
	}
	return &rpc.RenewResponse{}, nil
}

// Done implements rpc.V1Server.
func (s *Service) Done(ctx context.Context, req *rpc.DoneRequest) (*rpc.DoneResponse, error) {
	if req.Owner == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid owner")
	}

	handleID, err := uuid.FromBytes(req.HandleId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid handle ID")
	}

	if err := s.b.Done(ctx, req.Owner, handleID, req.Metadata); err != nil {
		return nil, err
	}
	return &rpc.DoneResponse{}, nil
}

// List implements rpc.V1Server.
func (s *Service) List(req *rpc.ListRequest, srv rpc.V1_ListServer) error {
	return s.b.List(srv.Context(), req, func(data *backend.HandleData) error {
		return srv.Send(convertHandle(data))
	})
}

func convertHandle(data *backend.HandleData) *rpc.Handle {
	return &rpc.Handle{
		Id:          data.ID[:],
		Name:        data.Name,
		Namespace:   data.Namespace,
		ExpTms:      timeToMillis(data.ExpTime),
		DoneTms:     timeToMillis(data.DoneTime),
		NumAcquired: uint32(data.NumAcquired),
		Metadata:    data.Metadata,
	}
}

func expTime(ttl uint32) time.Time {
	return time.Now().Add(time.Second * time.Duration(ttl))
}

func timeToMillis(t time.Time) int64 {
	if u := t.Unix(); u > 0 {
		return u*1e3 + int64(t.Nanosecond())/1e6
	}
	return 0
}
