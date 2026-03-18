/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package gcs provides a Google Cloud Storage-backed workqueue implementation.
//
// Keys are stored as GCS objects under queued/, in-progress/, and dead-letter/
// prefixes. The implementation supports priority ordering, not-before delays,
// and automatic lease refresh to detect orphaned work.
package gcs
