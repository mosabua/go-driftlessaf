/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package prvalidation_test

import (
	"fmt"

	"chainguard.dev/driftlessaf/examples/prvalidation"
)

func ExampleValidatePR() {
	titleValid, descValid, issues := prvalidation.ValidatePR(
		"feat: add new feature",
		"This PR adds a new feature to the system.",
	)
	fmt.Println("title valid:", titleValid)
	fmt.Println("desc valid:", descValid)
	fmt.Println("issues:", len(issues))
	// Output:
	// title valid: true
	// desc valid: true
	// issues: 0
}

func ExampleValidatePR_invalid() {
	titleValid, descValid, issues := prvalidation.ValidatePR(
		"bad title",
		"",
	)
	fmt.Println("title valid:", titleValid)
	fmt.Println("desc valid:", descValid)
	fmt.Println("issues:", len(issues))
	// Output:
	// title valid: false
	// desc valid: false
	// issues: 2
}

func ExampleComputeGeneration() {
	gen := prvalidation.ComputeGeneration("abc123", "feat: title", "body text")
	fmt.Println("length:", len(gen))
	// Output:
	// length: 64
}

func ExampleDetails_Markdown() {
	d := prvalidation.Details{
		TitleValid:       true,
		DescriptionValid: true,
	}
	md := d.Markdown()
	fmt.Println("contains report:", len(md) > 0)
	// Output:
	// contains report: true
}
