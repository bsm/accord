package accord

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/bsm/accord/internal/cache"
	"github.com/bsm/accord/rpc"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// ClientOptions contains options for the client
type ClientOptions struct {
	Owner     string            // owner, default: random UUID
	Namespace string            // namespace, default: ""
	Metadata  map[string]string // default metadata
	TTL       time.Duration     // TTL, default: 10 minutes
	Dir       string            // Temporary directory, defaults to os.TempDir()
	OnError   func(error)       // custom error handler for background tasks
}

func (o *ClientOptions) ttlSeconds() uint32 {
	return uint32(o.TTL / time.Second)
}

func (o *ClientOptions) handleError(err error) {
	if o.OnError != nil {
		o.OnError(err)
	}
}

func (o *ClientOptions) norm() *ClientOptions {
	var p ClientOptions
	if o != nil {
		p = *o
	}

	if p.Owner == "" {
		p.Owner = uuid.New().String()
	}
	if p.TTL < time.Second {
		p.TTL = 10 * time.Minute
	}
	return &p
}

func (o *ClientOptions) mergeMeta(meta map[string]string) map[string]string {
	if meta == nil && len(o.Metadata) != 0 {
		meta = make(map[string]string, len(o.Metadata))
	}
	for k, v := range o.Metadata {
		if _, ok := meta[k]; !ok {
			meta[k] = v
		}
	}

	return meta
}

// --------------------------------------------------------------------

// Client is a convenience client to the accord API.
type Client struct {
	rpc   rpc.V1Client
	opt   *ClientOptions
	cache cache.Cache
	ownCC *grpc.ClientConn
}

// RPCClient inits a new client.
func RPCClient(ctx context.Context, rpc rpc.V1Client, opt *ClientOptions) (*Client, error) {
	opt = opt.norm()

	cacheDir, err := os.MkdirTemp(opt.Dir, "accord-client-cache")
	if err != nil {
		return nil, err
	}

	cache, err := cache.OpenBadger(cacheDir)
	if err != nil {
		return nil, err
	}

	client := &Client{
		rpc:   rpc,
		opt:   opt,
		cache: cache,
	}
	if err := client.fetchDone(ctx); err != nil {
		_ = cache.Close()
		return nil, err
	}
	return client, nil
}

// WrapClient inits a new client by wrapping a gRCP client connection.
func WrapClient(ctx context.Context, cc *grpc.ClientConn, opt *ClientOptions) (*Client, error) {
	return RPCClient(ctx, rpc.NewV1Client(cc), opt)
}

// DialClient creates a new client connection.
func DialClient(ctx context.Context, target string, opt *ClientOptions, dialOpt ...grpc.DialOption) (*Client, error) {
	cc, err := grpc.DialContext(ctx, target, dialOpt...)
	if err != nil {
		return nil, err
	}
	ci, err := WrapClient(ctx, cc, opt)
	if err != nil {
		_ = cc.Close()
		return nil, err
	}

	ci.ownCC = cc
	return ci, nil
}

// Acquire implements ClientConn interface.
func (c *Client) Acquire(ctx context.Context, name string, meta map[string]string) (*Handle, error) {
	// check in cache first
	if found, err := c.cache.Contains(name); err != nil {
		return nil, err
	} else if found {
		return nil, ErrDone
	}

	// try to acquire
	res, err := c.rpc.Acquire(ctx, &rpc.AcquireRequest{
		Owner:     c.opt.Owner,
		Name:      name,
		Namespace: c.opt.Namespace,
		Ttl:       c.opt.ttlSeconds(),
		Metadata:  c.opt.mergeMeta(meta),
	})
	if err != nil {
		return nil, err
	}

	switch res.Status {
	case rpc.Status_HELD:
		return nil, ErrAcquired
	case rpc.Status_DONE:
		if err := c.cache.Add(name); err != nil {
			return nil, err
		}
		return nil, ErrDone
	}

	handleID := uuid.Must(uuid.FromBytes(res.Handle.Id))
	return newHandle(handleID, c.rpc, res.Handle.Metadata, c.opt), nil
}

// RPC implements Client interface.
func (c *Client) RPC() rpc.V1Client {
	return c.rpc
}

// Close implements Client interface.
func (c *Client) Close() error {
	var err error
	if c.cache != nil {
		if e2 := c.cache.Close(); e2 != nil {
			err = e2
		}
	}
	if c.ownCC != nil {
		if e2 := c.ownCC.Close(); e2 != nil {
			err = e2
		}
	}
	return err
}

func (c *Client) fetchDone(ctx context.Context) error {
	res, err := c.rpc.List(ctx, &rpc.ListRequest{
		Filter: &rpc.ListRequest_Filter{
			Prefix: c.opt.Namespace,
			Status: rpc.ListRequest_Filter_DONE,
		},
	})
	if err != nil {
		return err
	}

	wb, err := c.cache.AddBatch()
	if err != nil {
		return err
	}
	defer wb.Discard()

	for {
		handle, err := res.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if handle.Namespace == c.opt.Namespace {
			if err := wb.Add(handle.Name); err != nil {
				return err
			}
		}
	}
	return wb.Flush()
}
