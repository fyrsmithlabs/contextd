package reflection

import (
	"context"
	"time"
)

// PatternCategory represents the category of a behavioral pattern.
type PatternCategory string

const (
	// PatternSuccess indicates a pattern of successful behaviors.
	PatternSuccess PatternCategory = "success"
	// PatternFailure indicates a pattern of failed behaviors.
	PatternFailure PatternCategory = "failure"
	// PatternRecurring indicates a recurring behavioral pattern.
	PatternRecurring PatternCategory = "recurring"
	// PatternImproving indicates a pattern showing improvement.
	PatternImproving PatternCategory = "improving"
	// PatternDeclining indicates a pattern showing decline.
	PatternDeclining PatternCategory = "declining"
)

// Pattern represents a behavioral pattern identified in memories.
type Pattern struct {
	// ID is the unique identifier for this pattern.
	ID string `json:"id"`
	// Category classifies the type of pattern.
	Category PatternCategory `json:"category"`
	// Description summarizes the pattern.
	Description string `json:"description"`
	// Tags associated with this pattern.
	Tags []string `json:"tags"`
	// Domains where this pattern appears.
	Domains []string `json:"domains"`
	// Frequency is how often this pattern occurs.
	Frequency int `json:"frequency"`
	// Confidence score for this pattern (0-1).
	Confidence float64 `json:"confidence"`
	// MemoryIDs are the memories that contributed to this pattern.
	MemoryIDs []string `json:"memory_ids"`
	// FirstSeen is when this pattern was first observed.
	FirstSeen time.Time `json:"first_seen"`
	// LastSeen is when this pattern was last observed.
	LastSeen time.Time `json:"last_seen"`
}

// Correlation represents a relationship between patterns or memories.
type Correlation struct {
	// ID is the unique identifier for this correlation.
	ID string `json:"id"`
	// SourceID is the ID of the source pattern/memory.
	SourceID string `json:"source_id"`
	// TargetID is the ID of the target pattern/memory.
	TargetID string `json:"target_id"`
	// Type describes the type of correlation.
	Type CorrelationType `json:"type"`
	// Strength indicates correlation strength (0-1).
	Strength float64 `json:"strength"`
	// Description explains the correlation.
	Description string `json:"description"`
}

// CorrelationType represents the type of correlation between items.
type CorrelationType string

const (
	// CorrelationCausal indicates a causal relationship.
	CorrelationCausal CorrelationType = "causal"
	// CorrelationSimilar indicates similar patterns.
	CorrelationSimilar CorrelationType = "similar"
	// CorrelationOpposite indicates contrasting patterns.
	CorrelationOpposite CorrelationType = "opposite"
	// CorrelationSequential indicates sequential occurrence.
	CorrelationSequential CorrelationType = "sequential"
	// CorrelationCoOccurs indicates patterns that occur together.
	CorrelationCoOccurs CorrelationType = "co_occurs"
)

// Insight represents a key learning derived from analysis.
type Insight struct {
	// Title is a brief summary of the insight.
	Title string `json:"title"`
	// Description provides details about the insight.
	Description string `json:"description"`
	// Category classifies the insight.
	Category string `json:"category"`
	// Confidence score for this insight (0-1).
	Confidence float64 `json:"confidence"`
	// RelatedPatterns are pattern IDs that support this insight.
	RelatedPatterns []string `json:"related_patterns,omitempty"`
	// Recommendations based on this insight.
	Recommendations []string `json:"recommendations,omitempty"`
}

// ReflectionReport is a comprehensive analysis report.
type ReflectionReport struct {
	// ID is the unique identifier for this report.
	ID string `json:"id"`
	// ProjectID is the project this report covers.
	ProjectID string `json:"project_id"`
	// GeneratedAt is when the report was created.
	GeneratedAt time.Time `json:"generated_at"`
	// Period describes the time period covered.
	Period ReportPeriod `json:"period"`
	// Summary provides a high-level overview.
	Summary string `json:"summary"`
	// Patterns discovered during analysis.
	Patterns []Pattern `json:"patterns"`
	// Correlations between patterns/memories.
	Correlations []Correlation `json:"correlations"`
	// Insights derived from analysis.
	Insights []Insight `json:"insights"`
	// Statistics about the analyzed data.
	Statistics ReportStatistics `json:"statistics"`
	// Recommendations for improvement.
	Recommendations []string `json:"recommendations"`
}

// ReportPeriod describes the time period for a report.
type ReportPeriod struct {
	// Start of the period.
	Start time.Time `json:"start"`
	// End of the period.
	End time.Time `json:"end"`
	// Description like "Last 7 days" or "This month".
	Description string `json:"description"`
}

// ReportStatistics contains numerical summary of the report.
type ReportStatistics struct {
	// TotalMemories analyzed.
	TotalMemories int `json:"total_memories"`
	// SuccessfulMemories count.
	SuccessfulMemories int `json:"successful_memories"`
	// FailedMemories count.
	FailedMemories int `json:"failed_memories"`
	// UniquePatterns identified.
	UniquePatterns int `json:"unique_patterns"`
	// UniqueCorrelations found.
	UniqueCorrelations int `json:"unique_correlations"`
	// TopTags most frequently used.
	TopTags []TagCount `json:"top_tags"`
	// TopDomains most active.
	TopDomains []DomainCount `json:"top_domains"`
	// SuccessRate overall.
	SuccessRate float64 `json:"success_rate"`
	// AverageConfidence across memories.
	AverageConfidence float64 `json:"average_confidence"`
}

// TagCount represents a tag and its occurrence count.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// DomainCount represents a domain and its occurrence count.
type DomainCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

// AnalyzeOptions configures pattern analysis.
type AnalyzeOptions struct {
	// ProjectID to analyze.
	ProjectID string
	// MinConfidence threshold for patterns.
	MinConfidence float64
	// MinFrequency threshold for patterns.
	MinFrequency int
	// IncludeTags to filter by.
	IncludeTags []string
	// ExcludeTags to filter out.
	ExcludeTags []string
	// Period to analyze.
	Period *ReportPeriod
	// MaxPatterns to return.
	MaxPatterns int
}

// CorrelateOptions configures correlation analysis.
type CorrelateOptions struct {
	// PatternIDs to correlate (empty = all).
	PatternIDs []string
	// MinStrength threshold for correlations.
	MinStrength float64
	// Types of correlations to find.
	Types []CorrelationType
	// MaxCorrelations to return.
	MaxCorrelations int
}

// ReportOptions configures report generation.
type ReportOptions struct {
	// ProjectID to report on.
	ProjectID string
	// Period to cover.
	Period ReportPeriod
	// IncludePatterns in the report.
	IncludePatterns bool
	// IncludeCorrelations in the report.
	IncludeCorrelations bool
	// IncludeInsights in the report.
	IncludeInsights bool
	// MaxInsights to include.
	MaxInsights int
	// Format for output ("text", "json", "markdown").
	Format string
}

// Analyzer identifies patterns in memories.
type Analyzer interface {
	// Analyze finds patterns in memories for a project.
	Analyze(ctx context.Context, opts AnalyzeOptions) ([]Pattern, error)
}

// Correlator finds relationships between patterns and memories.
type Correlator interface {
	// Correlate finds relationships between patterns.
	Correlate(patterns []Pattern, opts CorrelateOptions) ([]Correlation, error)
}

// Reporter generates reflection reports.
type Reporter interface {
	// Generate creates a reflection report.
	Generate(ctx context.Context, opts ReportOptions) (*ReflectionReport, error)
}
