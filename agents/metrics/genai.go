/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"context"

	"chainguard.dev/driftlessaf/agents/agenttrace"
	"github.com/chainguard-dev/clog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// GenAI provides OpenTelemetry metrics for generative AI operations.
// It includes counters for token usage (prompt and completion), tool calls,
// and prompt cache metrics, with support for graceful degradation if metric
// creation fails.
type GenAI struct {
	meter            metric.Meter
	promptTokens     metric.Int64Counter
	completionTokens metric.Int64Counter
	toolCallCounter  metric.Int64Counter
}

// NewGenAI creates a new GenAI metrics instance with the specified meter name.
// Uses graceful degradation: if any metric counter fails to initialize, logs a warning
// and uses a no-op counter instead of failing entirely.
//
// The meterName should be unified across all agent executors (e.g., "chainguard.ai.agents")
// with the model name serving as a dimension on the recorded metrics to differentiate
// between different models (Claude, Gemini, etc.).
func NewGenAI(meterName string) *GenAI {
	meter := otel.Meter(meterName, metric.WithInstrumentationVersion("1.0.0"))

	// Create prompt tokens counter with graceful degradation
	promptTokens, err := meter.Int64Counter("genai.token.prompt",
		metric.WithDescription("The number of prompt tokens used"),
		metric.WithUnit("{tokens}"))
	if err != nil {
		clog.WarnContext(context.Background(), "Failed to create prompt tokens counter, metrics will be disabled", "error", err, "meter", meterName)
		promptTokens = noop.Int64Counter{}
	}

	// Create completion tokens counter with graceful degradation
	completionTokens, err := meter.Int64Counter("genai.token.completion",
		metric.WithDescription("The number of completion tokens used"),
		metric.WithUnit("{tokens}"))
	if err != nil {
		clog.WarnContext(context.Background(), "Failed to create completion tokens counter, metrics will be disabled", "error", err, "meter", meterName)
		completionTokens = noop.Int64Counter{}
	}

	// Create tool call counter with graceful degradation
	toolCallCounter, err := meter.Int64Counter("genai.tool.calls",
		metric.WithDescription("The number of tool calls made during execution"),
		metric.WithUnit("{calls}"))
	if err != nil {
		clog.WarnContext(context.Background(), "Failed to create tool call counter, metrics will be disabled", "error", err, "meter", meterName)
		toolCallCounter = noop.Int64Counter{}
	}

	return &GenAI{
		meter:            meter,
		promptTokens:     promptTokens,
		completionTokens: completionTokens,
		toolCallCounter:  toolCallCounter,
	}
}

// RecordTokens records prompt and completion token usage.
// Enriches attributes from the execution context propagated via context.Context.
func (m *GenAI) RecordTokens(ctx context.Context, model string, promptTokens, completionTokens int64, attrs ...attribute.KeyValue) {
	baseAttrs := []attribute.KeyValue{
		attribute.String("model", model),
	}
	baseAttrs = agenttrace.GetExecutionContext(ctx).EnrichAttributes(baseAttrs)
	baseAttrs = append(baseAttrs, attrs...)

	m.promptTokens.Add(ctx, promptTokens, metric.WithAttributes(baseAttrs...))
	m.completionTokens.Add(ctx, completionTokens, metric.WithAttributes(baseAttrs...))
}

// RecordCacheTokens records Anthropic prompt cache token usage on the existing
// prompt tokens counter using gen_ai.token.type dimension values "cache_read"
// and "cache_creation". This aligns with the OpenTelemetry GenAI semantic
// conventions pattern from gen_ai.client.token.usage (see #35633).
//
// cacheRead: tokens served from cache (cheap — 0.1x base input price).
// cacheCreation: tokens written to cache (1.25x base input price, amortized over reads).
//
// A healthy caching setup shows high cache_read and low/zero cache_creation
// after the first turn.
func (m *GenAI) RecordCacheTokens(ctx context.Context, model string, cacheRead, cacheCreation int64, attrs ...attribute.KeyValue) {
	baseAttrs := []attribute.KeyValue{
		attribute.String("model", model),
	}
	baseAttrs = agenttrace.GetExecutionContext(ctx).EnrichAttributes(baseAttrs)
	baseAttrs = append(baseAttrs, attrs...)

	if cacheRead > 0 {
		readAttrs := append(append([]attribute.KeyValue{}, baseAttrs...), attribute.String("gen_ai.token.type", "cache_read"))
		m.promptTokens.Add(ctx, cacheRead, metric.WithAttributes(readAttrs...))
	}
	if cacheCreation > 0 {
		creationAttrs := append(append([]attribute.KeyValue{}, baseAttrs...), attribute.String("gen_ai.token.type", "cache_creation"))
		m.promptTokens.Add(ctx, cacheCreation, metric.WithAttributes(creationAttrs...))
	}
}

// RecordToolCall records a tool invocation.
// Enriches attributes from the execution context propagated via context.Context.
func (m *GenAI) RecordToolCall(ctx context.Context, model, toolName string, attrs ...attribute.KeyValue) {
	baseAttrs := []attribute.KeyValue{
		attribute.String("model", model),
		attribute.String("tool", toolName),
	}
	baseAttrs = agenttrace.GetExecutionContext(ctx).EnrichAttributes(baseAttrs)
	baseAttrs = append(baseAttrs, attrs...)

	m.toolCallCounter.Add(ctx, 1, metric.WithAttributes(baseAttrs...))
}
