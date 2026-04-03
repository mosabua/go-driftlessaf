# RAG: Retrieval-Augmented Generation for driftlessAF

This package provides **embedding generation**, **vector storage**, and **semantic
retrieval** as reusable infrastructure for driftlessAF agents. It enables agents to
learn from historical data by storing embeddings of past outcomes and retrieving
semantically similar records at inference time.

## What is RAG?

Retrieval-Augmented Generation (RAG) augments an LLM's context window with
relevant information retrieved from a knowledge base. Instead of relying solely on
the model's training data, RAG searches a vector index for semantically similar
content and injects it into the prompt. This allows agents to:

- **Learn from history**: Surface past fixes, advisories, or configurations that
  match a current problem.
- **Reduce hallucination**: Ground responses in real data rather than guessing.
- **Stay current**: The knowledge base can be updated continuously without
  retraining the model.

The flow is:

```
User query
    |
    v
[Embed query] --> [Search vector index] --> [Retrieve matching records]
    |                                              |
    v                                              v
                 [Inject into LLM prompt as context]
                              |
                              v
                     [Generate response]
```

## Architecture

The package has three core components, each defined as an interface with Vertex AI
Matching Engine implementations provided:

```
                    +------------------+
                    |     Client       |  Convenience wrapper
                    | EmbedAndStore()  |
                    | EmbedAndSearch() |
                    +--------+---------+
                             |
              +--------------+--------------+
              |              |              |
     +--------v--+    +------v-----+   +----v-------+
     |  Embedder  |    |   Store    |   |  Retriever |
     |  (Vertex   |    | (write)   |   |  (read)    |
     |   AI)      |    +------+----+   +----+-------+
     +------------+           |              |
                    +---------+--------+     |
                    |                  |     |
              +-----v-----+    +------v--+  |
              | GCSStore   |    | Match.  |  |
              | (durable)  |    | Engine  |  |
              +------------+    | Store   |  |
                                +----+----+  |
                                     |       |
                              +------v-------v------+
                              |  Vertex AI Matching  |
                              |  Engine (index)      |
                              +----------------------+
```

### Components

| Component | Interface | Implementation | Purpose |
|-----------|-----------|----------------|---------|
| Embedder | Concrete `*Embedder` | Vertex AI `genai.Client` | Convert text to vector embeddings |
| Store | `Store` | `MatchingEngineStore` | Write vectors to the index |
| Store | `Store` | `GCSStore` | Persist records to GCS for durability |
| Store | `Store` | `MultiStore` | Fan-out writes to multiple stores |
| Retriever | `Retriever` | `MatchingEngineRetriever` | Search the index via FindNeighbors REST API |

### gRPC retriever: connecting to public endpoints

The retriever uses the Go `MatchClient` gRPC API for FindNeighbors queries.
This avoids JSON serialization overhead for 3072-dimensional float32 vectors
and uses Google's standard authenticated gRPC transport.

**Key configuration detail:** The `MatchClient` defaults to
`aiplatform.googleapis.com:443`, which does **not** serve `FindNeighbors`
(returns `Unimplemented`). For public endpoints, you must connect directly to
the index endpoint's domain:

```go
client, err := aiplatform.NewMatchClient(ctx,
    option.WithEndpoint("1234.us-central1-5678.vdb.vertexai.goog:443"),
)
```

Note: `FindNeighbors` lives on `MatchClient`, **not** `IndexEndpointClient` —
this is a common source of confusion. Authentication uses the same
`cloud-platform` scope as all other Vertex AI APIs.

For private (VPC) endpoints, use the regional endpoint instead:

```go
option.WithEndpoint("us-central1-aiplatform.googleapis.com:443")
```

## Quick Start

### Store a document

