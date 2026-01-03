package reflection

import (
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/stretchr/testify/assert"
)

func TestReporter_CalculateStatistics_Empty(t *testing.T) {
	reporter := &DefaultReporter{}
	stats := reporter.calculateStatistics([]*reasoningbank.Memory{})

	assert.Equal(t, 0, stats.TotalMemories)
	assert.Equal(t, 0, stats.SuccessfulMemories)
	assert.Equal(t, 0, stats.FailedMemories)
	assert.Equal(t, 0.0, stats.SuccessRate)
	assert.Equal(t, 0.0, stats.AverageConfidence)
}

func TestReporter_CalculateStatistics_WithMemories(t *testing.T) {
	reporter := &DefaultReporter{}
	memories := []*reasoningbank.Memory{
		{Outcome: reasoningbank.OutcomeSuccess, Confidence: 0.8, Tags: []string{"golang", "api"}},
		{Outcome: reasoningbank.OutcomeSuccess, Confidence: 0.7, Tags: []string{"golang"}},
		{Outcome: reasoningbank.OutcomeFailure, Confidence: 0.4, Tags: []string{"debugging"}},
		{Outcome: reasoningbank.OutcomeSuccess, Confidence: 0.9, Tags: []string{"golang", "testing"}},
	}

	stats := reporter.calculateStatistics(memories)

	assert.Equal(t, 4, stats.TotalMemories)
	assert.Equal(t, 3, stats.SuccessfulMemories)
	assert.Equal(t, 1, stats.FailedMemories)
	assert.Equal(t, 0.75, stats.SuccessRate)
	assert.InDelta(t, 0.7, stats.AverageConfidence, 0.01)

	// Check top tags
	assert.NotEmpty(t, stats.TopTags)
	// golang should be the top tag (3 occurrences)
	assert.Equal(t, "golang", stats.TopTags[0].Tag)
	assert.Equal(t, 3, stats.TopTags[0].Count)
}

func TestReporter_GenerateInsights_HighSuccessRate(t *testing.T) {
	reporter := &DefaultReporter{}
	report := &ReflectionReport{
		Statistics: ReportStatistics{
			SuccessRate: 0.85,
		},
		Patterns: []Pattern{},
	}

	insights := reporter.generateInsights(report, 10)

	hasHighSuccessInsight := false
	for _, i := range insights {
		if i.Title == "High Success Rate" {
			hasHighSuccessInsight = true
		}
	}
	assert.True(t, hasHighSuccessInsight)
}

func TestReporter_GenerateInsights_LowSuccessRate(t *testing.T) {
	reporter := &DefaultReporter{}
	report := &ReflectionReport{
		Statistics: ReportStatistics{
			SuccessRate: 0.35,
		},
		Patterns: []Pattern{},
	}

	insights := reporter.generateInsights(report, 10)

	hasImprovementInsight := false
	for _, i := range insights {
		if i.Title == "Improvement Opportunity" {
			hasImprovementInsight = true
			assert.NotEmpty(t, i.Recommendations)
		}
	}
	assert.True(t, hasImprovementInsight)
}

func TestReporter_GenerateInsights_PrimaryFocus(t *testing.T) {
	reporter := &DefaultReporter{}
	report := &ReflectionReport{
		Statistics: ReportStatistics{
			SuccessRate: 0.6,
			TopTags: []TagCount{
				{Tag: "kubernetes", Count: 15},
				{Tag: "debugging", Count: 5},
			},
		},
		Patterns: []Pattern{},
	}

	insights := reporter.generateInsights(report, 10)

	hasFocusInsight := false
	for _, i := range insights {
		if i.Title == "Primary Focus Area" {
			hasFocusInsight = true
			assert.Contains(t, i.Description, "kubernetes")
		}
	}
	assert.True(t, hasFocusInsight)
}

func TestReporter_GenerateRecommendations(t *testing.T) {
	reporter := &DefaultReporter{}

	t.Run("few memories", func(t *testing.T) {
		report := &ReflectionReport{
			Statistics: ReportStatistics{
				TotalMemories: 5,
				SuccessRate:   0.6,
			},
		}
		recs := reporter.generateRecommendations(report)
		assert.NotEmpty(t, recs)

		hasRecordMore := false
		for _, r := range recs {
			if contains(r, "recording") || contains(r, "knowledge base") {
				hasRecordMore = true
			}
		}
		assert.True(t, hasRecordMore)
	})

	t.Run("low success rate", func(t *testing.T) {
		report := &ReflectionReport{
			Statistics: ReportStatistics{
				TotalMemories: 20,
				SuccessRate:   0.3,
			},
		}
		recs := reporter.generateRecommendations(report)
		assert.NotEmpty(t, recs)

		hasAnalyzeFailures := false
		for _, r := range recs {
			if contains(r, "failure") {
				hasAnalyzeFailures = true
			}
		}
		assert.True(t, hasAnalyzeFailures)
	})
}

