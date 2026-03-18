/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler_test

import (
	"fmt"

	"chainguard.dev/driftlessaf/reconcilers/githubreconciler"
)

// ExampleResource_String demonstrates the string representation of a GitHub
// pull request resource.
func ExampleResource_String() {
	res := &githubreconciler.Resource{
		Owner:  "chainguard-dev",
		Repo:   "enterprise-packages",
		Number: 42,
		Type:   githubreconciler.ResourceTypePullRequest,
	}
	fmt.Println(res.String())
	// Output: chainguard-dev/enterprise-packages#42
}

// ExampleResource_String_path demonstrates the string representation of a
// path resource.
func ExampleResource_String_path() {
	res := &githubreconciler.Resource{
		Owner: "chainguard-dev",
		Repo:  "enterprise-packages",
		Ref:   "main",
		Path:  "packages/glibc",
		Type:  githubreconciler.ResourceTypePath,
	}
	fmt.Println(res.String())
	// Output: chainguard-dev/enterprise-packages@main:packages/glibc
}
