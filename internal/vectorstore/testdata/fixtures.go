// Package testdata provides test fixtures for semantic similarity testing.
//
// This package defines known query-document pairs with expected relevance
// rankings and score ranges. These fixtures are used across multiple test
// suites to validate semantic search quality and detect regressions.
package testdata

// SemanticTestCase represents a test case for semantic similarity testing.
// It includes a query, a set of documents, and expected search behavior.
type SemanticTestCase struct {
	// Name is a descriptive identifier for the test case
	Name string

	// Query is the search query string
	Query string

	// Documents are the candidate documents to search against
	Documents []TestDocument

	// ExpectedRanking is the expected order of document IDs by relevance.
	// Index 0 is the most relevant document, index n-1 is the least relevant.
	// Document IDs should match the IDs in Documents slice.
	ExpectedRanking []string

	// ExpectedScoreRanges defines expected similarity score ranges for documents.
	// Key is document ID, value is the expected score range [min, max].
	ExpectedScoreRanges map[string]ScoreRange

	// Description provides additional context about what this test case validates
	Description string
}

// TestDocument represents a document in a test case.
type TestDocument struct {
	// ID is a unique identifier for the document
	ID string

	// Content is the document text content
	Content string

	// Metadata are optional key-value pairs associated with the document
	Metadata map[string]string
}

// ScoreRange defines the expected range of similarity scores.
type ScoreRange struct {
	// Min is the minimum expected similarity score (inclusive)
	Min float32

	// Max is the maximum expected similarity score (inclusive)
	Max float32
}

// AllFixtures returns all available test fixtures.
// This is the primary API for accessing test cases.
func AllFixtures() []SemanticTestCase {
	return []SemanticTestCase{
		HighSimilarityPair(),
		LowSimilarityPair(),
		SynonymHandling(),
		MultiTopicDocuments(),
		GradualRelevanceDecay(),
	}
}

// HighSimilarityPair tests that very similar documents (e.g., Go vs Golang)
// receive high similarity scores (>0.7).
func HighSimilarityPair() SemanticTestCase {
	return SemanticTestCase{
		Name:  "high_similarity_pair",
		Query: "Go programming language tutorial",
		Documents: []TestDocument{
			{
				ID:      "doc1",
				Content: "Go programming language tutorial for beginners",
				Metadata: map[string]string{
					"category": "programming",
					"language": "go",
				},
			},
			{
				ID:      "doc2",
				Content: "Golang programming guide and best practices",
				Metadata: map[string]string{
					"category": "programming",
					"language": "golang",
				},
			},
			{
				ID:      "doc3",
				Content: "Python machine learning tutorial with examples",
				Metadata: map[string]string{
					"category": "programming",
					"language": "python",
				},
			},
		},
		ExpectedRanking: []string{"doc1", "doc2", "doc3"},
		ExpectedScoreRanges: map[string]ScoreRange{
			"doc1": {Min: 0.7, Max: 1.0},  // Very high similarity
			"doc2": {Min: 0.7, Max: 1.0},  // Very high similarity (Go = Golang)
			"doc3": {Min: 0.0, Max: 0.6},  // Different language
		},
		Description: "Validates that semantically similar documents (Go/Golang) receive high similarity scores",
	}
}

// LowSimilarityPair tests that dissimilar documents (e.g., Go programming vs Cooking)
// receive low similarity scores (<0.5).
func LowSimilarityPair() SemanticTestCase {
	return SemanticTestCase{
		Name:  "low_similarity_pair",
		Query: "Go programming language concurrency",
		Documents: []TestDocument{
			{
				ID:      "doc1",
				Content: "Go programming language concurrency patterns with goroutines and channels",
				Metadata: map[string]string{
					"category": "programming",
				},
			},
			{
				ID:      "doc2",
				Content: "Italian cooking recipes with fresh ingredients and herbs",
				Metadata: map[string]string{
					"category": "cooking",
				},
			},
			{
				ID:      "doc3",
				Content: "Advanced Go concurrency: context, select, and synchronization primitives",
				Metadata: map[string]string{
					"category": "programming",
				},
			},
		},
		ExpectedRanking: []string{"doc1", "doc3", "doc2"},
		ExpectedScoreRanges: map[string]ScoreRange{
			"doc1": {Min: 0.7, Max: 1.0},  // High similarity
			"doc3": {Min: 0.6, Max: 1.0},  // High similarity (concurrency-related)
			"doc2": {Min: 0.0, Max: 0.3},  // Very low similarity (different domain)
		},
		Description: "Validates that dissimilar documents from different domains receive low similarity scores",
	}
}

