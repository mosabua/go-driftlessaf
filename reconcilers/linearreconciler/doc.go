/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package linearreconciler provides a workqueue-based reconciliation framework
// for Linear issues.
//
// The reconciler processes issue keys from a workqueue, fetches the
// corresponding Linear issue via the GraphQL API, and invokes a user-supplied
// ReconcilerFunc for each one. It handles rate limiting, label gating, and
// team filtering, and supports persisting reconciler state as issue
// attachments.
package linearreconciler
