/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package metapathreconciler

import (
	"context"
	"fmt"

	"chainguard.dev/driftlessaf/agents/metaagent"
	"chainguard.dev/driftlessaf/agents/promptbuilder"
	"chainguard.dev/driftlessaf/agents/toolcall/callbacks"
	"chainguard.dev/driftlessaf/reconcilers/githubreconciler"
	"chainguard.dev/driftlessaf/reconcilers/githubreconciler/changemanager"
	"chainguard.dev/driftlessaf/reconcilers/githubreconciler/clonemanager"
	"chainguard.dev/driftlessaf/reconcilers/githubreconciler/statusmanager"
	"github.com/chainguard-dev/clog"
	"github.com/google/go-github/v75/github"
)

// Mode controls which behaviors the reconciler performs.
// Modes can be combined with bitwise OR.
type Mode int

const (
	// ModeFix handles paths and own PRs.
	ModeFix Mode = 1 << iota
	// ModeReview reviews other PRs.
	ModeReview
	// ModeNone disables all behaviors.
	ModeNone Mode = 0
	// ModeAll handles paths, own PRs, and reviews other PRs.
	ModeAll = ModeFix | ModeReview
)

// ShouldFix reports whether m includes fix behavior.
func (m Mode) ShouldFix() bool { return m&ModeFix != 0 }

// ShouldReview reports whether m includes review behavior.
func (m Mode) ShouldReview() bool { return m&ModeReview != 0 }

// Reconciler is a generic reconciler for metaagent-based path handlers.
type Reconciler[Req promptbuilder.Bindable, Resp Result, CB any] struct {
	identity      string
	analyzer      Analyzer
	statusManager *statusmanager.StatusManager[CheckDetails]
	changeManager *changemanager.CM[PRData[Req]]
	cloneMeta     *clonemanager.Meta
	prLabels      []string
	mode          Mode

	// Agent and its adapters
	agent          metaagent.Agent[Req, Resp, CB]
	buildRequest   func(context.Context, []callbacks.Finding) (Req, error)
	buildCallbacks func(context.Context, *changemanager.Session[PRData[Req]], *clonemanager.Lease) (CB, error)
}

// Option configures a Reconciler.
type Option func(*option)

type option struct {
	mode Mode
}

// WithMode configures the reconciler's operating mode.
func WithMode(m Mode) Option {
	return func(o *option) {
		o.mode = m
	}
}

// New creates a new generic metaagent path reconciler.
func New[Req promptbuilder.Bindable, Resp Result, CB any](
	ctx context.Context,
	identity string,
	analyzer Analyzer,
	changeManager *changemanager.CM[PRData[Req]],
	cloneMeta *clonemanager.Meta,
	prLabels []string,
	agent metaagent.Agent[Req, Resp, CB],
	buildRequest func(context.Context, []callbacks.Finding) (Req, error),
	buildCallbacks func(context.Context, *changemanager.Session[PRData[Req]], *clonemanager.Lease) (CB, error),
	opts ...Option,
) (*Reconciler[Req, Resp, CB], error) {
	o := option{mode: ModeAll}
	for _, opt := range opts {
		opt(&o)
	}

	sm, err := statusmanager.NewStatusManager[CheckDetails](ctx, identity)
	if err != nil {
		return nil, fmt.Errorf("create status manager: %w", err)
	}
	return &Reconciler[Req, Resp, CB]{
		identity:       identity,
		analyzer:       analyzer,
		statusManager:  sm,
		changeManager:  changeManager,
		cloneMeta:      cloneMeta,
		prLabels:       prLabels,
		mode:           o.mode,
		agent:          agent,
		buildRequest:   buildRequest,
		buildCallbacks: buildCallbacks,
	}, nil
}

// Reconcile processes a path or pull request URL.
// For paths: runs the analyzer and agent to create/update a PR.
// For PRs: extracts the original path from the branch name and queues it.
func (r *Reconciler[Req, Resp, CB]) Reconcile(ctx context.Context, res *githubreconciler.Resource, gh *github.Client) error {
	log := clog.FromContext(ctx)

	switch res.Type {
	case githubreconciler.ResourceTypePath:
		if !r.mode.ShouldFix() {
			log.Info("Skipping path (review-only mode)")
			return nil
		}
		return r.reconcilePath(ctx, res, gh)
	case githubreconciler.ResourceTypePullRequest:
		return r.reconcilePullRequest(ctx, res, gh)
	default:
		log.With("type", res.Type).Warn("Unexpected resource type")
		return nil
	}
}
