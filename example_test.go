package accord_test

import (
	"context"
	"fmt"

	"github.com/bsm/accord"
)

func ExampleClient() {
	ctx := context.Background()

	// Create a new client
	client, err := accord.DialClient(ctx, "10.0.0.1:8432", &accord.ClientOptions{
		Namespace: "/custom/namespace",
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Acquire resource handle.
	handle, err := client.Acquire(ctx, "my::resource", nil)
	if err == accord.ErrDone {
		fmt.Println("Resource has been already marked as done")
		return
	} else if err == accord.ErrAcquired {
		fmt.Println("Resource is currently held by another process")
		return
	} else if err != nil {
		panic(err)
	}
	defer handle.Discard()

	// Yay, we have acquired a handle on the resource, now let's do something!
	// ...

	// When done, we can mark the resource as done.
	if err := handle.Done(ctx, nil); err != nil {
		panic(err)
	}
}
