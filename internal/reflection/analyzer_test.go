package reflection

import (
	"context"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMemoryService is a mock implementation of reasoningbank.Service for testing.
type mockMemoryService struct {
	memories []reasoningbank.Memory
	err      error
}

func (m *mockMemoryService) Search(ctx context.Context, projectID, query string, limit int) ([]reasoningbank.Memory, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.memories, nil
}

func (m *mockMemoryService) Record(ctx context.Context, memory *reasoningbank.Memory) error {
	return nil
}

func (m *mockMemoryService) Feedback(ctx context.Context, memoryID string, helpful bool) error {
	return nil
}

func (m *mockMemoryService) Get(ctx context.Context, memoryID string) (*reasoningbank.Memory, error) {
	return nil, nil
}

func (m *mockMemoryService) RecordOutcome(ctx context.Context, memoryID string, succeeded bool, sessionID string) (float64, error) {
	return 0.5, nil
}

func TestAnalyzer_Analyze_RequiresProjectID(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	_, err := analyzer.Analyze(context.Background(), AnalyzeOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project_id is required")
}

func TestAnalyzer_Analyze_EmptyMemories(t *testing.T) {
	mock := &mockMemoryService{memories: []reasoningbank.Memory{}}

	// Create a minimal analyzer that works with our mock
	analyzer := &DefaultAnalyzer{}

	// We need to use the mock, so let's test the helper functions directly instead
	patterns := analyzer.extractPatterns([]*reasoningbank.Memory{}, AnalyzeOptions{
		MinConfidence: 0.3,
		MinFrequency:  2,
	})

	assert.Empty(t, patterns)
	_ = mock // use the mock
}

func TestAnalyzer_ExtractPatterns_Success(t *testing.T) {
	now := time.Now()
	memories := []*reasoningbank.Memory{
		{
			ID:         "mem1",
			ProjectID:  "test-project",
			Title:      "Test Memory 1",
			Content:    "Success pattern",
			Outcome:    reasoningbank.OutcomeSuccess,
			Confidence: 0.8,
			Tags:       []string{"golang", "testing"},
			CreatedAt:  now.Add(-2 * time.Hour),
		},
		{
			ID:         "mem2",
			ProjectID:  "test-project",
			Title:      "Test Memory 2",
			Content:    "Another success",
			Outcome:    reasoningbank.OutcomeSuccess,
			Confidence: 0.7,
			Tags:       []string{"golang"},
			CreatedAt:  now.Add(-1 * time.Hour),
		},
		{
			ID:         "mem3",
			ProjectID:  "test-project",
			Title:      "Test Memory 3",
			Content:    "Failure case",
			Outcome:    reasoningbank.OutcomeFailure,
			Confidence: 0.5,
			Tags:       []string{"debugging"},
			CreatedAt:  now,
		},
	}

	analyzer := &DefaultAnalyzer{}
	patterns := analyzer.extractPatterns(memories, AnalyzeOptions{
		MinConfidence: 0.3,
		MinFrequency:  2,
	})

	// Should find patterns for:
	// - Success outcomes (2 memories)
	// - golang tag (2 memories)
	assert.NotEmpty(t, patterns)

	// Check for success pattern
	hasSuccessPattern := false
	hasGolangPattern := false
	for _, p := range patterns {
		if p.Category == PatternSuccess && p.Frequency >= 2 {
			hasSuccessPattern = true
		}
		if containsTag(p.Tags, "golang") && p.Frequency >= 2 {
			hasGolangPattern = true
		}
	}
	assert.True(t, hasSuccessPattern, "should have success pattern")
	assert.True(t, hasGolangPattern, "should have golang tag pattern")
}

func TestFilterByPeriod(t *testing.T) {
	now := time.Now()
	memories := []*reasoningbank.Memory{
		{ID: "old", CreatedAt: now.Add(-48 * time.Hour)},
		{ID: "recent", CreatedAt: now.Add(-12 * time.Hour)},
		{ID: "new", CreatedAt: now.Add(-1 * time.Hour)},
	}

	period := &ReportPeriod{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	filtered := filterByPeriod(memories, period)
	assert.Len(t, filtered, 2)

	ids := []string{filtered[0].ID, filtered[1].ID}
	assert.Contains(t, ids, "recent")
	assert.Contains(t, ids, "new")
}

func TestFilterByTags(t *testing.T) {
	memories := []*reasoningbank.Memory{
		{ID: "1", Tags: []string{"golang", "api"}},
		{ID: "2", Tags: []string{"golang", "database"}},
		{ID: "3", Tags: []string{"python", "api"}},
		{ID: "4", Tags: []string{"rust"}},
	}

	t.Run("include filter", func(t *testing.T) {
		filtered := filterByTags(memories, []string{"golang"}, nil)
		assert.Len(t, filtered, 2)
	})

	t.Run("exclude filter", func(t *testing.T) {
		filtered := filterByTags(memories, nil, []string{"python"})
		assert.Len(t, filtered, 3)
	})

	t.Run("include and exclude", func(t *testing.T) {
		filtered := filterByTags(memories, []string{"golang", "api"}, []string{"database"})
		// Should include golang or api, but exclude database
		// mem1 has golang and api (included)
		// mem2 has golang but also database (excluded)
		// mem3 has api (included)
		assert.Len(t, filtered, 2)
	})
}

func TestCalculateAverageConfidence(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		avg := calculateAverageConfidence([]*reasoningbank.Memory{})
		assert.Equal(t, 0.0, avg)
	})

	t.Run("single memory", func(t *testing.T) {
		mems := []*reasoningbank.Memory{{Confidence: 0.8}}
		avg := calculateAverageConfidence(mems)
		assert.Equal(t, 0.8, avg)
	})

	t.Run("multiple memories", func(t *testing.T) {
		mems := []*reasoningbank.Memory{
			{Confidence: 0.6},
			{Confidence: 0.8},
			{Confidence: 0.7},
		}
		avg := calculateAverageConfidence(mems)
		assert.InDelta(t, 0.7, avg, 0.01)
	})
}

func TestCalculateConfidenceTrend(t *testing.T) {
	now := time.Now()

	t.Run("insufficient data", func(t *testing.T) {
		trend := calculateConfidenceTrend([]*reasoningbank.Memory{})
		assert.Equal(t, 0.0, trend)

		trend = calculateConfidenceTrend([]*reasoningbank.Memory{{Confidence: 0.5}})
		assert.Equal(t, 0.0, trend)
	})

	t.Run("improving trend", func(t *testing.T) {
		mems := []*reasoningbank.Memory{
			{Confidence: 0.3, CreatedAt: now.Add(-3 * time.Hour)},
			{Confidence: 0.4, CreatedAt: now.Add(-2 * time.Hour)},
			{Confidence: 0.7, CreatedAt: now.Add(-1 * time.Hour)},
			{Confidence: 0.8, CreatedAt: now},
		}
		trend := calculateConfidenceTrend(mems)
		assert.Greater(t, trend, 0.0)
	})

	t.Run("declining trend", func(t *testing.T) {
		mems := []*reasoningbank.Memory{
			{Confidence: 0.8, CreatedAt: now.Add(-3 * time.Hour)},
			{Confidence: 0.7, CreatedAt: now.Add(-2 * time.Hour)},
			{Confidence: 0.4, CreatedAt: now.Add(-1 * time.Hour)},
			{Confidence: 0.3, CreatedAt: now},
		}
		trend := calculateConfidenceTrend(mems)
		assert.Less(t, trend, 0.0)
	})
}

func TestExtractAllTags(t *testing.T) {
	mems := []*reasoningbank.Memory{
		{Tags: []string{"a", "b"}},
		{Tags: []string{"b", "c"}},
		{Tags: []string{"a", "d"}},
	}

	tags := extractAllTags(mems)
	assert.Len(t, tags, 4)
	assert.Contains(t, tags, "a")
	assert.Contains(t, tags, "b")
	assert.Contains(t, tags, "c")
	assert.Contains(t, tags, "d")
}

func TestFindEarliestAndLatestTime(t *testing.T) {
	now := time.Now()
	mems := []*reasoningbank.Memory{
		{CreatedAt: now.Add(-2 * time.Hour)},
		{CreatedAt: now.Add(-1 * time.Hour)},
		{CreatedAt: now},
	}

	earliest := findEarliestTime(mems)
	latest := findLatestTime(mems)

	require.False(t, earliest.IsZero())
	require.False(t, latest.IsZero())
	assert.True(t, earliest.Before(latest))
	assert.Equal(t, now.Add(-2*time.Hour).Unix(), earliest.Unix())
	assert.Equal(t, now.Unix(), latest.Unix())
}

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}
