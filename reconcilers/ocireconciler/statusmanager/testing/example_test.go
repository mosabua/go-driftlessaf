/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package testing_test

import (
	"fmt"
)

// Example_new demonstrates the intended usage of New.
//
// New requires chainctl to be installed and authenticated. It is intended for
// integration tests that run with the withauth build tag.
func Example_new() {
	// In an integration test, call:
	//
	//   mgr, err := smtesting.New[MyDetails](ctx, t, "my-reconciler")
	//   if err != nil {
	//       t.Fatal(err)
	//   }
	//
	// The manager is authenticated via chainctl and ready to write attestations.
	fmt.Println("New requires chainctl authentication")
	// Output: New requires chainctl authentication
}
