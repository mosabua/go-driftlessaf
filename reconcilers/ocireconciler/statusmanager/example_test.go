/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package statusmanager_test

import (
	"fmt"

	"chainguard.dev/driftlessaf/reconcilers/ocireconciler/statusmanager"
)

// ExampleStatus demonstrates constructing a Status value with custom details.
func ExampleStatus() {
	type MyDetails struct {
		Result string
	}

	s := statusmanager.Status[MyDetails]{
		ObservedGeneration: "sha256:abc123",
		Details:            MyDetails{Result: "success"},
	}
	fmt.Printf("generation=%s result=%s\n", s.ObservedGeneration, s.Details.Result)
	// Output: generation=sha256:abc123 result=success
}