// SynonymHandling tests that synonyms and related terms (e.g., tutorial/guide)
// are recognized as semantically similar.
func SynonymHandling() SemanticTestCase {
	return SemanticTestCase{
		Name:  "synonym_handling",
		Query: "database tutorial for beginners",
		Documents: []TestDocument{
			{
				ID:      "doc1",
				Content: "Comprehensive database tutorial for beginners: SQL, tables, and queries",
				Metadata: map[string]string{
					"type": "tutorial",
				},
			},
			{
				ID:      "doc2",
				Content: "Database guide for beginners: introduction to relational databases",
				Metadata: map[string]string{
					"type": "guide",
				},
			},
			{
				ID:      "doc3",
				Content: "Advanced database optimization techniques for production systems",
				Metadata: map[string]string{
					"type": "advanced",
				},
			},
			{
				ID:      "doc4",
				Content: "Web development framework comparison: React, Vue, and Angular",
				Metadata: map[string]string{
					"type": "comparison",
				},
			},
		},
		ExpectedRanking: []string{"doc1", "doc2", "doc3", "doc4"},
		ExpectedScoreRanges: map[string]ScoreRange{
			"doc1": {Min: 0.7, Max: 1.0},  // Exact match
			"doc2": {Min: 0.7, Max: 1.0},  // Tutorial/guide synonym
			"doc3": {Min: 0.4, Max: 0.7},  // Same topic, different level
			"doc4": {Min: 0.0, Max: 0.4},  // Different topic
		},
		Description: "Validates that synonyms (tutorial/guide) are recognized as semantically similar",
	}
}

// MultiTopicDocuments tests ranking when documents contain multiple topics
// and partial matches.
func MultiTopicDocuments() SemanticTestCase {
	return SemanticTestCase{
		Name:  "multi_topic_documents",
		Query: "machine learning with Python",
		Documents: []TestDocument{
			{
				ID:      "doc1",
				Content: "Machine learning with Python: scikit-learn, TensorFlow, and PyTorch tutorials",
				Metadata: map[string]string{
					"topics": "ml,python",
				},
			},
			{
				ID:      "doc2",
				Content: "Python programming basics: variables, loops, and functions",
				Metadata: map[string]string{
					"topics": "python,basics",
				},
			},
			{
				ID:      "doc3",
				Content: "Machine learning fundamentals: supervised learning, neural networks, and deep learning",
				Metadata: map[string]string{
					"topics": "ml,theory",
				},
			},
			{
				ID:      "doc4",
				Content: "Java enterprise application development with Spring Boot",
				Metadata: map[string]string{
					"topics": "java,enterprise",
				},
			},
		},
		ExpectedRanking: []string{"doc1", "doc3", "doc2", "doc4"},
		ExpectedScoreRanges: map[string]ScoreRange{
			"doc1": {Min: 0.7, Max: 1.0},  // Both ML and Python
			"doc3": {Min: 0.5, Max: 0.8},  // ML but not Python
			"doc2": {Min: 0.4, Max: 0.7},  // Python but not ML
			"doc4": {Min: 0.0, Max: 0.4},  // Neither ML nor Python
		},
		Description: "Validates correct ranking when documents contain multiple topics with partial query matches",
	}
}

