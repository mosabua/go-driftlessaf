/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package metapathreconciler_test

import (
	"fmt"

	"chainguard.dev/driftlessaf/reconcilers/githubreconciler/metapathreconciler"
)

// Example_prData demonstrates the PRData type used for change detection.
// The PRData is embedded in PR bodies to track state across reconciliations.
func Example_prData() {
	data := metapathreconciler.PRData{
		Identity: "my-bot",
	}

	fmt.Printf("Bot: %s\n", data.Identity)

	// Output:
	// Bot: my-bot
}

// Example_diagnostic demonstrates how Diagnostics are converted to Findings.
func Example_diagnostic() {
	diag := metapathreconciler.Diagnostic{
		Path:    "pkg/handler.go",
		Line:    42,
		Message: "use slices.Contains instead of manual loop",
		Rule:    "modernize",
	}

	finding := diag.AsFinding()
	fmt.Printf("Kind: %s\n", finding.Kind)
	fmt.Printf("ID: %s\n", finding.Identifier)

	// Output:
	// Kind: ciCheck
	// ID: modernize:pkg/handler.go:42
}
