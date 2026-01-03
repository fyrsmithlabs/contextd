package reflection

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
)

// DefaultReporter implements report generation.
type DefaultReporter struct {
	memorySvc  *reasoningbank.Service
	analyzer   *DefaultAnalyzer
	correlator *DefaultCorrelator
}

// NewReporter creates a new report generator.
func NewReporter(memorySvc *reasoningbank.Service) *DefaultReporter {
	return &DefaultReporter{
		memorySvc:  memorySvc,
		analyzer:   NewAnalyzer(memorySvc),
		correlator: NewCorrelator(),
	}
}

// Generate creates a comprehensive reflection report.
func (r *DefaultReporter) Generate(ctx context.Context, opts ReportOptions) (*ReflectionReport, error) {
	if opts.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	// Set defaults
	if opts.MaxInsights == 0 {
		opts.MaxInsights = 10
	}
	if opts.Format == "" {
		opts.Format = "json"
	}

	// Retrieve memories for statistics
	rawMemories, err := r.memorySvc.Search(ctx, opts.ProjectID, "", 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve memories: %w", err)
	}

	// Convert to pointer slice for filtering
	memories := make([]*reasoningbank.Memory, len(rawMemories))
	for i := range rawMemories {
		memories[i] = &rawMemories[i]
	}

	// Filter by period
	if !opts.Period.Start.IsZero() && !opts.Period.End.IsZero() {
		memories = filterMemoriesByPeriod(memories, opts.Period)
	}

	report := &ReflectionReport{
		ID:          uuid.New().String(),
		ProjectID:   opts.ProjectID,
		GeneratedAt: time.Now(),
		Period:      opts.Period,
	}

	// Calculate statistics
	report.Statistics = r.calculateStatistics(memories)

	// Analyze patterns if requested
	if opts.IncludePatterns {
		patterns, err := r.analyzer.Analyze(ctx, AnalyzeOptions{
			ProjectID:     opts.ProjectID,
			MinConfidence: 0.3,
			MinFrequency:  2,
			Period:        &opts.Period,
			MaxPatterns:   20,
		})
		if err != nil {
			return nil, fmt.Errorf("pattern analysis failed: %w", err)
		}
		report.Patterns = patterns
	}

	// Find correlations if requested
	if opts.IncludeCorrelations && len(report.Patterns) > 0 {
		correlations, err := r.correlator.Correlate(report.Patterns, CorrelateOptions{
			MinStrength:     0.3,
			MaxCorrelations: 30,
		})
		if err != nil {
			return nil, fmt.Errorf("correlation analysis failed: %w", err)
		}
		report.Correlations = correlations
	}

	// Generate insights if requested
	if opts.IncludeInsights {
		report.Insights = r.generateInsights(report, opts.MaxInsights)
	}

	// Generate recommendations
	report.Recommendations = r.generateRecommendations(report)

	// Generate summary
	report.Summary = r.generateSummary(report)

	return report, nil
}

