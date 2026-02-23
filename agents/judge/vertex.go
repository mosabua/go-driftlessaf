/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package judge

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"chainguard.dev/driftlessaf/agents/executor/claudeexecutor"
	"chainguard.dev/driftlessaf/agents/executor/googleexecutor"
	"chainguard.dev/driftlessaf/agents/metrics"
)

// NewVertex creates a new Interface instance by delegating to the appropriate
// implementation based on the model name. Claude models use Anthropic SDK,
// Gemini models use Google's Generative AI SDK.
// Accepts optional executor options that will be passed through to the underlying executor.
func NewVertex(ctx context.Context, projectID, region, modelName string, opts ...any) (Interface, error) {
	modelLower := strings.ToLower(modelName)

	// Delegate to Claude implementation for Claude models
	if strings.HasPrefix(modelLower, "claude-") {
		// Extract Claude options
		claudeOpts := make([]claudeexecutor.Option[*Request, *Judgement], 0, len(opts))
		for _, opt := range opts {
			if claudeOpt, ok := opt.(claudeexecutor.Option[*Request, *Judgement]); ok {
				claudeOpts = append(claudeOpts, claudeOpt)
			}
		}
		return newClaude(ctx, projectID, region, modelName, claudeOpts...)
	}

	// Delegate to Google implementation for Gemini models
	if strings.HasPrefix(modelLower, "gemini-") {
		// Extract Google options
		googleOpts := make([]googleexecutor.Option[*Request, *Judgement], 0, len(opts))
		for _, opt := range opts {
			if googleOpt, ok := opt.(googleexecutor.Option[*Request, *Judgement]); ok {
				googleOpts = append(googleOpts, googleOpt)
			}
		}
		return newGoogle(ctx, projectID, region, modelName, googleOpts...)
	}

	return nil, fmt.Errorf("unsupported model: %s (expected claude-* or gemini-*)", modelName)
}

// NewVertexWithEnricher creates a new Interface instance with an attribute enricher for metrics.
// This is a convenience function for the common case of passing an enricher.
func NewVertexWithEnricher(ctx context.Context, projectID, region, modelName string, enricher metrics.AttributeEnricher, resourceLabels ...map[string]string) (Interface, error) {
	modelLower := strings.ToLower(modelName)

	// Build executor options
	labels := make(map[string]string)
	if len(resourceLabels) > 0 {
		// Copy existing labels
		maps.Copy(labels, resourceLabels[0])
	}
	labels["model_name"] = modelLower

	// Delegate to Claude implementation for Claude models
	if strings.HasPrefix(modelLower, "claude-") {
		opts := []claudeexecutor.Option[*Request, *Judgement]{
			claudeexecutor.WithAttributeEnricher[*Request, *Judgement](enricher),
			claudeexecutor.WithResourceLabels[*Request, *Judgement](labels),
		}
		return newClaude(ctx, projectID, region, modelName, opts...)
	}

	// Delegate to Google implementation for Gemini models
	if strings.HasPrefix(modelLower, "gemini-") {
		opts := []googleexecutor.Option[*Request, *Judgement]{
			googleexecutor.WithAttributeEnricher[*Request, *Judgement](enricher),
			googleexecutor.WithResourceLabels[*Request, *Judgement](labels),
		}
		return newGoogle(ctx, projectID, region, modelName, opts...)
	}

	return nil, fmt.Errorf("unsupported model: %s (expected claude-* or gemini-*)", modelName)
}
