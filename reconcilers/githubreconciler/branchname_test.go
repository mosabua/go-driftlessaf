/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func TestPathToBranchSuffix(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{{
		name: "leading dot in first component",
		path: ".github/workflows",
		want: "_dot_github/workflows",
	}, {
		name: "no dots at all",
		path: "api-impl/internal/registry",
		want: "api-impl/internal/registry",
	}, {
		name: "dots in every component",
		path: ".hidden/.nested/.deep",
		want: "_dot_hidden/_dot_nested/_dot_deep",
	}, {
		name: "simple path without dots",
		path: "normal/path",
		want: "normal/path",
	}, {
		name: "single dot-prefixed component",
		path: ".dotfile",
		want: "_dot_dotfile",
	}, {
		name: "mixed dot and non-dot components",
		path: "a/.b/c/.d",
		want: "a/_dot_b/c/_dot_d",
	}, {
		name: "dot in middle of component is not escaped",
		path: "my.file/other.dir",
		want: "my.file/other.dir",
	}, {
		name: "single component no dot",
		path: "README",
		want: "README",
	}, {
		name: "double dot prefix",
		path: "..weird/path",
		want: "_dot_.weird/path",
	}, {
		name: "dot-only component",
		path: "a/./b",
		want: "a/_dot_/b",
	}, {
		name: "deeply nested with trailing dot-file",
		path: "a/b/c/d/.env",
		want: "a/b/c/d/_dot_env",
	}, {
		name: "component starting with _dot_ is left alone",
		path: "_dot_already/safe",
		want: "_dot_already/safe",
	}, {
		name: "empty string",
		path: "",
		want: "",
	}, {
		name: "single dot",
		path: ".",
		want: "_dot_",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PathToBranchSuffix(tt.path)
			if got != tt.want {
				t.Errorf("PathToBranchSuffix(%q): got = %q, wanted = %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestBranchSuffixToPath(t *testing.T) {
	tests := []struct {
		name   string
		suffix string
		want   string
	}{{
		name:   "escaped dot-github",
		suffix: "_dot_github/workflows",
		want:   ".github/workflows",
	}, {
		name:   "no escaping needed",
		suffix: "api-impl/internal/registry",
		want:   "api-impl/internal/registry",
	}, {
		name:   "multiple escaped components",
		suffix: "_dot_hidden/_dot_nested/file",
		want:   ".hidden/.nested/file",
	}, {
		name:   "simple path passthrough",
		suffix: "normal/path",
		want:   "normal/path",
	}, {
		name:   "escaped single component",
		suffix: "_dot_dotfile",
		want:   ".dotfile",
	}, {
		name:   "escaped dot-only component",
		suffix: "a/_dot_/b",
		want:   "a/./b",
	}, {
		name:   "empty string",
		suffix: "",
		want:   "",
	}, {
		name:   "bare _dot_ is just a dot",
		suffix: "_dot_",
		want:   ".",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BranchSuffixToPath(tt.suffix)
			if got != tt.want {
				t.Errorf("BranchSuffixToPath(%q): got = %q, wanted = %q", tt.suffix, got, tt.want)
			}
		})
	}
}

func TestPathBranchRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{{
		name: "dot-github workflows",
		path: ".github/workflows",
	}, {
		name: "regular Go module path",
		path: "api-impl/internal/registry/v2alpha1",
	}, {
		name: "nested hidden dirs",
		path: ".hidden/.nested/.deep/file.go",
	}, {
		name: "simple path",
		path: "normal/path",
	}, {
		name: "single dot-prefixed file",
		path: ".dotfile",
	}, {
		name: "mixed dot and non-dot",
		path: "a/.b/c/.d/e",
	}, {
		name: "dots in middle of names preserved",
		path: "my.file/some.thing/other.dir",
	}, {
		name: "single component",
		path: "README",
	}, {
		name: "double dot prefix",
		path: "..weird/path",
	}, {
		name: "deeply nested dot-env",
		path: "services/api/deploy/.env.production",
	}, {
		name: "dot only component",
		path: "a/./b",
	}, {
		name: "empty string",
		path: "",
	}, {
		name: "single dot",
		path: ".",
	}, {
		name: "random path with dot dir",
		path: fmt.Sprintf("path/%d/.dir/%d", rand.Int63(), rand.Int63()),
	}, {
		name: "typical monorepo paths",
		path: "bots/skillup/internal/agents",
	}, {
		name: "github actions path",
		path: ".github/workflows/ci.yml",
	}, {
		name: "gitignore at root",
		path: ".gitignore",
	}, {
		name: "claude hidden dir",
		path: ".claude/skills/go-standards.md",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suffix := PathToBranchSuffix(tt.path)
			got := BranchSuffixToPath(suffix)
			if got != tt.path {
				t.Errorf("round trip failed: path = %q → suffix = %q → got = %q", tt.path, suffix, got)
			}
		})
	}
}

func TestSanitizedSuffixHasNoDotComponents(t *testing.T) {
	// Verify that PathToBranchSuffix produces branch names that git will accept
	// (no path component starts with ".").
	paths := []string{
		".github/workflows",
		".hidden/.nested/.deep",
		"a/.b/c/.d",
		".dotfile",
		"..double-dot/prefix",
		"a/./b",
		".",
		".a/.b/.c/.d/.e",
		".github/workflows/ci.yml",
		".claude/skills/go-standards.md",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			suffix := PathToBranchSuffix(path)
			for component := range strings.SplitSeq(suffix, "/") {
				if strings.HasPrefix(component, ".") {
					t.Errorf("sanitized suffix %q still has dot-prefixed component %q (from path %q)", suffix, component, path)
				}
			}
		})
	}
}

