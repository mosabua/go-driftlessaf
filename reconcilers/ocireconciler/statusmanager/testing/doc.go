/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package testing provides test helpers for the ocireconciler statusmanager.
//
// It creates writable and read-only statusmanager instances authenticated via
// chainctl, suitable for integration tests that require real Sigstore keyless
// signing.
package testing