```go
client, err := rag.NewClient(ctx, rag.ClientConfig{
    Project:         "my-project",
    Location:        "us-central1",
    EmbeddingModel:  "gemini-embedding-001",
    IndexName:       "projects/my-project/locations/us-central1/indexes/12345",
    IndexEndpointID: "67890",
    DeployedIndexID: "my_deployed_index",
    PublicDomainName: "1234.us-central1-5678.vdb.vertexai.goog",
})
if err != nil {
    log.Fatal(err)
}
defer client.Close()

err = client.EmbedAndStore(ctx, "fix-123",
    "dependency version conflict in go.mod",
    rag.TaskTypeRetrievalDocument,
    map[string]string{
        "pr_url": "https://github.com/org/repo/pull/456",
        "repo":   "my-repo",
    })
```

### Search for similar records

```go
results, err := client.EmbedAndSearch(ctx,
    "cannot resolve module dependency",
    rag.SearchOptions{TopK: 5})
for _, r := range results {
    fmt.Printf("Match (distance %.3f): %s\n", r.Distance, r.Metadata["pr_url"])
}
```

## Embedding Model and Task Types

The package uses Google's `gemini-embedding-001` model by default at 3072
dimensions, which provides the best recall quality.

### Task type pairing

When storing and searching, use the correct **asymmetric task type pair**. Google
optimizes the embedding space differently for documents vs queries:

| Operation | Task Type | When to use |
|-----------|-----------|-------------|
| Storing documents | `TaskTypeRetrievalDocument` | Ingesting records into the index |
| Searching | `TaskTypeRetrievalQuery` | Finding similar records (used automatically by `EmbedAndSearch`) |
| Comparing texts directly | `TaskTypeSemanticSimilarity` | When both texts are the same "type" (e.g., comparing two error messages) |
| Question answering | `TaskTypeQuestionAnswering` | Matching questions to answers |
| Classification | `TaskTypeClassification` | Categorizing text |
| Clustering | `TaskTypeClustering` | Grouping similar texts |

**Important**: `EmbedAndSearch` always uses `TaskTypeRetrievalQuery` automatically.
When ingesting data, use `TaskTypeRetrievalDocument` for best results. These two
are designed as a pair.

## Search Options

```go
// Start with no threshold to see raw distances for your corpus.
rag.SearchOptions{TopK: 10}

// Once you know what "good" distances look like, set a threshold.
rag.SearchOptions{TopK: 10, DistanceThreshold: 0.4}
```

- **TopK**: Maximum number of results to return. Defaults to 5. The retriever
  requests `TopK * 2` neighbors from the API to leave room for threshold
  filtering.
- **DistanceThreshold**: Maximum cosine distance. Lower values = stricter
  matching. **Defaults to 0 (no filtering)** — all TopK results are returned
  regardless of distance.

### Choosing a threshold

The right threshold depends on your corpus, embedding model, and use case. There
is no universal default. We recommend starting without a threshold, examining your
results, and then setting one based on real data:

1. Run searches with `SearchOptions{TopK: 20}` (no threshold)
2. Look at the `Distance` field in each result
3. Identify where "useful" results end and noise begins
4. Set `DistanceThreshold` to that boundary value

**Typical distance ranges** (cosine distance, 0 = identical, 2 = opposite):

| Range | Meaning | Example |
|-------|---------|---------|
| 0.0–0.3 | Very similar | Same error with minor wording differences |
| 0.3–0.5 | Moderately similar | Same category of problem, related root causes |
| 0.5–0.8 | Loosely related | Same domain but different specifics |
| 0.8+ | Weak or no meaningful similarity | Likely noise |

These ranges are **guidelines only** — always verify with your own data. Different
corpora and embedding models produce different distance distributions.

## Stores and Durability

When you call `EmbedAndStore`, the vector is written to Matching Engine via a
**stream update** — it's searchable immediately with no batch job required.

GCS is optional but recommended. When configured, `MultiStore` writes to both
simultaneously:

```
EmbedAndStore("fix-123", "dependency conflict in go.mod", ...)
    |
    ├──> GCS: full JSON record (text + metadata + vector + timestamp)
    |         gs://my-bucket/build-failures/fix-123.json
    |
    └──> Matching Engine: vector + metadata (searchable immediately)
```