func TestFullBranchNameRoundTrip(t *testing.T) {
	// Simulate the full flow: identity + "/" + sanitized path → extract path from branch.
	// This mirrors the actual code in changemanager and metapathreconciler.
	tests := []struct {
		identity string
		path     string
	}{{
		identity: "staging-skillup",
		path:     ".github/workflows",
	}, {
		identity: "staging-skillup",
		path:     "api-impl/internal/registry/v2alpha1",
	}, {
		identity: "staging-skillup",
		path:     ".claude/skills/go-standards.md",
	}, {
		identity: "prod-bot",
		path:     ".github/workflows/ci.yml",
	}, {
		identity: "my-bot",
		path:     "normal/path/to/module",
	}, {
		identity: "bot",
		path:     ".a/.b/.c",
	}}

	for _, tt := range tests {
		name := fmt.Sprintf("%s/%s", tt.identity, tt.path)
		t.Run(name, func(t *testing.T) {
			// Build branch name (mirrors changemanager/manager.go and metapathreconciler/path.go)
			branchName := tt.identity + "/" + PathToBranchSuffix(tt.path)

			// Verify no dot-prefixed components in the branch name
			for component := range strings.SplitSeq(branchName, "/") {
				if strings.HasPrefix(component, ".") {
					t.Errorf("branch name %q has dot-prefixed component %q", branchName, component)
				}
			}

			// Extract path back (mirrors metapathreconciler/pr.go)
			prefix := tt.identity + "/"
			if !strings.HasPrefix(branchName, prefix) {
				t.Fatalf("branch name %q does not start with prefix %q", branchName, prefix)
			}
			gotPath := BranchSuffixToPath(strings.TrimPrefix(branchName, prefix))
			if gotPath != tt.path {
				t.Errorf("full round trip: got = %q, wanted = %q (branch was %q)", gotPath, tt.path, branchName)
			}
		})
	}
}

func TestPathsWithUnderscoreDotPrefixRoundTrip(t *testing.T) {
	// Edge case: a path that already contains "_dot_" as a literal directory name.
	// BranchSuffixToPath will convert it to a dot-prefixed component, meaning a path
	// with a literal "_dot_" component does NOT round-trip. This test documents that
	// behavior — such paths are extremely unlikely in practice.
	path := "_dot_github/workflows"
	suffix := PathToBranchSuffix(path)

	// "_dot_github" does NOT start with ".", so PathToBranchSuffix leaves it alone.
	if suffix != "_dot_github/workflows" {
		t.Errorf("PathToBranchSuffix(%q): got = %q, wanted = %q", path, suffix, "_dot_github/workflows")
	}

	// BranchSuffixToPath converts "_dot_" prefix back to ".", so we get ".github/workflows".
	got := BranchSuffixToPath(suffix)
	if got != ".github/workflows" {
		t.Errorf("BranchSuffixToPath(%q): got = %q, wanted = %q", suffix, got, ".github/workflows")
	}

	// This demonstrates the collision: a literal "_dot_github" directory would be
	// misinterpreted. This is acceptable because "_dot_" prefixed directories are
	// not a real-world concern.
	if got == path {
		t.Error("unexpected: literal _dot_ prefix round-tripped, but it shouldn't")
	}
}
