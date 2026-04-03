/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

/*
Package rag provides embedding generation, vector storage, and semantic retrieval
for driftlessAF agents.

This package enables agents to learn from historical data by storing embeddings
of past outcomes (fixes, advisories, build configurations) and retrieving
semantically similar records at inference time.

# Architecture

The package has three core components:

  - [Embedder]: Generates vector embeddings from text using Vertex AI (gemini-embedding-001).
  - [Store]: Persists embeddings with metadata in a vector index (Vertex AI Matching Engine).
  - [Retriever]: Searches the vector index for similar embeddings.

Each component is defined as an interface (Store, Retriever) with Vertex AI Matching Engine
implementations provided. The [Client] type wraps all three for convenience.

# Basic Usage

Store a document embedding:

	client, err := rag.NewClient(ctx, rag.ClientConfig{
		Project:         "my-project",
		Location:        "us-east5",
		EmbeddingModel:  "gemini-embedding-001",
		IndexName:       "projects/my-project/locations/us-east5/indexes/12345",
		IndexEndpointID: "67890",
		DeployedIndexID: "my_deployed_index",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Store a past fix
	err = client.EmbedAndStore(ctx, "fix-123", "dependency version conflict in go.mod",
		rag.TaskTypeRetrievalDocument,
		map[string]string{
			"pr_url": "https://github.com/org/repo/pull/456",
			"patch":  "--- a/go.mod\n+++ b/go.mod\n...",
		})

Search for similar past fixes:

	// Start with no threshold to examine raw distances, then set one for your corpus.
	results, err := client.EmbedAndSearch(ctx, "cannot find module providing package foo/bar",
		rag.SearchOptions{TopK: 3})
	for _, r := range results {
		fmt.Printf("Similar fix (distance %.3f): %s\n", r.Distance, r.Metadata["pr_url"])
	}

# MCP Integration

This package is designed to power a RAG MCP server that exposes search/retrieve
tools to any driftlessAF agent via the Model Context Protocol. See the rag-mcp-server
service for the MCP integration layer.
*/
package rag
