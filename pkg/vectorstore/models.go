package vectorstore

// Document represents a document to be stored in the vector store.
type Document struct {
	// ID is the unique identifier for the document
	ID string

	// Content is the text content of the document
	Content string

	// Metadata contains additional key-value pairs for filtering
	// Common fields: owner, project, file, branch, timestamp
	Metadata map[string]interface{}
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
