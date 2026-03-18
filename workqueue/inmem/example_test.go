/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package inmem_test

import (
	"context"
	"fmt"

	"chainguard.dev/driftlessaf/workqueue"
	"chainguard.dev/driftlessaf/workqueue/inmem"
)

// ExampleNewWorkQueue demonstrates creating an in-memory workqueue and
// performing basic queue and enumerate operations.
func ExampleNewWorkQueue() {
	wq := inmem.NewWorkQueue(5)
	ctx := context.Background()

	if err := wq.Queue(ctx, "my-key", workqueue.Options{}); err != nil {
		panic(err)
	}

	_, queued, _, err := wq.Enumerate(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Queued keys: %d\n", len(queued))
	// Output: Queued keys: 1
}
