package main

import (
	"context"
	"flag"
	"log"
	"net"
	"strings"
	"time"

	"github.com/bsm/accord/backend/postgres"
	"github.com/bsm/accord/internal/service"
	"github.com/bsm/accord/rpc"
	"google.golang.org/grpc"
)

var flags struct {
	addr    string
	backend string
}

func init() {
	flag.StringVar(&flags.addr, "addr", ":7475", "Address for the server to listen on")
	flag.StringVar(&flags.backend, "backend", "postgres://127.0.0.1:5432/accord", "Backend URL")
}

func main() {
	flag.Parse()

	if err := run(context.Background()); err != nil {
		log.Fatalln(err)
	}
}

func run(ctx context.Context) error {
	driver := strings.SplitN(flags.backend, ":", 2)[0]
	backend, err := postgres.Open(ctx, driver, flags.backend)
	if err != nil {
		return err
	}
	log.Printf("Connected to %q backend\n", driver)
	defer backend.Close()

	lis, err := net.Listen("tcp", flags.addr)
	if err != nil {
		return err
	}

	srv := grpc.NewServer()
	svc := service.New(backend)
	rpc.RegisterV1Server(srv, svc)
	hch := rpc.RunHealthCheck(srv, svc, "accord", 5*time.Second)
	defer hch.Stop()

	log.Printf("Listening on %s\n", flags.addr)
	return srv.Serve(lis)
}
