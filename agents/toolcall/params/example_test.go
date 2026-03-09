/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package params_test

import (
	"errors"
	"fmt"

	"chainguard.dev/driftlessaf/agents/toolcall/params"
)

func ExampleExtract() {
	args := map[string]any{
		"name":  "hello",
		"count": float64(42),
	}

	name, err := params.Extract[string](args, "name")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(name)

	count, err := params.Extract[int](args, "count")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(count)

	// Output:
	// hello
	// 42
}

func ExampleExtractOptional() {
	args := map[string]any{
		"name": "hello",
	}

	name, err := params.ExtractOptional(args, "name", "default")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(name)

	missing, err := params.ExtractOptional(args, "missing", "default")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(missing)

	// Output:
	// hello
	// default
}

func ExampleError() {
	result := params.Error("invalid parameter %q", "foo")
	fmt.Println(result["error"])

	// Output:
	// invalid parameter "foo"
}

func ExampleErrorWithContext() {
	result := params.ErrorWithContext(errors.New("file not found"), map[string]any{
		"path": "/tmp/missing.txt",
	})
	fmt.Println(result["error"])
	fmt.Println(result["path"])

	// Output:
	// file not found
	// /tmp/missing.txt
}