### Why both?

| | Matching Engine | GCS |
|--|----------------|-----|
| Purpose | Search | Durability / re-embedding |
| Searchable? | Yes | No (just JSON files) |
| Can list all records? | No | Yes |
| Needed for search? | Yes | No |
| Write latency | Immediate (stream update) | Immediate |

The search index is great for searching, but it's a black box — you can't easily
list everything in it, export it, or reprocess it. GCS gives you a durable,
readable copy of every record.

### Re-embedding with a better model

This is the killer use case for GCS dual-write. Say Google releases
`gemini-embedding-002` with better recall. Your vectors from `001` are now stale.
Without GCS, you'd need to go back to the **original data source** (whatever
system produced the build failures), re-analyze everything, and re-embed.

With GCS, every record stores the original text under `_source_text`. You just:

1. Read all the JSON files from GCS
2. Re-embed each `source_text` with the new model
3. Upsert the new vectors into Matching Engine

The ingest tool (`cmd/ingest`) follows exactly this pattern.

### MatchingEngineStore (search index)

Writes vectors directly to Vertex AI Matching Engine using stream updates.

```go
store, err := rag.NewMatchingEngineStore(ctx, "us-central1",
    "projects/my-project/locations/us-central1/indexes/12345")
```

### GCSStore (durable persistence)

Writes full records (source text, metadata, vector) to GCS as JSON files.

```go
gcsStore, err := rag.NewGCSStore(ctx, "my-bucket", "embeddings/build-failures")
```

Records are written to `gs://{bucket}/{prefix}/{id}.json`:

```json
{
  "id": "fix-123",
  "source_text": "the original text that was embedded",
  "metadata": {"pr_url": "...", "repo": "..."},
  "vector": [0.123, 0.456, ...],
  "stored_at": "2026-04-01T12:00:00Z"
}
```

### MultiStore (dual-write)

Fans out writes to multiple stores for both durability and search:

```go
store := rag.NewMultiStore(gcsStore, matchingEngineStore)
```

All stores are attempted regardless of individual failures. Errors are collected
and returned via `errors.Join`, so partial failures are visible to the caller.

The `Client` automatically creates a `MultiStore` when `GCSBucket` is set in
`ClientConfig`.

## Metadata

Metadata is stored as `map[string]string` alongside each vector. It is returned
in search results and can be used to provide context to the LLM.

### Reserved keys

| Key | Purpose |
|-----|---------|
| `_source_text` (`MetadataKeySourceText`) | Original text used to generate the embedding. Automatically set by `EmbedAndStore`. Used for re-embedding with newer models. |
| `_stored_at` (`MetadataKeyStoredAt`) | RFC3339 timestamp when the record was stored. Automatically set by `MatchingEngineStore`. |

### Example metadata for build failures

```go
metadata := map[string]string{
    "error_message": "cannot find module providing package foo/bar",
    "error_type":    "DEPENDENCY_VERSION",
    "build_system":  "melange/go",
    "root_cause":    "Version mismatch between...",
    "org":           "chainguard-dev",
    "repo":          "enterprise-packages",
    "pr_number":     "38911",
    "pr_url":        "https://github.com/chainguard-dev/enterprise-packages/pull/38911",
}
```

## Agent Integration

RAG integrates into driftlessAF agents in two ways: as a **composable
ToolProvider** (direct, in-process) or via an **MCP server** (remote, over HTTP).
Both expose the same `search` and `list_corpora` tools.

```
┌─────────────────────────────────────────────────────────┐
│                  driftlessAF Agent                       │
│                                                         │
│  ToolProvider Stack                                     │
│  ┌───────────────────────────────────────────────────┐  │
│  │  RAGToolsProvider  (search, list_corpora)         │  │
│  │  SkillsToolsProvider (read_skill, ...)            │  │
│  │  MCPProvider  (remote MCP server tools)           │  │
│  │  EmptyToolsProvider  (base)                       │  │
│  └───────────────────────────────────────────────────┘  │
│         │                            │                  │
│    Direct (in-process)        MCP (cross-project)       │
│         │                            │                  │
│         v                            v                  │
│  [RAG Client]              [RAG MCP Server]             │
│         │                            │                  │
│         v                            v                  │
│           [Vertex AI Matching Engine]                   │
└─────────────────────────────────────────────────────────┘
```

