package testdata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFixtures validates that all test fixtures are well-formed.
func TestFixtures(t *testing.T) {
	fixtures := AllFixtures()
	require.NotEmpty(t, fixtures, "AllFixtures should return at least one fixture")

	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			// Validate the fixture
			err := ValidateFixture(fixture)
			assert.NoError(t, err, "Fixture %s should be valid", fixture.Name)

			// Additional sanity checks
			assert.NotEmpty(t, fixture.Query, "Query should not be empty")
			assert.GreaterOrEqual(t, len(fixture.Documents), 2, "Should have at least 2 documents")
			assert.Equal(t, len(fixture.Documents), len(fixture.ExpectedRanking),
				"ExpectedRanking length should match document count")

			// Verify all document IDs are unique
			docIDs := make(map[string]bool)
			for _, doc := range fixture.Documents {
				assert.NotEmpty(t, doc.ID, "Document ID should not be empty")
				assert.NotEmpty(t, doc.Content, "Document content should not be empty")
				assert.False(t, docIDs[doc.ID], "Document IDs should be unique: %s", doc.ID)
				docIDs[doc.ID] = true
			}

			// Verify expected ranking references valid document IDs
			for _, id := range fixture.ExpectedRanking {
				assert.True(t, docIDs[id], "ExpectedRanking should reference valid document ID: %s", id)
			}

			// Verify score ranges are valid
			for docID, scoreRange := range fixture.ExpectedScoreRanges {
				assert.True(t, docIDs[docID], "ScoreRange should reference valid document ID: %s", docID)
				assert.GreaterOrEqual(t, scoreRange.Min, float32(0.0), "Score min should be >= 0")
				assert.LessOrEqual(t, scoreRange.Min, float32(1.0), "Score min should be <= 1")
				assert.GreaterOrEqual(t, scoreRange.Max, float32(0.0), "Score max should be >= 0")
				assert.LessOrEqual(t, scoreRange.Max, float32(1.0), "Score max should be <= 1")
				assert.LessOrEqual(t, scoreRange.Min, scoreRange.Max, "Score min should be <= max")
			}
		})
	}
}

// TestGetFixtureByName validates the GetFixtureByName function.
func TestGetFixtureByName(t *testing.T) {
	t.Run("existing fixture", func(t *testing.T) {
		fixture := GetFixtureByName("high_similarity_pair")
		require.NotNil(t, fixture, "Should find existing fixture")
		assert.Equal(t, "high_similarity_pair", fixture.Name)
	})

	t.Run("non-existent fixture", func(t *testing.T) {
		fixture := GetFixtureByName("non_existent_fixture")
		assert.Nil(t, fixture, "Should return nil for non-existent fixture")
	})
}

