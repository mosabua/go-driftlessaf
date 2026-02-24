/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package metapathreconciler

import (
	"context"
	"fmt"

	"chainguard.dev/driftlessaf/agents/toolcall/callbacks"
	gogit "github.com/go-git/go-git/v5"
)

// Result is implemented by all agent result types.
// The commit message is used when pushing changes to the repository.
type Result interface {
	GetCommitMessage() string
}

// Analyzer runs a static analysis tool over a worktree and returns diagnostics.
// Each path is relative to the repo root (e.g., "path/to/package").
type Analyzer interface {
	// Analyze runs the tool scoped to the given paths within the worktree
	// and returns diagnostics. An empty slice means the paths are clean.
	Analyze(ctx context.Context, wt *gogit.Worktree, paths ...string) ([]Diagnostic, error)
}

// Diagnostic represents a single issue discovered by an Analyzer.
type Diagnostic struct {
	// Path is the file path relative to the repo root.
	Path string

	// Line is the line number (0 if not applicable).
	Line int

	// Message is a human-readable description of the issue.
	Message string

	// Rule is the specific check/rule ID (e.g., "S1000", "modernize").
	Rule string
}

// AsFinding converts a Diagnostic into a Finding so that diagnostics and
// CI/review findings can be combined into a single slice for the metaagent.
func (d Diagnostic) AsFinding() callbacks.Finding {
	id := d.Rule + ":" + d.Path
	if d.Line > 0 {
		id += fmt.Sprintf(":%d", d.Line)
	}
	details := d.Path
	if d.Line > 0 {
		details += fmt.Sprintf(":%d", d.Line)
	}
	details += ": " + d.Message

	return callbacks.Finding{
		Kind:       callbacks.FindingKindCICheck,
		Identifier: id,
		Details:    details,
	}
}

// PRData is the data embedded in PR bodies for change detection.
// This is used by the changemanager to track state across reconciliations.
type PRData struct {
	Identity string `json:"identity"`
	Path     string `json:"path"`
}