// GradualRelevanceDecay tests that relevance scores decay gradually
// as documents become less relevant to the query.
func GradualRelevanceDecay() SemanticTestCase {
	return SemanticTestCase{
		Name:  "gradual_relevance_decay",
		Query: "REST API authentication with JWT tokens",
		Documents: []TestDocument{
			{
				ID:      "doc1",
				Content: "REST API authentication using JWT tokens: implementation guide",
				Metadata: map[string]string{
					"relevance": "exact",
				},
			},
			{
				ID:      "doc2",
				Content: "API authentication methods: JWT, OAuth2, and API keys",
				Metadata: map[string]string{
					"relevance": "high",
				},
			},
			{
				ID:      "doc3",
				Content: "Building RESTful APIs with proper security practices",
				Metadata: map[string]string{
					"relevance": "medium",
				},
			},
			{
				ID:      "doc4",
				Content: "Web security basics: HTTPS, CORS, and authentication overview",
				Metadata: map[string]string{
					"relevance": "low",
				},
			},
			{
				ID:      "doc5",
				Content: "Database indexing strategies for query performance optimization",
				Metadata: map[string]string{
					"relevance": "none",
				},
			},
		},
		ExpectedRanking: []string{"doc1", "doc2", "doc3", "doc4", "doc5"},
		ExpectedScoreRanges: map[string]ScoreRange{
			"doc1": {Min: 0.8, Max: 1.0},  // Exact match
			"doc2": {Min: 0.6, Max: 0.9},  // High relevance
			"doc3": {Min: 0.4, Max: 0.7},  // Medium relevance
			"doc4": {Min: 0.2, Max: 0.5},  // Low relevance
			"doc5": {Min: 0.0, Max: 0.3},  // No relevance
		},
		Description: "Validates that similarity scores decay gradually as document relevance decreases",
	}
}

// GetFixtureByName returns a specific fixture by name.
// Returns nil if the fixture is not found.
func GetFixtureByName(name string) *SemanticTestCase {
	for _, fixture := range AllFixtures() {
		if fixture.Name == name {
			return &fixture
		}
	}
	return nil
}

// ValidateFixture checks that a test case is well-formed.
// Returns an error if the fixture is invalid.
func ValidateFixture(tc SemanticTestCase) error {
	if tc.Name == "" {
		return newValidationError("name cannot be empty")
	}
	if tc.Query == "" {
		return newValidationError("query cannot be empty")
	}
	if len(tc.Documents) < 2 {
		return newValidationError("must have at least 2 documents")
	}
	if len(tc.ExpectedRanking) != len(tc.Documents) {
		return newValidationError("expected ranking must match document count")
	}

	// Validate document IDs are unique
	docIDs := make(map[string]bool)
	for _, doc := range tc.Documents {
		if doc.ID == "" {
			return newValidationError("document ID cannot be empty")
		}
		if doc.Content == "" {
			return newValidationError("document content cannot be empty")
		}
		if docIDs[doc.ID] {
			return newValidationError("duplicate document ID: " + doc.ID)
		}
		docIDs[doc.ID] = true
	}

	// Validate expected ranking references valid document IDs
	for _, id := range tc.ExpectedRanking {
		if !docIDs[id] {
			return newValidationError("expected ranking references non-existent document ID: " + id)
		}
	}

	// Validate score ranges
	for docID, scoreRange := range tc.ExpectedScoreRanges {
		if !docIDs[docID] {
			return newValidationError("score range references non-existent document ID: " + docID)
		}
		if scoreRange.Min < 0 || scoreRange.Min > 1 {
			return newValidationError("score range min must be in [0, 1]")
		}
		if scoreRange.Max < 0 || scoreRange.Max > 1 {
			return newValidationError("score range max must be in [0, 1]")
		}
		if scoreRange.Min > scoreRange.Max {
			return newValidationError("score range min cannot exceed max")
		}
	}

	return nil
}

// validationError is an error type for fixture validation failures.
type validationError struct {
	msg string
}

func newValidationError(msg string) *validationError {
	return &validationError{msg: msg}
}

func (e *validationError) Error() string {
	return "fixture validation error: " + e.msg
}
