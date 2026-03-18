/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package githubreconciler provides a workqueue-based reconciliation framework
// for GitHub pull requests.
//
// The reconciler processes pull request keys from a workqueue, parses them into
// structured PR references, and invokes a user-supplied ReconcilerFunc for each
// one. It integrates with GitHub's API for cloning, committing, and status
// reporting via check runs.
package githubreconciler
