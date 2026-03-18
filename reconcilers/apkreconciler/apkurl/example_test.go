/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package apkurl_test

import (
	"fmt"

	"chainguard.dev/driftlessaf/reconcilers/apkreconciler/apkurl"
)

// ExampleParse demonstrates parsing an APK URL key into its components.
func ExampleParse() {
	key, err := apkurl.Parse("packages.wolfi.dev/os/x86_64/glibc-2.42-r0.apk")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Host: %s\n", key.Host)
	fmt.Printf("Package: %s\n", key.Package.Name)
	fmt.Printf("Version: %s\n", key.Package.Version)
	fmt.Printf("Arch: %s\n", key.Package.Arch)
	// Output:
	// Host: packages.wolfi.dev
	// Package: glibc
	// Version: 2.42-r0
	// Arch: x86_64
}