### Option A: Direct ToolProvider (in-process)

Use this when the agent runs in the same GCP project as the Matching Engine index.
This is the recommended path for most agents — it avoids network hops, has no
additional auth complexity, and any number of agents and teams within the same
project can share the same index.

RAG follows the standard driftlessAF `ToolProvider` composition pattern
(`Empty -> Wrapper -> Wrapper -> ...`). Create a provider that wraps a base
provider and adds RAG tools:

```go
// RAGTools wraps a base tools type and adds RAG search capabilities.
type RAGTools[T any] struct {
    base     T
    registry *CorpusRegistry
}

type ragToolsProvider[Resp, T any] struct {
    base toolcall.ToolProvider[Resp, T]
}

func NewToolsProvider[Resp, T any](
    base toolcall.ToolProvider[Resp, T],
) toolcall.ToolProvider[Resp, RAGTools[T]] {
    return ragToolsProvider[Resp, T]{base: base}
}

func (p ragToolsProvider[Resp, T]) Tools(ctx context.Context, cb RAGTools[T]) (map[string]toolcall.Tool[Resp], error) {
    tools, err := p.base.Tools(ctx, cb.base)
    if err != nil {
        return nil, err
    }

    // Add search tool
    tools["search"] = toolcall.Tool[Resp]{
        Def: toolcall.Definition{
            Name:        "search",
            Description: "Search a RAG corpus for semantically similar content",
            Parameters: []toolcall.Parameter{
                {Name: "query", Type: "string", Required: true},
                {Name: "corpus", Type: "string", Required: true},
                {Name: "limit", Type: "integer"},
            },
        },
        Handler: func(ctx context.Context, call toolcall.ToolCall,
            trace *agenttrace.Trace[Resp], _ *Resp) map[string]any {
            // ... embed query and search ...
        },
    }

    return tools, nil
}
```

Compose into an agent's tool stack:

```go
// Empty -> Skills -> RAG
provider := ragtool.NewToolsProvider[*Response, skills.Tools[toolcall.EmptyTools]](
    skills.NewToolsProvider[*Response, toolcall.EmptyTools](
        toolcall.NewEmptyToolsProvider[*Response](),
    ),
)

// Pass callbacks at execution time
cb := ragtool.RAGTools[skills.Tools[toolcall.EmptyTools]]{
    base:     skills.NewTools(skillsdata.FS, toolcall.EmptyTools{}),
    registry: myRegistry,
}
response, err := agent.Execute(ctx, request, cb)
```

### Option B: MCP server (cross-project / external clients)

Use this when RAG needs to be accessed from a **different GCP project**, from
**external clients** like Claude Code, or from environments that don't have direct
Vertex AI access. The MCP server runs on Cloud Run with OAuth 2.0 authentication.

```
Claude Code / External Agent
    |
    | (MCP Streamable HTTP + OAuth 2.0)
    v
[RAG MCP Server on Cloud Run]
    |
    | (GCP service account credentials)
    v
[Vertex AI Matching Engine]
```

Agents consume MCP tools via the `mcptool.NewProvider` wrapper, which connects
lazily when `Tools()` is called:

```go
// Empty -> Skills -> MCP (includes remote RAG tools)
provider := mcptool.NewProvider[*Response, skills.Tools[toolcall.EmptyTools]](
    skills.NewToolsProvider[*Response, toolcall.EmptyTools](
        toolcall.NewEmptyToolsProvider[*Response](),
    ),
)

// MCP tools appear as "rag__search", "rag__list_corpora"
cb := mcptool.NewTools(
    skills.NewTools(skillsdata.FS, toolcall.EmptyTools{}),
    mcptool.Config{
        MCPServers: map[string]mcptool.ServerConfig{
            "rag": {Type: "http", URL: "https://rag-mcp-server.../"},
        },
    },
    mcptool.WithAuth(myAuthenticator),
)
```