// TestValidateFixture_InvalidCases validates that ValidateFixture catches invalid fixtures.
func TestValidateFixture_InvalidCases(t *testing.T) {
	t.Run("empty name", func(t *testing.T) {
		fixture := SemanticTestCase{
			Query: "test query",
			Documents: []TestDocument{
				{ID: "doc1", Content: "content1"},
				{ID: "doc2", Content: "content2"},
			},
			ExpectedRanking: []string{"doc1", "doc2"},
		}
		err := ValidateFixture(fixture)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("empty query", func(t *testing.T) {
		fixture := SemanticTestCase{
			Name: "test",
			Documents: []TestDocument{
				{ID: "doc1", Content: "content1"},
				{ID: "doc2", Content: "content2"},
			},
			ExpectedRanking: []string{"doc1", "doc2"},
		}
		err := ValidateFixture(fixture)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "query cannot be empty")
	})

	t.Run("too few documents", func(t *testing.T) {
		fixture := SemanticTestCase{
			Name:  "test",
			Query: "test query",
			Documents: []TestDocument{
				{ID: "doc1", Content: "content1"},
			},
			ExpectedRanking: []string{"doc1"},
		}
		err := ValidateFixture(fixture)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must have at least 2 documents")
	})

	t.Run("ranking mismatch", func(t *testing.T) {
		fixture := SemanticTestCase{
			Name:  "test",
			Query: "test query",
			Documents: []TestDocument{
				{ID: "doc1", Content: "content1"},
				{ID: "doc2", Content: "content2"},
			},
			ExpectedRanking: []string{"doc1"}, // Missing doc2
		}
		err := ValidateFixture(fixture)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected ranking must match document count")
	})

	t.Run("empty document ID", func(t *testing.T) {
		fixture := SemanticTestCase{
			Name:  "test",
			Query: "test query",
			Documents: []TestDocument{
				{ID: "", Content: "content1"},
				{ID: "doc2", Content: "content2"},
			},
			ExpectedRanking: []string{"", "doc2"},
		}
		err := ValidateFixture(fixture)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "document ID cannot be empty")
	})

	t.Run("empty document content", func(t *testing.T) {
		fixture := SemanticTestCase{
			Name:  "test",
			Query: "test query",
			Documents: []TestDocument{
				{ID: "doc1", Content: ""},
				{ID: "doc2", Content: "content2"},
			},
			ExpectedRanking: []string{"doc1", "doc2"},
		}
		err := ValidateFixture(fixture)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "document content cannot be empty")
	})

	t.Run("duplicate document IDs", func(t *testing.T) {
		fixture := SemanticTestCase{
			Name:  "test",
			Query: "test query",
			Documents: []TestDocument{
				{ID: "doc1", Content: "content1"},
				{ID: "doc1", Content: "content2"}, // Duplicate ID
			},
			ExpectedRanking: []string{"doc1", "doc1"},
		}
		err := ValidateFixture(fixture)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate document ID")
	})

	t.Run("invalid ranking ID", func(t *testing.T) {
		fixture := SemanticTestCase{
			Name:  "test",
			Query: "test query",
			Documents: []TestDocument{
				{ID: "doc1", Content: "content1"},
				{ID: "doc2", Content: "content2"},
			},
			ExpectedRanking: []string{"doc1", "doc3"}, // doc3 doesn't exist
		}
		err := ValidateFixture(fixture)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected ranking references non-existent document ID")
	})

	t.Run("invalid score range document ID", func(t *testing.T) {
		fixture := SemanticTestCase{
			Name:  "test",
			Query: "test query",
			Documents: []TestDocument{
				{ID: "doc1", Content: "content1"},
				{ID: "doc2", Content: "content2"},
			},
			ExpectedRanking: []string{"doc1", "doc2"},
			ExpectedScoreRanges: map[string]ScoreRange{
				"doc3": {Min: 0.5, Max: 1.0}, // doc3 doesn't exist
			},
		}
		err := ValidateFixture(fixture)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "score range references non-existent document ID")
	})

	t.Run("score range min too low", func(t *testing.T) {
		fixture := SemanticTestCase{
			Name:  "test",
			Query: "test query",
			Documents: []TestDocument{
				{ID: "doc1", Content: "content1"},
				{ID: "doc2", Content: "content2"},
			},
			ExpectedRanking: []string{"doc1", "doc2"},
			ExpectedScoreRanges: map[string]ScoreRange{
				"doc1": {Min: -0.1, Max: 1.0}, // Min too low
			},
		}
		err := ValidateFixture(fixture)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "score range min must be in [0, 1]")
	})

	t.Run("score range max too high", func(t *testing.T) {
		fixture := SemanticTestCase{
			Name:  "test",
			Query: "test query",
			Documents: []TestDocument{
				{ID: "doc1", Content: "content1"},
				{ID: "doc2", Content: "content2"},
			},
			ExpectedRanking: []string{"doc1", "doc2"},
			ExpectedScoreRanges: map[string]ScoreRange{
				"doc1": {Min: 0.5, Max: 1.5}, // Max too high
			},
		}
		err := ValidateFixture(fixture)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "score range max must be in [0, 1]")
	})

	t.Run("score range min exceeds max", func(t *testing.T) {
		fixture := SemanticTestCase{
			Name:  "test",
			Query: "test query",
			Documents: []TestDocument{
				{ID: "doc1", Content: "content1"},
				{ID: "doc2", Content: "content2"},
			},
			ExpectedRanking: []string{"doc1", "doc2"},
			ExpectedScoreRanges: map[string]ScoreRange{
				"doc1": {Min: 0.8, Max: 0.5}, // Min > Max
			},
		}
		err := ValidateFixture(fixture)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "score range min cannot exceed max")
	})
}

