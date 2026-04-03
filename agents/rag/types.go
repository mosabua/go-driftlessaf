/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package rag

// TaskType defines the purpose for which an embedding is optimized.
// Different task types affect how vectors are positioned in the embedding space.
type TaskType string

const (
	// TaskTypeSemanticSimilarity optimizes embeddings for measuring
	// semantic similarity between texts.
	TaskTypeSemanticSimilarity TaskType = "SEMANTIC_SIMILARITY"

	// TaskTypeRetrievalDocument optimizes embeddings for document representation
	// in retrieval systems. Use this when storing documents; pair with
	// TaskTypeRetrievalQuery when searching.
	TaskTypeRetrievalDocument TaskType = "RETRIEVAL_DOCUMENT"

	// TaskTypeRetrievalQuery optimizes embeddings for query representation
	// in retrieval systems (counterpart to TaskTypeRetrievalDocument).
	TaskTypeRetrievalQuery TaskType = "RETRIEVAL_QUERY"

	// TaskTypeQuestionAnswering optimizes embeddings for matching queries
	// with relevant answers.
	TaskTypeQuestionAnswering TaskType = "QUESTION_ANSWERING"

	// TaskTypeClassification optimizes embeddings for text classification.
	TaskTypeClassification TaskType = "CLASSIFICATION"

	// TaskTypeClustering optimizes embeddings for grouping similar texts.
	TaskTypeClustering TaskType = "CLUSTERING"
)

const (
	// DefaultTopK is the default number of results returned by a search.
	DefaultTopK = 5

	// MetadataKeySourceText is the metadata key used to store the original
	// source text alongside embeddings, enabling re-embedding with newer models.
	MetadataKeySourceText = "_source_text"

	// MetadataKeyStoredAt is the metadata key for the timestamp when a
	// datapoint was stored.
	MetadataKeyStoredAt = "_stored_at"
)

// SearchOptions configures a vector similarity search.
//
// # Distance threshold
//
// DistanceThreshold controls which results are returned based on their cosine
// distance from the query vector. Cosine distance ranges from 0 (identical) to
// 2 (opposite), with lower values indicating higher similarity.
//
// By default, no threshold filtering is applied — all TopK results are returned
// regardless of distance. This is intentional: the right threshold depends
// entirely on your corpus, embedding model, and use case. We strongly recommend
// examining your result distances before setting a threshold.
//
// To find the right threshold for your corpus:
//
//  1. Run searches with no threshold (the default) and examine the distances
//  2. Note the distance range for results you consider "good" vs "irrelevant"
//  3. Set DistanceThreshold to a value that separates the two groups
//
// Typical ranges (these vary by corpus — always verify with your own data):
//
//   - 0.0–0.3: Very similar (near-duplicate content, same error with minor variations)
//   - 0.3–0.5: Moderately similar (same category of problem, related issues)
//   - 0.5–0.8: Loosely related (same domain but different specifics)
//   - 0.8+:    Weak or no meaningful similarity
type SearchOptions struct {
	// TopK is the maximum number of results to return.
	// Defaults to DefaultTopK (5) when zero.
	TopK int

	// DistanceThreshold is the maximum cosine distance for results.
	// Lower values mean stricter matching (higher similarity required).
	//
	// When zero (the default), no threshold filtering is applied and all
	// TopK results are returned. This lets you examine raw distances to
	// calibrate the right threshold for your corpus.
	//
	// Set to a positive value (e.g., 0.4) to filter out results with
	// distance greater than that value.
	//
	// Examples:
	//   SearchOptions{TopK: 5}                       // no filtering, return all 5
	//   SearchOptions{TopK: 10, DistanceThreshold: 0.3} // strict: only very similar
	//   SearchOptions{TopK: 10, DistanceThreshold: 0.6} // moderate: related content
	DistanceThreshold float64
}

// Result represents a single vector search result.
type Result struct {
	// ID is the unique identifier of the matched datapoint.
	ID string

	// Distance is the cosine distance from the query vector.
	// Lower values indicate higher similarity (0 = identical, 2 = opposite).
	Distance float64

	// Metadata contains the key-value pairs stored with the datapoint.
	Metadata map[string]string
}

// defaults fills in zero-value fields with sensible defaults.
// DistanceThreshold 0 means "no filtering" — all TopK results are returned.
func (o SearchOptions) defaults() SearchOptions {
	if o.TopK <= 0 {
		o.TopK = DefaultTopK
	}
	return o
}
