package direct_test

import (
	"context"

	"github.com/bsm/accord"
	"github.com/bsm/accord/backend/direct"
	"github.com/bsm/accord/backend/postgres"
)

func Example() {
	ctx := context.Background()

	// Open a backend connection.
	backend, err := postgres.Open(ctx, "postgres", "postgres://127.0.0.1:5432/accord")
	if err != nil {
		panic(err)
	}
	defer backend.Close()

	// Bypass gRPC servers and connect a client directly to a backend (not recommended).
	client, err := accord.RPCClient(ctx, direct.Connect(backend), nil)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// ... use client
}
