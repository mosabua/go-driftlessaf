/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package apkreconciler_test

import (
	"context"
	"fmt"

	"chainguard.dev/driftlessaf/reconcilers/apkreconciler"
	"chainguard.dev/driftlessaf/reconcilers/apkreconciler/apkurl"
)

// ExampleNew demonstrates creating an APK reconciler with a reconcile function.
func ExampleNew() {
	r := apkreconciler.New(
		apkreconciler.WithReconciler(func(_ context.Context, key *apkurl.Key) error {
			fmt.Printf("Reconciling: %s\n", key.Package.Name)
			return nil
		}),
	)
	fmt.Printf("reconciler type: %T\n", r)
	// Output: reconciler type: *apkreconciler.Reconciler
}
