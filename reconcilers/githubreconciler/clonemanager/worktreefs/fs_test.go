/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package worktreefs_test

import (
	"context"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"chainguard.dev/driftlessaf/agents/toolcall/callbacks"
	"chainguard.dev/driftlessaf/reconcilers/githubreconciler/clonemanager"
	"chainguard.dev/driftlessaf/reconcilers/githubreconciler/clonemanager/worktreefs"
	gogit "github.com/go-git/go-git/v5"
)

// initCallbacks creates a temporary git repo with test fixtures and returns
// worktree callbacks for it.
func initCallbacks(t *testing.T) callbacks.WorktreeCallbacks {
	t.Helper()

	dir := t.TempDir()
	repo, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: got = %v, wanted = nil", err)
	}

	// Create test files and directories.
	writeFile(t, dir, "hello.txt", fmt.Sprintf("hello-%d", rand.Int64()))
	writeFile(t, dir, "sub/nested.txt", fmt.Sprintf("nested-%d", rand.Int64()))
	writeFile(t, dir, "sub/deep/file.txt", fmt.Sprintf("deep-%d", rand.Int64()))

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: got = %v, wanted = nil", err)
	}
	return clonemanager.WorktreeCallbacks(wt)
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): got = %v, wanted = nil", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): got = %v, wanted = nil", rel, err)
	}
}

func TestFS(t *testing.T) {
	cb := initCallbacks(t)
	fsys := worktreefs.New(context.Background(), cb)

	if err := fstest.TestFS(fsys, "hello.txt", "sub/nested.txt", "sub/deep/file.txt"); err != nil {
		t.Fatal(err)
	}
}

func TestModTime(t *testing.T) {
	cb := initCallbacks(t)
	fsys := worktreefs.New(context.Background(), cb)
	want := time.Unix(0, 0)

	t.Run("Stat", func(t *testing.T) {
		info, err := fsys.Stat("hello.txt")
		if err != nil {
			t.Fatalf("Stat: got = %v, wanted = nil", err)
		}
		if got := info.ModTime(); !got.Equal(want) {
			t.Errorf("ModTime: got = %v, wanted = %v", got, want)
		}
	})

	t.Run("Open+Stat", func(t *testing.T) {
		f, err := fsys.Open("hello.txt")
		if err != nil {
			t.Fatalf("Open: got = %v, wanted = nil", err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			t.Fatalf("File.Stat: got = %v, wanted = nil", err)
		}
		if got := info.ModTime(); !got.Equal(want) {
			t.Errorf("ModTime: got = %v, wanted = %v", got, want)
		}
	})

	t.Run("ReadDir+Info", func(t *testing.T) {
		entries, err := fsys.ReadDir("sub")
		if err != nil {
			t.Fatalf("ReadDir: got = %v, wanted = nil", err)
		}
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				t.Fatalf("Info(%q): got = %v, wanted = nil", e.Name(), err)
			}
			if got := info.ModTime(); !got.Equal(want) {
				t.Errorf("Info(%q).ModTime: got = %v, wanted = %v", e.Name(), got, want)
			}
		}
	})

	t.Run("Open+ReadDir+Info", func(t *testing.T) {
		f, err := fsys.Open("sub")
		if err != nil {
			t.Fatalf("Open: got = %v, wanted = nil", err)
		}
		defer f.Close()

		df, ok := f.(fs.ReadDirFile)
		if !ok {
			t.Fatal("directory file does not implement ReadDirFile")
		}
		entries, err := df.ReadDir(-1)
		if err != nil {
			t.Fatalf("ReadDir: got = %v, wanted = nil", err)
		}
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				t.Fatalf("Info(%q): got = %v, wanted = nil", e.Name(), err)
			}
			if got := info.ModTime(); !got.Equal(want) {
				t.Errorf("Info(%q).ModTime: got = %v, wanted = %v", e.Name(), got, want)
			}
		}
	})
}

func TestInvalidPaths(t *testing.T) {
	cb := initCallbacks(t)
	fsys := worktreefs.New(context.Background(), cb)

	for _, name := range []string{"", "..", "../escape", "/absolute", "a/../b"} {
		t.Run(name, func(t *testing.T) {
			if _, err := fsys.Open(name); err == nil {
				t.Errorf("Open(%q): got = nil, wanted = error", name)
			}
			if _, err := fsys.Stat(name); err == nil {
				t.Errorf("Stat(%q): got = nil, wanted = error", name)
			}
			if _, err := fsys.ReadFile(name); err == nil {
				t.Errorf("ReadFile(%q): got = nil, wanted = error", name)
			}
			if _, err := fsys.ReadDir(name); err == nil {
				t.Errorf("ReadDir(%q): got = nil, wanted = error", name)
			}
		})
	}
}

func TestReadFile(t *testing.T) {
	cb := initCallbacks(t)
	fsys := worktreefs.New(context.Background(), cb)

	data, err := fsys.ReadFile("hello.txt")
	if err != nil {
		t.Fatalf("ReadFile: got = %v, wanted = nil", err)
	}
	if len(data) == 0 {
		t.Error("ReadFile: got empty content, wanted non-empty")
	}

	info, err := fsys.Stat("hello.txt")
	if err != nil {
		t.Fatalf("Stat: got = %v, wanted = nil", err)
	}
	if got, want := info.Size(), int64(len(data)); got != want {
		t.Errorf("Size: got = %d, wanted = %d", got, want)
	}
}

func TestNonexistent(t *testing.T) {
	cb := initCallbacks(t)
	fsys := worktreefs.New(context.Background(), cb)

	if _, err := fsys.Open("does-not-exist.txt"); err == nil {
		t.Error("Open nonexistent: got = nil, wanted = error")
	}
	if _, err := fsys.Stat("does-not-exist.txt"); err == nil {
		t.Error("Stat nonexistent: got = nil, wanted = error")
	}
	if _, err := fsys.ReadFile("does-not-exist.txt"); err == nil {
		t.Error("ReadFile nonexistent: got = nil, wanted = error")
	}
	if _, err := fsys.ReadDir("does-not-exist"); err == nil {
		t.Error("ReadDir nonexistent: got = nil, wanted = error")
	}
}
