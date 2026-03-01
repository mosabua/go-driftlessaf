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
	gogit "github.com/go-git/go-git/v5"
	"github.com/google/go-github/v75/github"
)

// Reconciler is a generic reconciler for metaagent-based path handlers.
type Reconciler[Req promptbuilder.Bindable, Resp Result, CB any] struct {
	identity      string
	analyzer      Analyzer
	statusManager *statusmanager.StatusManager[CheckDetails]
	changeManager *changemanager.CM[PRData[Req]]
	cloneMeta     *clonemanager.Meta
	prLabels      []string

	// Agent and its adapters
	agent          metaagent.Agent[Req, Resp, CB]
	buildRequest   func([]callbacks.Finding) Req
	buildCallbacks func(*gogit.Worktree, *changemanager.Session[PRData[Req]]) CB
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
	buildRequest func([]callbacks.Finding) Req,
	buildCallbacks func(*gogit.Worktree, *changemanager.Session[PRData[Req]]) CB,
) (*Reconciler[Req, Resp, CB], error) {
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
		return r.reconcilePath(ctx, res, gh)
	case githubreconciler.ResourceTypePullRequest:
		return r.reconcilePullRequest(ctx, res, gh)
	default:
		log.With("type", res.Type).Warn("Unexpected resource type")
		return nil
	}
}
