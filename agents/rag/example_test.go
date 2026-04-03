/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package rag_test

import (
	"fmt"

	"chainguard.dev/driftlessaf/agents/rag"
)

func ExampleSearchOptions() {
	// Default: TopK=5, no distance filtering. All results returned so you
	// can examine distances and calibrate a threshold for your corpus.
	opts := rag.SearchOptions{}
	fmt.Println("default TopK:", opts.TopK, "-> defaults to", rag.DefaultTopK)

	// After examining your results, set a threshold. Typical ranges:
	//   0.0–0.3: very similar (near-duplicates)
	//   0.3–0.5: moderately similar (same category)
	//   0.5–0.8: loosely related
	strict := rag.SearchOptions{TopK: 10, DistanceThreshold: 0.3}
	fmt.Printf("strict: TopK=%d, threshold=%.1f\n", strict.TopK, strict.DistanceThreshold)

	// More permissive — include related but not identical content.
	moderate := rag.SearchOptions{TopK: 10, DistanceThreshold: 0.6}
	fmt.Printf("moderate: TopK=%d, threshold=%.1f\n", moderate.TopK, moderate.DistanceThreshold)

	// Output:
	// default TopK: 0 -> defaults to 5
	// strict: TopK=10, threshold=0.3
	// moderate: TopK=10, threshold=0.6
}

func ExampleNewMultiStore() {
	// MultiStore fans out writes to multiple Store backends.
	// Typical usage: GCSStore (durability) + MatchingEngineStore (search).
	//
	//   gcsStore, _ := rag.NewGCSStore(ctx, "my-bucket", "prefix")
	//   meStore, _ := rag.NewMatchingEngineStore(ctx, "us-central1", indexName)
	//   store := rag.NewMultiStore(gcsStore, meStore)
	//
	// All stores are attempted regardless of individual failures.
	// Errors are collected via errors.Join.
	fmt.Println("MultiStore fans out writes to all stores")
	// Output: MultiStore fans out writes to all stores
}

func ExampleClientConfig() {
	// ClientConfig wires up all three RAG components.
	cfg := rag.ClientConfig{
		Project:          "my-project",
		Location:         "us-central1",
		EmbeddingModel:   "gemini-embedding-001",
		IndexName:        "projects/my-project/locations/us-central1/indexes/12345",
		IndexEndpointID:  "67890",
		DeployedIndexID:  "my_deployed_index",
		PublicDomainName: "1234.us-central1-5678.vdb.vertexai.goog",
		// Optional: enable GCS dual-write for durability / re-embedding.
		GCSBucket: "my-embeddings-bucket",
		GCSPrefix: "build-failures",
	}

	fmt.Println("project:", cfg.Project)
	fmt.Println("model:", cfg.EmbeddingModel)
	fmt.Println("gcs:", cfg.GCSBucket)
	// Output:
	// project: my-project
	// model: gemini-embedding-001
	// gcs: my-embeddings-bucket
}
