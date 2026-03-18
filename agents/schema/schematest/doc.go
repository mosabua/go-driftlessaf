/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package schematest provides test helpers for comparing JSON schemas.
//
// Use CompareReflected and CompareSubset to assert that generated schemas
// match expected structures, with support for subset matching to allow
// additional descriptive fields in the actual schema.
package schematest