**Authentication layers:**

1. **MCP clients -> Cloud Run**: Chainguard OIDC token via OAuth 2.0 protected
   resource flow. The server advertises its issuer at
   `/.well-known/oauth-protected-resource`.
2. **Cloud Run -> Matching Engine**: GCP service account credentials (ADC,
   automatic on Cloud Run). Only the service account with `roles/aiplatform.user`
   can query the index.

### When to use which

| Criteria | Direct ToolProvider | MCP Server |
|----------|-------------------|------------|
| Same GCP project | Yes (recommended) | Works but unnecessary |
| Cross-project access | No | Yes |
| External clients (Claude Code) | No | Yes |
| Multiple teams, same project | Yes | Not needed |
| Network latency | None (in-process) | HTTP round trip |
| Auth complexity | GCP ADC only | OIDC + GCP ADC |
| Deployment | Part of agent binary | Separate Cloud Run service |

### Tools exposed

Both integration paths expose the same tools:

| Tool | Parameters | Description |
|------|-----------|-------------|
| `search` | `query` (required), `corpus` (required), `limit` (optional) | Embed query text and search for similar vectors |
| `list_corpora` | none | List available knowledge corpora |

### Building an MCP server

Use `mcptool.FromTools` for static tool sets or `mcptool.NewHandler` with a
`ToolProvider` for dynamic, per-session tools:

```go
handler := mcptool.FromTools("my-rag", tools, authFunc,
    mcptool.WithTokenVerifier(verifier, &auth.RequireBearerTokenOptions{
        ResourceMetadataURL: self + "/.well-known/oauth-protected-resource",
    }),
)
```

### Multi-corpus configuration

The MCP server supports multiple corpora via the `CORPORA_CONFIG` environment
variable (JSON array):

```json
[
  {
    "name": "build-failures",
    "description": "Historical build failure errors and their fix PRs",
    "index_name": "projects/my-project/locations/us-central1/indexes/12345",
    "index_endpoint_id": "67890",
    "deployed_index_id": "my_deployed_index",
    "public_domain_name": "1234.us-central1-5678.vdb.vertexai.goog"
  },
  {
    "name": "advisories",
    "description": "Security advisory embeddings",
    "index_name": "projects/my-project/locations/us-central1/indexes/99999",
    "index_endpoint_id": "11111",
    "deployed_index_id": "advisories_index",
    "public_domain_name": "5678.us-central1-9999.vdb.vertexai.goog"
  }
]
```

## Infrastructure (Terraform)

### Required GCP resources

| Resource | Purpose |
|----------|---------|
| `google_vertex_ai_index` | Vector index (3072 dims, cosine distance, stream update) |
| `google_vertex_ai_index_endpoint` | Public or private endpoint for querying |
| `google_vertex_ai_index_endpoint_deployed_index` | Deployed index on the endpoint |
| `google_storage_bucket` | (Optional) GCS bucket for durable embedding storage |
| `google_service_account` | Service account with `roles/aiplatform.user` |
| Cloud Run service | MCP server via `regional-go-service` module |

### Example Terraform for a new corpus