func TestReporter_GenerateSummary(t *testing.T) {
	reporter := &DefaultReporter{}
	report := &ReflectionReport{
		Statistics: ReportStatistics{
			TotalMemories: 50,
			SuccessRate:   0.7,
		},
		Patterns:     make([]Pattern, 5),
		Correlations: make([]Correlation, 3),
		Insights:     make([]Insight, 4),
	}

	summary := reporter.generateSummary(report)

	assert.Contains(t, summary, "50 memories")
	assert.Contains(t, summary, "70%")
	assert.Contains(t, summary, "5 patterns")
	assert.Contains(t, summary, "3 correlations")
	assert.Contains(t, summary, "4 insights")
}

func TestFormatReport_Markdown(t *testing.T) {
	report := &ReflectionReport{
		ID:          "test-report",
		ProjectID:   "test-project",
		GeneratedAt: time.Now(),
		Summary:     "Test summary",
		Statistics: ReportStatistics{
			TotalMemories:     10,
			SuccessRate:       0.8,
			AverageConfidence: 0.75,
		},
		Insights: []Insight{
			{Title: "Test Insight", Description: "Test description"},
		},
		Recommendations: []string{"Do this", "Do that"},
	}

	output := FormatReport(report, "markdown")

	assert.Contains(t, output, "# Reflection Report")
	assert.Contains(t, output, "**Project:** test-project")
	assert.Contains(t, output, "## Summary")
	assert.Contains(t, output, "Test summary")
	assert.Contains(t, output, "## Statistics")
	assert.Contains(t, output, "## Key Insights")
	assert.Contains(t, output, "### Test Insight")
	assert.Contains(t, output, "## Recommendations")
}

func TestFormatReport_Text(t *testing.T) {
	report := &ReflectionReport{
		ID:          "test-report",
		ProjectID:   "test-project",
		GeneratedAt: time.Now(),
		Summary:     "Test summary",
		Statistics: ReportStatistics{
			TotalMemories:     10,
			SuccessRate:       0.8,
			AverageConfidence: 0.75,
		},
		Insights: []Insight{
			{Title: "Test Insight", Description: "Test description"},
		},
		Recommendations: []string{"Do this"},
	}

	output := FormatReport(report, "text")

	assert.Contains(t, output, "REFLECTION REPORT")
	assert.Contains(t, output, "Project: test-project")
	assert.Contains(t, output, "SUMMARY")
	assert.Contains(t, output, "STATISTICS")
	assert.Contains(t, output, "KEY INSIGHTS")
	assert.Contains(t, output, "RECOMMENDATIONS")
}

func TestFilterMemoriesByPeriod(t *testing.T) {
	now := time.Now()
	memories := []*reasoningbank.Memory{
		{ID: "old", CreatedAt: now.Add(-7 * 24 * time.Hour)},
		{ID: "recent", CreatedAt: now.Add(-2 * 24 * time.Hour)},
		{ID: "new", CreatedAt: now.Add(-1 * time.Hour)},
	}

	period := ReportPeriod{
		Start: now.Add(-3 * 24 * time.Hour),
		End:   now,
	}

	filtered := filterMemoriesByPeriod(memories, period)
	assert.Len(t, filtered, 2)
}

func TestGetTopTags(t *testing.T) {
	tagCounts := map[string]int{
		"golang":   15,
		"python":   5,
		"rust":     3,
		"database": 10,
		"api":      8,
	}

	t.Run("limit 3", func(t *testing.T) {
		top := getTopTags(tagCounts, 3)
		assert.Len(t, top, 3)
		assert.Equal(t, "golang", top[0].Tag)
		assert.Equal(t, 15, top[0].Count)
		assert.Equal(t, "database", top[1].Tag)
	})

	t.Run("limit exceeds count", func(t *testing.T) {
		top := getTopTags(tagCounts, 10)
		assert.Len(t, top, 5)
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
