package rpc

import (
	context "context"
	"time"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/health"
	hpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Pinger servers can handle ping requests.
type Pinger interface {
	Ping() error
}

// HealthCheck instances can be stopped.
type HealthCheck interface {
	Stop()
}

type healthCheck struct{ cancel context.CancelFunc }

func (h *healthCheck) Stop() { h.cancel() }

// RunHealthCheck starts a standard grpc health check.
func RunHealthCheck(s *grpc.Server, c Pinger, name string, interval time.Duration) HealthCheck {
	svc := health.NewServer()
	hpb.RegisterHealthServer(s, svc)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				svc.SetServingStatus(name, hpb.HealthCheckResponse_NOT_SERVING)
				return
			case <-ticker.C:
				if err := c.Ping(); err == nil {
					svc.SetServingStatus(name, hpb.HealthCheckResponse_SERVING)
				} else {
					svc.SetServingStatus(name, hpb.HealthCheckResponse_NOT_SERVING)
				}
			}
		}
	}()
	return &healthCheck{cancel: cancel}
}
