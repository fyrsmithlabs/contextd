// Package conversation provides indexing and search capabilities for Claude Code
// conversation files (JSONL format).
//
// The package supports:
//   - Parsing Claude Code conversation files from ~/.claude/projects/
//   - Extracting file references and git commit metadata from messages
//   - Indexing conversations into a vector store for semantic search
//   - Searching indexed conversations with filters for type, tags, files, and domains
//
// # Architecture
//
// The main components are:
//   - Parser: Reads JSONL conversation files and extracts messages
//   - Extractor: Extracts file references and commit metadata from messages
//   - Service: Coordinates indexing and search operations
//
// # Usage
//
// Create a service with a vector store and optional secret scrubber:
//
//	svc := conversation.NewService(
//	    vectorStore,
//	    secretScrubber,
//	    logger,
//	    conversation.ServiceConfig{
//	        ConversationsPath: "/path/to/conversations",
//	    },
//	)
//
// Index conversations for a project:
//
//	result, err := svc.Index(ctx, conversation.IndexOptions{
//	    ProjectPath: "/path/to/project",
//	    TenantID:    "my-tenant",
//	})
//
// Search indexed conversations:
//
//	result, err := svc.Search(ctx, conversation.SearchOptions{
//	    Query:       "authentication flow",
//	    ProjectPath: "/path/to/project",
//	    TenantID:    "my-tenant",
//	    Limit:       10,
//	})
//
// # Multi-Tenancy
//
// The service uses payload-based tenant isolation. All indexed documents are
// tagged with tenant metadata and queries are automatically filtered by tenant.
// Collection names are sanitized to ensure safe storage, with SHA-256 hash
// fallback for non-ASCII tenant/project names.
//
// # Secret Scrubbing
//
// If a Scrubber is provided, all message content is scrubbed before indexing
// and before returning search results. This prevents accidental storage or
// exposure of secrets like API keys or tokens.
package conversation
