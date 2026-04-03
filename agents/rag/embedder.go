/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package rag

import (
	"context"
	"errors"
	"fmt"

	"chainguard.dev/driftlessaf/agents/agenttrace"
	"github.com/chainguard-dev/clog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"google.golang.org/genai"
)

// DefaultDimensions is the output dimensionality for gemini-embedding-001.
// 3072 is the maximum supported and provides the best recall quality.
const DefaultDimensions = 3072

// Embedder generates vector embeddings from text using Vertex AI.
type Embedder struct {
	client       *genai.Client
	model        string
	dimensions   int
	tokenCounter metric.Int64Counter
}

// NewEmbedder creates a new embedding generator.
//
// The model should be a Google embedding model name (e.g., "gemini-embedding-001").
// Project and location identify the GCP project and region for Vertex AI.
// Dimensions defaults to DefaultDimensions (3072) for best recall quality.
// Pass a non-zero dimensions to override.
func NewEmbedder(ctx context.Context, project, location, model string, dimensions ...int) (*Embedder, error) {
	if project == "" {
		return nil, errors.New("project is required")
	}
	if location == "" {
		return nil, errors.New("location is required")
	}
	if model == "" {
		return nil, errors.New("model is required")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("creating genai client: %w", err)
	}

	meter := otel.Meter("chainguard.ai.rag", metric.WithInstrumentationVersion("1.0.0"))
	tokenCounter, err := meter.Int64Counter(
		"rag.embedding.tokens",
		metric.WithDescription("Estimated tokens used for embedding generation"),
		metric.WithUnit("{tokens}"),
	)
	if err != nil {
		clog.WarnContext(ctx, "Failed to create token counter metric, falling back to noop", "error", err)
		tokenCounter = noop.Int64Counter{}
	}

	dims := DefaultDimensions
	if len(dimensions) > 0 && dimensions[0] > 0 {
		dims = dimensions[0]
	}

	return &Embedder{
		client:       client,
		model:        model,
		dimensions:   dims,
		tokenCounter: tokenCounter,
	}, nil
}

// Embed generates a vector embedding for the given text.
//
// The taskType affects how the embedding is optimized. Use TaskTypeRetrievalDocument
// when storing documents and TaskTypeRetrievalQuery when searching.
// Use TaskTypeSemanticSimilarity when comparing texts directly.
func (e *Embedder) Embed(ctx context.Context, text string, taskType TaskType) ([]float32, error) {
	dims := int32(e.dimensions)
	resp, err := e.client.Models.EmbedContent(ctx, e.model, []*genai.Content{
		genai.NewContentFromText(text, genai.Role("")),
	}, &genai.EmbedContentConfig{
		TaskType:             string(taskType),
		OutputDimensionality: &dims,
	})
	if err != nil {
		return nil, fmt.Errorf("generating embedding: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, errors.New("no embeddings returned")
	}

	e.recordTokens(ctx, len(text))
	return resp.Embeddings[0].Values, nil
}

// Close releases resources held by the embedder.
func (e *Embedder) Close() error {
	// genai.Client does not have a Close method.
	return nil
}

func (e *Embedder) recordTokens(ctx context.Context, inputLength int) {
	// Estimate ~4 chars/token for English text (industry standard approximation).
	// Vertex AI embedding APIs don't return token counts.
	estimated := int64(inputLength / 4)
	if estimated == 0 {
		estimated = 1
	}

	attrs := []attribute.KeyValue{
		attribute.String("model", e.model),
		attribute.String("operation", "embedding"),
	}
	attrs = agenttrace.GetExecutionContext(ctx).EnrichAttributes(attrs)
	e.tokenCounter.Add(ctx, estimated, metric.WithAttributes(attrs...))
}