```hcl
resource "google_vertex_ai_index" "my_corpus" {
  project      = local.project_id
  region       = "us-central1"
  display_name = "my-corpus"

  metadata {
    config {
      dimensions                  = 3072  # gemini-embedding-001 at max dims
      approximate_neighbors_count = 150
      distance_measure_type       = "COSINE_DISTANCE"
      feature_norm_type           = "UNIT_L2_NORM"

      algorithm_config {
        tree_ah_config {
          leaf_node_embedding_count    = 1000
          leaf_nodes_to_search_percent = 10
        }
      }
    }
  }

  index_update_method = "STREAM_UPDATE"
}

resource "google_vertex_ai_index_endpoint" "my_corpus" {
  project      = local.project_id
  region       = "us-central1"
  display_name = "my-corpus-endpoint"
}

resource "google_vertex_ai_index_endpoint_deployed_index" "my_corpus" {
  index_endpoint    = google_vertex_ai_index_endpoint.my_corpus.id
  index             = google_vertex_ai_index.my_corpus.id
  deployed_index_id = "my_corpus_v1"

  dedicated_resources {
    machine_spec {
      machine_type = "e2-standard-16"  # Required for 3072-dim vectors
    }
    min_replica_count = 1
    max_replica_count = 2
  }
}
```

### Machine sizing

3072-dimensional vectors require `SHARD_SIZE_MEDIUM` or larger, which needs at
least `e2-standard-16`. Smaller machines will be rejected by the API.

## Data Ingestion

### Bulk ingestion pattern

For ingesting large amounts of existing data, use concurrent workers:

```go
embedder, _ := rag.NewEmbedder(ctx, project, location, "gemini-embedding-001")
store := rag.NewMultiStore(gcsStore, meStore)

work := make(chan Record, workers)
var wg sync.WaitGroup

for range workers {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for r := range work {
            vector, _ := embedder.Embed(ctx, r.Text, rag.TaskTypeRetrievalDocument)
            store.Upsert(ctx, r.ID, vector, r.Metadata)
        }
    }()
}
```

### Considerations

- **Rate limiting**: Vertex AI embedding APIs have per-project quotas. Start with
  a small worker count (e.g., 10) and increase if quota allows.
- **Idempotency**: `Upsert` is idempotent by ID. Re-running ingestion with the
  same IDs updates existing records.
- **Source text preservation**: Always include source text in metadata
  (`MetadataKeySourceText`) so records can be re-embedded when upgrading models.

## Use Cases

### Build failure analysis

Store historical build failures with their error messages, root causes, and fix
PRs. When a new build fails, search for similar past failures to suggest fixes.

### Security advisories

Embed advisory descriptions and match incoming vulnerability reports against known
advisories for faster triage.

### Documentation search

Index internal documentation, runbooks, and incident reports. Agents can retrieve
relevant procedures when handling incidents.

### Code review assistance

Store past review comments and their resolutions. When reviewing similar code
patterns, surface relevant past feedback.

### Incident response

Embed past incident timelines and resolutions. When a new incident occurs, find
similar past incidents to inform the response.

## Extending the System

### Adding a new embedding provider

Implement a wrapper that matches the `Embed(ctx, text, taskType) ([]float32, error)`
signature. The `Embedder` is currently a concrete struct; for alternative providers,
construct the `Client` manually:

```go
client := &rag.Client{} // Use accessor methods after construction
// Or build components individually and wire them together
```

### Adding a new vector store

Implement the `Store` interface:

```go
type Store interface {
    Upsert(ctx context.Context, id string, vector []float32, metadata map[string]string) error
    Close() error
}
```

This could back onto pgvector, Pinecone, Weaviate, or any vector database.

### Adding a new retriever

Implement the `Retriever` interface:

```go
type Retriever interface {
    Search(ctx context.Context, query []float32, opts SearchOptions) ([]Result, error)
    Close() error
}
```

### Adding a new corpus

1. Create the Vertex AI index and endpoint (Terraform)
2. Add the corpus to `CORPORA_CONFIG` on the MCP server
3. Ingest data using `EmbedAndStore` or the ingest tool
4. The corpus is immediately searchable via the `search` MCP tool

## File Layout

```
public/go-driftlessaf/agents/rag/
  doc.go          # Package documentation
  types.go        # TaskType, SearchOptions, Result, constants
  embedder.go     # Vertex AI embedding generation
  store.go        # Store interface + MatchingEngineStore
  gcsstore.go     # GCSStore + MultiStore
  retriever.go    # Retriever interface + MatchingEngineRetriever
  client.go       # Client convenience wrapper
```
