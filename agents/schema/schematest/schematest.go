/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package schematest provides test helpers for comparing JSON schemas.
package schematest

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// CompareReflected compares a generated schema (as JSON bytes) against an expected
// JSON schema string. It performs a subset match — all expected properties must exist
// in the actual schema, but actual may have additional fields.
func CompareReflected(t *testing.T, actualJSON []byte, expectedSchema string) {
	t.Helper()

	var expected map[string]any
	if err := json.Unmarshal([]byte(expectedSchema), &expected); err != nil {
		t.Fatalf("failed to parse expected schema: %v", err)
	}

	var actual map[string]any
	if err := json.Unmarshal(actualJSON, &actual); err != nil {
		t.Fatalf("failed to parse actual schema: %v", err)
	}

	if err := compareSchemas(expected, actual, ""); err != nil {
		t.Errorf("Schema mismatch:\n%s\n\nActual schema:\n%s", err, string(actualJSON))
	}
}

// CompareSubset checks that expected is a subset of actual. Both are parsed from
// JSON strings. This is useful for comparing tool schemas where the actual schema
// may contain additional descriptive fields.
func CompareSubset(t *testing.T, expectedJSON string, actualJSON []byte) {
	t.Helper()

	var expected map[string]any
	if err := json.Unmarshal([]byte(expectedJSON), &expected); err != nil {
		t.Fatalf("failed to parse expected schema: %v", err)
	}

	var actual map[string]any
	if err := json.Unmarshal(actualJSON, &actual); err != nil {
		t.Fatalf("failed to unmarshal actual schema: %v", err)
	}

	if err := checkSubset("", expected, actual); err != nil {
		actualPretty, _ := json.MarshalIndent(actual, "", "  ")
		t.Errorf("schema mismatch: %v\n\nFull actual schema:\n%s", err, actualPretty)
	}
}

// compareSchemas recursively compares expected (subset) with actual schema,
// using case-insensitive string comparison for values.
func compareSchemas(expected, actual map[string]any, path string) error {
	for key, expectedVal := range expected {
		currentPath := path + "." + key
		if currentPath == "."+key {
			currentPath = key
		}

		actualVal, ok := actual[key]
		if !ok {
			return fmt.Errorf("missing key %q in actual schema", currentPath)
		}

		switch expV := expectedVal.(type) {
		case map[string]any:
			actV, ok := actualVal.(map[string]any)
			if !ok {
				return fmt.Errorf("at %q: expected object, got %T", currentPath, actualVal)
			}
			if err := compareSchemas(expV, actV, currentPath); err != nil {
				return err
			}

		case []any:
			actV, ok := actualVal.([]any)
			if !ok {
				return fmt.Errorf("at %q: expected array, got %T", currentPath, actualVal)
			}
			if len(expV) > 0 && len(actV) > 0 {
				if expObj, ok := expV[0].(map[string]any); ok {
					if actObj, ok := actV[0].(map[string]any); ok {
						if err := compareSchemas(expObj, actObj, currentPath+"[0]"); err != nil {
							return err
						}
					}
				}
			}

		case string:
			actV, ok := actualVal.(string)
			if !ok {
				return fmt.Errorf("at %q: expected string %q, got %T=%v", currentPath, expV, actualVal, actualVal)
			}
			if !strings.EqualFold(expV, actV) {
				return fmt.Errorf("at %q: expected %q, got %q", currentPath, expV, actV)
			}

		default:
			if expectedVal != actualVal {
				return fmt.Errorf("at %q: expected %v, got %v", currentPath, expectedVal, actualVal)
			}
		}
	}
	return nil
}

// checkSubset verifies that expected is a subset of actual (recursively),
// using exact match for primitive values.
func checkSubset(path string, expected, actual any) error {
	switch e := expected.(type) {
	case map[string]any:
		a, ok := actual.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: expected object, got %T", path, actual)
		}
		for key, expectedVal := range e {
			actualVal, ok := a[key]
			if !ok {
				return fmt.Errorf("%s.%s: missing in actual schema", path, key)
			}
			newPath := path
			if newPath == "" {
				newPath = key
			} else {
				newPath = path + "." + key
			}
			if err := checkSubset(newPath, expectedVal, actualVal); err != nil {
				return err
			}
		}
		return nil

	case []any:
		a, ok := actual.([]any)
		if !ok {
			return fmt.Errorf("%s: expected array, got %T", path, actual)
		}
		if len(e) != len(a) {
			return fmt.Errorf("%s: expected array length %d, got %d", path, len(e), len(a))
		}
		for i, expectedVal := range e {
			if err := checkSubset(fmt.Sprintf("%s[%d]", path, i), expectedVal, a[i]); err != nil {
				return err
			}
		}
		return nil

	default:
		if expected != actual {
			return fmt.Errorf("%s: expected %v, got %v", path, expected, actual)
		}
		return nil
	}
}
