/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package schema_test

import (
	"encoding/json"
	"fmt"

	"chainguard.dev/driftlessaf/agents/schema"
)

// ExampleReflectType demonstrates generating a JSON schema from a Go type.
func ExampleReflectType() {
	type Input struct {
		Path   string `json:"path" jsonschema:"required,description=File path to read"`
		Offset int    `json:"offset,omitempty" jsonschema:"description=Byte offset"`
	}

	s := schema.ReflectType[Input]()
	data, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		panic(err)
	}
	fmt.Println("type:", m["type"])
	// Output: type: object
}

// ExampleNewGenerator demonstrates creating a generator and reflecting a value.
func ExampleNewGenerator() {
	type Params struct {
		Query string `json:"query" jsonschema:"required"`
	}

	g := schema.NewGenerator()
	s := g.Reflect(&Params{})
	data, _ := json.Marshal(s)

	var m map[string]any
	_ = json.Unmarshal(data, &m)
	fmt.Println("type:", m["type"])
	// Output: type: object
}
