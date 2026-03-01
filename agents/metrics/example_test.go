/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package metrics_test

import (
	"context"
	"fmt"

	"chainguard.dev/driftlessaf/agents/metrics"
	"go.opentelemetry.io/otel/attribute"
)

// ExampleNewGenAI demonstrates creating a new GenAI metrics instance.
func ExampleNewGenAI() {
	// Create a metrics instance with a unified meter name
	// The meter name should be consistent across all agent executors
	m := metrics.NewGenAI("chainguard.ai.agents")

	// The metrics instance is ready to use
	fmt.Printf("Metrics instance created: %T\n", m)

	// Output:
	// Metrics instance created: *metrics.GenAI
}

// ExampleGenAI_RecordTokens demonstrates recording token usage.
func ExampleGenAI_RecordTokens() {
	ctx := context.Background()
	m := metrics.NewGenAI("chainguard.ai.agents")

	// Record token usage from an AI model response
	// Parameters: context, model name, prompt tokens, completion tokens
	m.RecordTokens(ctx, "claude-3-sonnet", 150, 250)

	// Record with additional custom attributes
	m.RecordTokens(ctx, "gemini-pro", 100, 200,
		attribute.String("agent", "code-reviewer"),
		attribute.Int("turn", 1))

	fmt.Println("Token metrics recorded")

	// Output:
	// Token metrics recorded
}

// ExampleGenAI_RecordToolCall demonstrates recording tool invocations.
func ExampleGenAI_RecordToolCall() {
	ctx := context.Background()
	m := metrics.NewGenAI("chainguard.ai.agents")

	// Record a tool call with model and tool name
	m.RecordToolCall(ctx, "claude-3-sonnet", "read_file")

	// Record with additional custom attributes
	m.RecordToolCall(ctx, "claude-3-sonnet", "edit_file",
		attribute.String("file_type", "go"),
		attribute.String("operation", "modify"))

	fmt.Println("Tool call metrics recorded")

	// Output:
	// Tool call metrics recorded
}

// ExampleGenAI_SetAttributeEnricher demonstrates adding contextual attributes.
func ExampleGenAI_SetAttributeEnricher() {
	ctx := context.Background()
	m := metrics.NewGenAI("chainguard.ai.agents")

	// Define an enricher that adds PR context to all metrics
	enricher := func(_ context.Context, baseAttrs []attribute.KeyValue) []attribute.KeyValue {
		return append(baseAttrs,
			attribute.String("repository", "chainguard-dev/example"),
			attribute.Int("pull_request", 123),
			attribute.String("commit_sha", "abc123"),
		)
	}

	// Set the enricher on the metrics instance
	m.SetAttributeEnricher(enricher)

	// All subsequent metrics will include the enriched attributes
	m.RecordTokens(ctx, "claude-3-sonnet", 100, 200)
	m.RecordToolCall(ctx, "claude-3-sonnet", "search_codebase")

	fmt.Println("Metrics recorded with enriched attributes")

	// Output:
	// Metrics recorded with enriched attributes
}

// ExampleAttributeEnricher demonstrates creating a custom attribute enricher.
func ExampleAttributeEnricher() {
	// Create an enricher that extracts context from the request
	var enricher metrics.AttributeEnricher = func(ctx context.Context, baseAttrs []attribute.KeyValue) []attribute.KeyValue {
		// In a real application, you might extract values from context
		// For example: repository info, user ID, request ID, etc.
		return append(baseAttrs,
			attribute.String("environment", "production"),
			attribute.String("service", "code-review-agent"),
		)
	}

	m := metrics.NewGenAI("chainguard.ai.agents")
	m.SetAttributeEnricher(enricher)

	fmt.Println("Custom enricher configured")

	// Output:
	// Custom enricher configured
}

// ExampleGenAI_multipleModels demonstrates tracking metrics across different models.
func ExampleGenAI_multipleModels() {
	ctx := context.Background()

	// Use a single metrics instance for all models
	// The model name serves as a dimension to differentiate between them
	m := metrics.NewGenAI("chainguard.ai.agents")

	// Track Claude usage
	m.RecordTokens(ctx, "claude-3-sonnet", 150, 250)
	m.RecordToolCall(ctx, "claude-3-sonnet", "read_file")

	// Track Gemini usage
	m.RecordTokens(ctx, "gemini-pro", 100, 300)
	m.RecordToolCall(ctx, "gemini-pro", "search_codebase")

	// Track GPT usage
	m.RecordTokens(ctx, "gpt-4", 200, 400)

	fmt.Println("Multi-model metrics recorded")

	// Output:
	// Multi-model metrics recorded
}
