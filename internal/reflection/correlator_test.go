package reflection

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCorrelator_Correlate_InsufficientPatterns(t *testing.T) {
	correlator := NewCorrelator()

	t.Run("empty patterns", func(t *testing.T) {
		correlations, err := correlator.Correlate([]Pattern{}, CorrelateOptions{})
		assert.NoError(t, err)
		assert.Empty(t, correlations)
	})

	t.Run("single pattern", func(t *testing.T) {
		correlations, err := correlator.Correlate([]Pattern{{ID: "p1"}}, CorrelateOptions{})
		assert.NoError(t, err)
		assert.Empty(t, correlations)
	})
}

func TestCorrelator_Correlate_SimilarPatterns(t *testing.T) {
	correlator := NewCorrelator()

	patterns := []Pattern{
		{
			ID:       "p1",
			Category: PatternSuccess,
			Tags:     []string{"golang", "api", "testing"},
			Domains:  []string{"backend"},
		},
		{
			ID:       "p2",
			Category: PatternSuccess,
			Tags:     []string{"golang", "api", "database"},
			Domains:  []string{"backend"},
		},
	}

	correlations, err := correlator.Correlate(patterns, CorrelateOptions{
		Types:       []CorrelationType{CorrelationSimilar},
		MinStrength: 0.3,
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, correlations)

	// Should find similar correlation based on shared tags
	hasSimilar := false
	for _, c := range correlations {
		if c.Type == CorrelationSimilar {
			hasSimilar = true
			assert.Greater(t, c.Strength, 0.0)
		}
	}
	assert.True(t, hasSimilar)
}

func TestCorrelator_Correlate_OppositePatterns(t *testing.T) {
	correlator := NewCorrelator()

	patterns := []Pattern{
		{
			ID:       "success",
			Category: PatternSuccess,
			Tags:     []string{"caching", "redis"},
			Domains:  []string{"performance"},
		},
		{
			ID:       "failure",
			Category: PatternFailure,
			Tags:     []string{"caching", "redis"},
			Domains:  []string{"performance"},
		},
	}

	correlations, err := correlator.Correlate(patterns, CorrelateOptions{
		Types:       []CorrelationType{CorrelationOpposite},
		MinStrength: 0.3,
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, correlations)

	hasOpposite := false
	for _, c := range correlations {
		if c.Type == CorrelationOpposite {
			hasOpposite = true
		}
	}
	assert.True(t, hasOpposite)
}

func TestCorrelator_Correlate_CoOccurringPatterns(t *testing.T) {
	correlator := NewCorrelator()

	patterns := []Pattern{
		{
			ID:        "p1",
			Category:  PatternRecurring,
			MemoryIDs: []string{"m1", "m2", "m3"},
		},
		{
			ID:        "p2",
			Category:  PatternRecurring,
			MemoryIDs: []string{"m2", "m3", "m4"},
		},
	}

	correlations, err := correlator.Correlate(patterns, CorrelateOptions{
		Types:       []CorrelationType{CorrelationCoOccurs},
		MinStrength: 0.3,
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, correlations)

	hasCoOccurs := false
	for _, c := range correlations {
		if c.Type == CorrelationCoOccurs {
			hasCoOccurs = true
			// Should have 2/4 = 0.5 strength (2 shared out of 4 unique)
			assert.Greater(t, c.Strength, 0.0)
		}
	}
	assert.True(t, hasCoOccurs)
}

func TestCorrelator_Correlate_SequentialPatterns(t *testing.T) {
	correlator := NewCorrelator()
	now := time.Now()

	patterns := []Pattern{
		{
			ID:        "early",
			Category:  PatternRecurring,
			FirstSeen: now.Add(-48 * time.Hour),
			LastSeen:  now.Add(-24 * time.Hour),
		},
		{
			ID:        "late",
			Category:  PatternRecurring,
			FirstSeen: now.Add(-12 * time.Hour),
			LastSeen:  now,
		},
	}

	correlations, err := correlator.Correlate(patterns, CorrelateOptions{
		Types:       []CorrelationType{CorrelationSequential},
		MinStrength: 0.3,
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, correlations)

	hasSequential := false
	for _, c := range correlations {
		if c.Type == CorrelationSequential {
			hasSequential = true
		}
	}
	assert.True(t, hasSequential)
}

func TestCorrelator_Correlate_FilterByPatternIDs(t *testing.T) {
	correlator := NewCorrelator()

	patterns := []Pattern{
		{ID: "p1", Tags: []string{"shared"}},
		{ID: "p2", Tags: []string{"shared"}},
		{ID: "p3", Tags: []string{"shared"}},
	}

	correlations, err := correlator.Correlate(patterns, CorrelateOptions{
		PatternIDs:  []string{"p1", "p2"},
		Types:       []CorrelationType{CorrelationSimilar},
		MinStrength: 0.3,
	})

	assert.NoError(t, err)
	// Should only correlate p1 and p2
	for _, c := range correlations {
		assert.NotEqual(t, "p3", c.SourceID)
		assert.NotEqual(t, "p3", c.TargetID)
	}
}

func TestCorrelator_Correlate_MaxCorrelations(t *testing.T) {
	correlator := NewCorrelator()

	// Create many patterns that will correlate
	patterns := make([]Pattern, 10)
	for i := 0; i < 10; i++ {
		patterns[i] = Pattern{
			ID:   string(rune('a' + i)),
			Tags: []string{"shared"},
		}
	}

	correlations, err := correlator.Correlate(patterns, CorrelateOptions{
		Types:           []CorrelationType{CorrelationSimilar},
		MinStrength:     0.1,
		MaxCorrelations: 5,
	})

	assert.NoError(t, err)
	assert.LessOrEqual(t, len(correlations), 5)
}

func TestCountShared(t *testing.T) {
	t.Run("no overlap", func(t *testing.T) {
		count := countShared([]string{"a", "b"}, []string{"c", "d"})
		assert.Equal(t, 0, count)
	})

	t.Run("partial overlap", func(t *testing.T) {
		count := countShared([]string{"a", "b", "c"}, []string{"b", "c", "d"})
		assert.Equal(t, 2, count)
	})

	t.Run("complete overlap", func(t *testing.T) {
		count := countShared([]string{"a", "b"}, []string{"a", "b"})
		assert.Equal(t, 2, count)
	})
}

func TestCountUnique(t *testing.T) {
	t.Run("no overlap", func(t *testing.T) {
		count := countUnique([]string{"a", "b"}, []string{"c", "d"})
		assert.Equal(t, 4, count)
	})

	t.Run("partial overlap", func(t *testing.T) {
		count := countUnique([]string{"a", "b", "c"}, []string{"b", "c", "d"})
		assert.Equal(t, 4, count)
	})

	t.Run("complete overlap", func(t *testing.T) {
		count := countUnique([]string{"a", "b"}, []string{"a", "b"})
		assert.Equal(t, 2, count)
	})
}

func TestCalculateSimilarity(t *testing.T) {
	t.Run("identical patterns", func(t *testing.T) {
		p1 := Pattern{Tags: []string{"a", "b"}, Domains: []string{"x"}}
		p2 := Pattern{Tags: []string{"a", "b"}, Domains: []string{"x"}}
		similarity := calculateSimilarity(p1, p2)
		assert.Equal(t, 1.0, similarity)
	})

	t.Run("no similarity", func(t *testing.T) {
		p1 := Pattern{Tags: []string{"a", "b"}, Domains: []string{"x"}}
		p2 := Pattern{Tags: []string{"c", "d"}, Domains: []string{"y"}}
		similarity := calculateSimilarity(p1, p2)
		assert.Equal(t, 0.0, similarity)
	})

	t.Run("partial similarity", func(t *testing.T) {
		p1 := Pattern{Tags: []string{"a", "b"}, Domains: []string{}}
		p2 := Pattern{Tags: []string{"b", "c"}, Domains: []string{}}
		similarity := calculateSimilarity(p1, p2)
		// 1 shared out of 3 unique = 0.333...
		assert.InDelta(t, 0.33, similarity, 0.01)
	})
}
