/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package worktreefs_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"chainguard.dev/driftlessaf/reconcilers/githubreconciler/clonemanager"
	"chainguard.dev/driftlessaf/reconcilers/githubreconciler/clonemanager/worktreefs"
	gogit "github.com/go-git/go-git/v5"
)

func Example() {
	// Create a temporary repo for demonstration.
	dir, err := os.MkdirTemp("", "worktreefs-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	repo, err := gogit.PlainInit(dir, false)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello"), 0o644); err != nil {
		panic(err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		panic(err)
	}

	// Create an fs.FS from worktree callbacks.
	cb := clonemanager.WorktreeCallbacks(wt)
	fsys := worktreefs.New(context.Background(), cb)

	// ReadFile works as expected.
	data, err := fs.ReadFile(fsys, "README.md")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))

	// Output:
	// # Hello
}
