/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package metareconciler provides a generic reconciler for metaagent-based
// GitHub issue handlers. It handles the common reconciliation flow:
//
//  1. Fetch issue
//  2. Create change session
//  3. Handle states (skip/closed/findings/pending)
//  4. Acquire clone lease
//  5. Run agent
//  6. Push changes
//
// The package is parameterized by request type, response type, and callbacks type,
// allowing different agents to plug in their specific logic while reusing the
// common reconciliation infrastructure.
//
// # Basic Usage
//
//	// Create the changemanager with your PR templates
//	cm, err := changemanager.New[metareconciler.PRData[*MyRequest]](identity, titleTmpl, bodyTmpl)
//
//	// Create the reconciler
//	rec := metareconciler.New(
//	    identity,
//	    cm,
//	    cloneMeta,
//	    prLabels,
//	    agent,
//	    func(ctx context.Context, issue *github.Issue, session *changemanager.Session[metareconciler.PRData[*MyRequest]]) (*MyRequest, error) {
//	        return &MyRequest{
//	            Title:    issue.GetTitle(),
//	            Body:     issue.GetBody(),
//	            Findings: session.Findings(),
//	        }, nil
//	    },
//	    func(ctx context.Context, session *changemanager.Session[metareconciler.PRData[*MyRequest]], lease *clonemanager.Lease) (MyCallbacks, error) {
//	        wt, err := lease.Repo().Worktree()
//	        if err != nil {
//	            return MyCallbacks{}, err
//	        }
//	        return toolcall.NewFindingTools(
//	            toolcall.NewWorktreeTools(toolcall.EmptyTools{}, clonemanager.WorktreeCallbacks(wt)),
//	            session.FindingCallbacks(),
//	        ), nil
//	    },
//	)
package metareconciler
