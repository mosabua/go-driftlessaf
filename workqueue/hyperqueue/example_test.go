/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package hyperqueue_test

import (
	"fmt"

	"chainguard.dev/driftlessaf/workqueue/hyperqueue"
)

// ExampleNew demonstrates creating a hyperqueue server from a set of backend
// WorkqueueServiceClient instances.
func ExampleNew() {
	// In production, pass real gRPC clients connected to backend workqueue
	// services. New returns an error if no backends are provided.
	_, err := hyperqueue.New(nil)
	fmt.Println("error with no backends:", err)
	// Output: error with no backends: at least one backend is required
}