// filterMemoriesByPeriod filters memories to those within the period.
func filterMemoriesByPeriod(memories []*reasoningbank.Memory, period ReportPeriod) []*reasoningbank.Memory {
	filtered := make([]*reasoningbank.Memory, 0)
	for _, m := range memories {
		if m.CreatedAt.After(period.Start) && m.CreatedAt.Before(period.End) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// calculateStatistics computes summary statistics.
func (r *DefaultReporter) calculateStatistics(memories []*reasoningbank.Memory) ReportStatistics {
	stats := ReportStatistics{
		TotalMemories: len(memories),
	}

	if len(memories) == 0 {
		return stats
	}

	tagCounts := make(map[string]int)
	var totalConfidence float64

	for _, m := range memories {
		// Count outcomes
		if m.Outcome == reasoningbank.OutcomeSuccess {
			stats.SuccessfulMemories++
		} else {
			stats.FailedMemories++
		}

		// Count tags
		for _, tag := range m.Tags {
			tagCounts[tag]++
		}

		totalConfidence += m.Confidence
	}

	// Calculate success rate
	stats.SuccessRate = float64(stats.SuccessfulMemories) / float64(stats.TotalMemories)

	// Calculate average confidence
	stats.AverageConfidence = totalConfidence / float64(stats.TotalMemories)

	// Get top tags
	stats.TopTags = getTopTags(tagCounts, 10)

	return stats
}

// getTopTags returns the most frequent tags.
func getTopTags(tagCounts map[string]int, limit int) []TagCount {
	tags := make([]TagCount, 0, len(tagCounts))
	for tag, count := range tagCounts {
		tags = append(tags, TagCount{Tag: tag, Count: count})
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Count > tags[j].Count
	})

	if len(tags) > limit {
		tags = tags[:limit]
	}
	return tags
}

// generateInsights derives key learnings from the analysis.
func (r *DefaultReporter) generateInsights(report *ReflectionReport, maxInsights int) []Insight {
	var insights []Insight

	// Insight: Success rate trend
	if report.Statistics.SuccessRate >= 0.7 {
		insights = append(insights, Insight{
			Title:       "High Success Rate",
			Description: fmt.Sprintf("%.0f%% success rate indicates effective approaches are being captured", report.Statistics.SuccessRate*100),
			Category:    "performance",
			Confidence:  0.8,
		})
	} else if report.Statistics.SuccessRate < 0.5 {
		insights = append(insights, Insight{
			Title:       "Improvement Opportunity",
			Description: fmt.Sprintf("%.0f%% success rate suggests room for learning from failures", report.Statistics.SuccessRate*100),
			Category:    "performance",
			Confidence:  0.8,
			Recommendations: []string{
				"Review failed approaches for common patterns",
				"Consider documenting what didn't work and why",
			},
		})
	}

	// Insight: Pattern distribution
	successPatterns := 0
	failurePatterns := 0
	for _, p := range report.Patterns {
		switch p.Category {
		case PatternSuccess:
			successPatterns++
		case PatternFailure:
			failurePatterns++
		}
	}

	if successPatterns > 0 && failurePatterns > 0 {
		insights = append(insights, Insight{
			Title: "Balanced Learning",
			Description: fmt.Sprintf("Found %d success patterns and %d failure patterns, indicating learning from both outcomes",
				successPatterns, failurePatterns),
			Category:   "learning",
			Confidence: 0.7,
		})
	}

	// Insight: Top performing tags
	if len(report.Statistics.TopTags) > 0 {
		topTag := report.Statistics.TopTags[0]
		insights = append(insights, Insight{
			Title:       "Primary Focus Area",
			Description: fmt.Sprintf("'%s' appears most frequently (%d occurrences), indicating a primary area of activity", topTag.Tag, topTag.Count),
			Category:    "focus",
			Confidence:  0.9,
		})
	}

	// Insight: Correlation patterns
	if len(report.Correlations) > 0 {
		strongCorrelations := 0
		for _, c := range report.Correlations {
			if c.Strength >= 0.6 {
				strongCorrelations++
			}
		}
		if strongCorrelations > 0 {
			insights = append(insights, Insight{
				Title:       "Related Patterns Found",
				Description: fmt.Sprintf("%d strong correlations between patterns suggest interconnected learning areas", strongCorrelations),
				Category:    "correlation",
				Confidence:  0.7,
			})
		}
	}

	// Insight: Improving trends
	improving := 0
	declining := 0
	for _, p := range report.Patterns {
		switch p.Category {
		case PatternImproving:
			improving++
		case PatternDeclining:
			declining++
		}
	}

	if improving > declining {
		insights = append(insights, Insight{
			Title:       "Positive Trend",
			Description: fmt.Sprintf("%d improving patterns vs %d declining patterns shows overall positive trajectory", improving, declining),
			Category:    "trend",
			Confidence:  0.75,
		})
	} else if declining > improving {
		insights = append(insights, Insight{
			Title:       "Attention Needed",
			Description: fmt.Sprintf("%d declining patterns vs %d improving patterns may need attention", declining, improving),
			Category:    "trend",
			Confidence:  0.75,
			Recommendations: []string{
				"Review declining patterns for root causes",
				"Consider updating strategies in declining areas",
			},
		})
	}

	// Limit insights
	if len(insights) > maxInsights {
		insights = insights[:maxInsights]
	}

	return insights
}

// generateRecommendations creates actionable recommendations.
func (r *DefaultReporter) generateRecommendations(report *ReflectionReport) []string {
	var recommendations []string

	// Based on statistics
	if report.Statistics.TotalMemories < 10 {
		recommendations = append(recommendations, "Continue recording learnings to build a stronger knowledge base")
	}

	if report.Statistics.SuccessRate < 0.5 && report.Statistics.TotalMemories > 5 {
		recommendations = append(recommendations, "Analyze failure patterns to identify common issues")
	}

	// Based on patterns
	for _, p := range report.Patterns {
		if p.Category == PatternDeclining && p.Confidence > 0.6 {
			recommendations = append(recommendations,
				fmt.Sprintf("Review declining pattern in '%s' area for potential issues", strings.Join(p.Tags, ", ")))
		}
	}

	// Based on correlations
	for _, c := range report.Correlations {
		if c.Type == CorrelationOpposite && c.Strength > 0.6 {
			recommendations = append(recommendations, "Compare opposing patterns to identify what distinguishes success from failure")
			break
		}
	}

	// General recommendations
	if len(report.Patterns) > 0 && len(recommendations) == 0 {
		recommendations = append(recommendations, "Continue building on successful patterns identified in this analysis")
	}

	return recommendations
}

// generateSummary creates a high-level summary of the report.
func (r *DefaultReporter) generateSummary(report *ReflectionReport) string {
	var parts []string

	// Memory overview
	parts = append(parts, fmt.Sprintf("Analyzed %d memories", report.Statistics.TotalMemories))

	// Success rate
	if report.Statistics.TotalMemories > 0 {
		parts = append(parts, fmt.Sprintf("with %.0f%% success rate", report.Statistics.SuccessRate*100))
	}

	// Patterns found
	if len(report.Patterns) > 0 {
		parts = append(parts, fmt.Sprintf("Identified %d patterns", len(report.Patterns)))
	}

	// Correlations found
	if len(report.Correlations) > 0 {
		parts = append(parts, fmt.Sprintf("and %d correlations", len(report.Correlations)))
	}

	// Insights
	if len(report.Insights) > 0 {
		parts = append(parts, fmt.Sprintf("Generated %d insights", len(report.Insights)))
	}

	return strings.Join(parts, ". ") + "."
}

// FormatReport formats a report as text, markdown, or JSON.
func FormatReport(report *ReflectionReport, format string) string {
	switch format {
	case "markdown":
		return formatAsMarkdown(report)
	case "text":
		return formatAsText(report)
	default:
		// JSON is handled by the caller via json.Marshal
		return ""
	}
}

// formatAsMarkdown formats the report as markdown.
func formatAsMarkdown(report *ReflectionReport) string {
	var sb strings.Builder

	sb.WriteString("# Reflection Report\n\n")
	sb.WriteString(fmt.Sprintf("**Project:** %s\n", report.ProjectID))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", report.GeneratedAt.Format(time.RFC3339)))

	sb.WriteString("## Summary\n\n")
	sb.WriteString(report.Summary + "\n\n")

	sb.WriteString("## Statistics\n\n")
	sb.WriteString(fmt.Sprintf("- Total Memories: %d\n", report.Statistics.TotalMemories))
	sb.WriteString(fmt.Sprintf("- Success Rate: %.1f%%\n", report.Statistics.SuccessRate*100))
	sb.WriteString(fmt.Sprintf("- Average Confidence: %.2f\n\n", report.Statistics.AverageConfidence))

	if len(report.Insights) > 0 {
		sb.WriteString("## Key Insights\n\n")
		for _, insight := range report.Insights {
			sb.WriteString(fmt.Sprintf("### %s\n\n", insight.Title))
			sb.WriteString(insight.Description + "\n\n")
		}
	}

	if len(report.Recommendations) > 0 {
		sb.WriteString("## Recommendations\n\n")
		for _, rec := range report.Recommendations {
			sb.WriteString(fmt.Sprintf("- %s\n", rec))
		}
	}

	return sb.String()
}

// formatAsText formats the report as plain text.
func formatAsText(report *ReflectionReport) string {
	var sb strings.Builder

	sb.WriteString("REFLECTION REPORT\n")
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")

	sb.WriteString(fmt.Sprintf("Project: %s\n", report.ProjectID))
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", report.GeneratedAt.Format(time.RFC3339)))

	sb.WriteString("SUMMARY\n")
	sb.WriteString(strings.Repeat("-", 20) + "\n")
	sb.WriteString(report.Summary + "\n\n")

	sb.WriteString("STATISTICS\n")
	sb.WriteString(strings.Repeat("-", 20) + "\n")
	sb.WriteString(fmt.Sprintf("Total Memories: %d\n", report.Statistics.TotalMemories))
	sb.WriteString(fmt.Sprintf("Success Rate: %.1f%%\n", report.Statistics.SuccessRate*100))
	sb.WriteString(fmt.Sprintf("Average Confidence: %.2f\n\n", report.Statistics.AverageConfidence))

	if len(report.Insights) > 0 {
		sb.WriteString("KEY INSIGHTS\n")
		sb.WriteString(strings.Repeat("-", 20) + "\n")
		for i, insight := range report.Insights {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, insight.Title))
			sb.WriteString(fmt.Sprintf("   %s\n\n", insight.Description))
		}
	}

	if len(report.Recommendations) > 0 {
		sb.WriteString("RECOMMENDATIONS\n")
		sb.WriteString(strings.Repeat("-", 20) + "\n")
		for i, rec := range report.Recommendations {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
		}
	}

	return sb.String()
}

// Ensure DefaultReporter implements Reporter.
var _ Reporter = (*DefaultReporter)(nil)
