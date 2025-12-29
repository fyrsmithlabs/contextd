package vectorstore

import "time"

// CompressionLevel represents the compression state of content.
type CompressionLevel string

const (
	// CompressionLevelNone represents uncompressed original content.
	CompressionLevelNone CompressionLevel = "none"
	// CompressionLevelFolded represents context-folded content (partial compression).
	CompressionLevelFolded CompressionLevel = "folded"
	// CompressionLevelSummary represents summarized content (high compression).
	CompressionLevelSummary CompressionLevel = "summary"
)

// CompressionMetadata tracks compression state and metrics.
type CompressionMetadata struct {
	Level            CompressionLevel `json:"compression_level"`               // Current compression level
	Algorithm        string           `json:"compression_algorithm,omitempty"` // Compression algorithm used
	OriginalSize     int              `json:"original_size"`                   // Original content size (tokens/chars)
	CompressedSize   int              `json:"compressed_size"`                 // Compressed content size
	CompressionRatio float64          `json:"compression_ratio"`               // Compression ratio (original/compressed)
	CompressedAt     *time.Time       `json:"compressed_at,omitempty"`         // When compression was applied
}

// Document represents a document to be stored in the vector store.
type Document struct {
	// ID is the unique identifier for the document
	ID string

	// Content is the text content of the document
	Content string

	// Metadata contains additional key-value pairs for filtering
	// Common fields: owner, project, file, branch, timestamp
	Metadata map[string]interface{}

	// Collection is the target collection name for this document.
	// If empty, uses the service's default collection.
	//
	// Collection naming convention:
	//   - Organization: org_{type} (e.g., org_memories)
	//   - Team: {team}_{type} (e.g., platform_memories)
	//   - Project: {team}_{project}_{type} (e.g., platform_contextd_memories)
	Collection string
}

// SearchResult represents a search result from the vector store.
type SearchResult struct {
	// ID is the document identifier
	ID string

	// Content is the document text content
	Content string

	// Score is the similarity score (higher = more similar)
	Score float32

	// Metadata contains the document metadata
	Metadata map[string]interface{}
}
