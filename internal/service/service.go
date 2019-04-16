package service

import (
	"context"
	"time"

	"github.com/bsm/accord"
	"github.com/bsm/accord/backend"
	"github.com/bsm/accord/internal/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type service struct {
	b backend.Backend
}

// New initalizes a new service
func New(b backend.Backend) proto.V1Server {
	return &service{b: b}
}

// Acquire implements proto.V1Server.
func (s *service) Acquire(ctx context.Context, req *proto.AcquireRequest) (*proto.AcquireResponse, error) {
	if req.Owner == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid owner")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid name")
	}

	data, err := s.b.Acquire(ctx, req.Owner, req.Namespace, req.Name, expTime(req.Ttl), req.Metadata)
	if err == accord.ErrDone {
		return &proto.AcquireResponse{Status: proto.Status_DONE}, nil
	} else if err == accord.ErrAcquired {
		return &proto.AcquireResponse{Status: proto.Status_HELD}, nil
	} else if err != nil {
		return nil, err
	}

	return &proto.AcquireResponse{
		Status: proto.Status_OK,
		Handle: convertHandle(data),
	}, nil
}

// Renew implements proto.V1Server.
func (s *service) Renew(ctx context.Context, req *proto.RenewRequest) (*proto.RenewResponse, error) {
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
	return &proto.RenewResponse{}, nil
}

// Done implements proto.V1Server.
func (s *service) Done(ctx context.Context, req *proto.DoneRequest) (*proto.DoneResponse, error) {
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
	return &proto.DoneResponse{}, nil
}

// List implements proto.V1Server.
func (s *service) List(req *proto.ListRequest, srv proto.V1_ListServer) error {
	return s.b.List(srv.Context(), req.Filter, func(data *backend.HandleData) error {
		return srv.Send(convertHandle(data))
	})
}

func convertHandle(data *backend.HandleData) *proto.Handle {
	expMillis := data.ExpTime.Unix()*1000 + int64(data.ExpTime.Nanosecond())/1e6
	return &proto.Handle{
		Id:          data.ID[:],
		Name:        data.Name,
		Namespace:   data.Namespace,
		ExpTime:     expMillis,
		NumAcquired: uint32(data.NumAcquired),
		Metadata:    data.Metadata,
	}
}

func expTime(ttl uint32) time.Time {
	return time.Now().Add(time.Second * time.Duration(ttl))
}