// TestAllFixturesCount verifies we have the expected number of fixtures.
func TestAllFixturesCount(t *testing.T) {
	fixtures := AllFixtures()
	// We defined 5 fixtures: HighSimilarityPair, LowSimilarityPair, SynonymHandling,
	// MultiTopicDocuments, and GradualRelevanceDecay
	assert.Equal(t, 5, len(fixtures), "Should have exactly 5 fixtures")
}

// TestHighSimilarityPair validates the high similarity pair fixture structure.
func TestHighSimilarityPair(t *testing.T) {
	fixture := HighSimilarityPair()
	assert.Equal(t, "high_similarity_pair", fixture.Name)
	assert.Contains(t, fixture.Query, "Go")
	assert.Len(t, fixture.Documents, 3)
	assert.Equal(t, []string{"doc1", "doc2", "doc3"}, fixture.ExpectedRanking)

	// Verify score ranges are set for high similarity documents
	assert.GreaterOrEqual(t, fixture.ExpectedScoreRanges["doc1"].Min, float32(0.7))
	assert.GreaterOrEqual(t, fixture.ExpectedScoreRanges["doc2"].Min, float32(0.7))
}

// TestLowSimilarityPair validates the low similarity pair fixture structure.
func TestLowSimilarityPair(t *testing.T) {
	fixture := LowSimilarityPair()
	assert.Equal(t, "low_similarity_pair", fixture.Name)
	assert.Contains(t, fixture.Query, "Go programming")
	assert.Len(t, fixture.Documents, 3)

	// Verify the dissimilar document (cooking) has low expected score
	assert.LessOrEqual(t, fixture.ExpectedScoreRanges["doc2"].Max, float32(0.3))
}

// TestSynonymHandling validates the synonym handling fixture structure.
func TestSynonymHandling(t *testing.T) {
	fixture := SynonymHandling()
	assert.Equal(t, "synonym_handling", fixture.Name)
	assert.Contains(t, fixture.Query, "tutorial")
	assert.Len(t, fixture.Documents, 4)

	// Verify both tutorial and guide documents have high expected scores
	assert.GreaterOrEqual(t, fixture.ExpectedScoreRanges["doc1"].Min, float32(0.7))
	assert.GreaterOrEqual(t, fixture.ExpectedScoreRanges["doc2"].Min, float32(0.7))
}

// TestMultiTopicDocuments validates the multi-topic documents fixture structure.
func TestMultiTopicDocuments(t *testing.T) {
	fixture := MultiTopicDocuments()
	assert.Equal(t, "multi_topic_documents", fixture.Name)
	assert.Contains(t, fixture.Query, "machine learning")
	assert.Contains(t, fixture.Query, "Python")
	assert.Len(t, fixture.Documents, 4)

	// Verify doc1 (both topics) has highest expected score
	assert.GreaterOrEqual(t, fixture.ExpectedScoreRanges["doc1"].Min, float32(0.7))
}

// TestGradualRelevanceDecay validates the gradual relevance decay fixture structure.
func TestGradualRelevanceDecay(t *testing.T) {
	fixture := GradualRelevanceDecay()
	assert.Equal(t, "gradual_relevance_decay", fixture.Name)
	assert.Len(t, fixture.Documents, 5)
	assert.Len(t, fixture.ExpectedRanking, 5)

	// Verify score ranges decay gradually
	doc1Range := fixture.ExpectedScoreRanges["doc1"]
	doc2Range := fixture.ExpectedScoreRanges["doc2"]
	doc3Range := fixture.ExpectedScoreRanges["doc3"]
	doc4Range := fixture.ExpectedScoreRanges["doc4"]
	doc5Range := fixture.ExpectedScoreRanges["doc5"]

	// Each subsequent document should have a lower minimum score
	assert.Greater(t, doc1Range.Min, doc2Range.Min)
	assert.Greater(t, doc2Range.Min, doc3Range.Min)
	assert.Greater(t, doc3Range.Min, doc4Range.Min)
	assert.Greater(t, doc4Range.Min, doc5Range.Min)
}
