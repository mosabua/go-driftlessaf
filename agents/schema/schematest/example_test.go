/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package schematest_test

import (
	"encoding/json"
	"testing"

	"chainguard.dev/driftlessaf/agents/schema"
	"chainguard.dev/driftlessaf/agents/schema/schematest"
)

// ExampleCompareReflected demonstrates comparing a generated schema against an
// expected JSON schema string.
func ExampleCompareReflected() {
	type Input struct {
		Path string `json:"path" jsonschema:"required"`
	}

	t := &testing.T{}
	actual, _ := json.Marshal(schema.ReflectType[Input]())
	schematest.CompareReflected(t, actual, `{"type":"object"}`)
}

// ExampleCompareSubset demonstrates checking that an expected schema is a
// subset of the actual schema.
func ExampleCompareSubset() {
	type Input struct {
		Path string `json:"path" jsonschema:"required"`
	}

	t := &testing.T{}
	actual, _ := json.Marshal(schema.ReflectType[Input]())
	schematest.CompareSubset(t, `{"type":"object"}`, actual)
}
