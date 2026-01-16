// Package vectorstore provides vector storage abstraction with multi-tenant isolation.
//
// The package offers a unified interface for vector storage operations with multiple
// provider implementations (chromem embedded, Qdrant external). It enables semantic
// search over embeddings while enforcing strict tenant isolation for multi-tenant
// deployments.
//
// # Security
//
// The package implements defense-in-depth security with fail-closed behavior:
//   - Multi-tenant isolation via payload filtering (default) or filesystem separation
//   - Mandatory tenant context for all operations (ErrMissingTenant if missing)
//   - Filter injection prevention (rejects user-provided tenant_id/team_id/project_id)
//   - Metadata poisoning protection (tenant fields always overwritten from context)
//   - Collection name validation (prevents path traversal)
//   - Query length limits (10,000 chars max)
//   - Result limits (capped at collection size or 10,000)
//
// # Usage
//
// Basic usage with PayloadIsolation (default):
//
//	import "github.com/fyrsmithlabs/contextd/internal/vectorstore"
//
//	// Configure store (PayloadIsolation is default)
//	config := vectorstore.ChromemConfig{
//	    Path:              "/data/vectorstore",
//	    DefaultCollection: "memories",
//	    VectorSize:        384,
//	    Compress:          true,
//	}
//
//	store, err := vectorstore.NewChromemStore(config, embedder, logger)
//	if err != nil {
//	    return err
//	}
//	defer store.Close()
//
//	// REQUIRED: Add tenant context to all operations
//	ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
//	    TenantID:  "org-123",      // Required
//	    TeamID:    "platform",     // Optional
//	    ProjectID: "contextd",     // Optional
//	})
//
//	// Documents automatically tagged with tenant metadata
//	docs := []vectorstore.Document{
//	    {
//	        ID:      "mem-1",
//	        Content: "User prefers dark mode",
//	        Metadata: map[string]interface{}{"category": "preference"},
//	    },
//	}
//	ids, err := store.AddDocuments(ctx, docs)
//
//	// Searches automatically filtered by tenant
//	results, err := store.Search(ctx, "user preferences", 10)
//
// # Isolation Modes
//
// The package supports three isolation strategies:
//
// PayloadIsolation (default, recommended):
//   - All documents in shared collections with metadata filtering
//   - tenant_id, team_id, project_id stored as document metadata
//   - All queries automatically filtered by tenant context
//   - Fail-closed: missing tenant context returns error, not empty results
//
// FilesystemIsolation (legacy):
//   - Separate database per tenant/team/project
//   - Physical filesystem isolation provides security boundary
//   - Use StoreProvider to manage per-tenant stores
//   - Migration path available to PayloadIsolation
//
// NoIsolation (testing only):
//   - No tenant filtering or validation
//   - WARNING: Provides no security guarantees
//   - Use only in tests where isolation is not relevant
//
// # Provider Selection
//
// The package supports multiple vector store providers:
//
// ChromemStore (default):
//   - Embedded chromem-go storage (no external dependencies)
//   - Local ONNX embeddings via FastEmbed
//   - Perfect for local dev and simple setups
//   - Just works: brew install contextd
//
// QdrantStore (optional):
//   - External Qdrant service via gRPC
//   - Requires external Qdrant server + embedder
//   - Recommended for production, high scale
//
// Provider selection via config:
//
//	vectorstore:
//	  provider: chromem  # "chromem" (default) or "qdrant"
//
// # Collection Naming Convention
//
// Collections follow a hierarchical naming pattern (with PayloadIsolation,
// these are optional - use a single collection with metadata filtering):
//   - Organization: org_{type} (e.g., org_memories)
//   - Team: {team}_{type} (e.g., platform_memories)
//   - Project: {team}_{project}_{type} (e.g., platform_contextd_memories)
//
// # Performance
//
// Current implementation optimizations:
//   - Batch embedding generation for multiple documents
//   - Concurrent search operations across collections
//   - Optional compression for storage efficiency
//   - HNSW index for fast approximate nearest neighbor search
//
// Future optimization opportunities:
//   - Connection pooling for Qdrant gRPC
//   - Result caching with TTL
//   - Adaptive batch sizing based on load
package vectorstore
