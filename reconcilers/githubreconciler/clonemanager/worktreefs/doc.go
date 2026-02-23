/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package worktreefs implements [io/fs.FS] backed by
// [callbacks.WorktreeCallbacks]. It satisfies [fs.FS], [fs.StatFS],
// [fs.ReadFileFS], and [fs.ReadDirFS], making a worktree's contents available
// to any consumer that accepts a standard [fs.FS].
//
// All reported modification times are the Unix epoch (1970-01-01T00:00:00Z).
// Git does not track file modification times, so a fixed epoch avoids exposing
// non-deterministic local timestamps.
//
// A [context.Context] is captured at construction time and used for all
// callback invocations, since the [fs.FS] interface methods do not accept
// a context.
//
// # Usage
//
//	cb := clonemanager.WorktreeCallbacks(wt)
//	fsys := worktreefs.New(ctx, cb)
//
//	// Use with any fs.FS consumer
//	data, _ := fs.ReadFile(fsys, "path/to/file.txt")
//	entries, _ := fs.ReadDir(fsys, "some/directory")
package worktreefs
