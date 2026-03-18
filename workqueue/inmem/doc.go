/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package inmem provides an in-memory workqueue implementation intended for
// testing.
//
// The in-memory workqueue is not suitable for production use — it does not
// persist state across restarts and does not support distributed deployments.
// Use it in tests alongside the conformance package to verify workqueue
// consumer behavior.
package inmem
