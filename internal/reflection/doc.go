// Package reflection provides self-reflection capabilities for analyzing
// memory patterns and generating insights from past sessions.
//
// The package supports:
//   - Pattern analysis across memories (success, failure, recurring, improving, declining)
//   - Correlation detection between tags and outcomes
//   - Insight generation based on patterns and correlations
//   - Report generation in multiple formats (JSON, text, markdown)
//
// # Architecture
//
// The main components are:
//   - Analyzer: Identifies patterns in memories by category and frequency
//   - Reporter: Generates comprehensive reflection reports
//   - Pattern: Represents a detected behavioral pattern
//   - Insight: Actionable recommendation derived from patterns
//
// # Usage
//
// Create an analyzer with a ReasoningBank service:
//
//	analyzer := reflection.NewAnalyzer(reasoningBankSvc)
//	patterns, err := analyzer.Analyze(ctx, reflection.AnalyzeOptions{
//	    ProjectID:     "my-project",
//	    MinConfidence: 0.3,
//	    MinFrequency:  2,
//	    MaxPatterns:   20,
//	})
//
// Generate a reflection report:
//
//	reporter := reflection.NewReporter(reasoningBankSvc)
//	report, err := reporter.Generate(ctx, reflection.ReportOptions{
//	    ProjectID:           "my-project",
//	    Period:              period,
//	    IncludePatterns:     true,
//	    IncludeCorrelations: true,
//	    IncludeInsights:     true,
//	    MaxInsights:         10,
//	    Format:              "markdown",
//	})
//
// # Pattern Categories
//
// Patterns are grouped into categories:
//   - Success: Strategies that consistently work well
//   - Failure: Approaches that frequently fail
//   - Recurring: Patterns that appear regularly
//   - Improving: Patterns with increasing confidence over time
//   - Declining: Patterns with decreasing effectiveness
//
// # Report Persistence
//
// Reports can be persisted to disk using StoreReflectionReport(), which
// saves to .claude/reflections/ in the project directory with path traversal
// protection and restrictive file permissions.
package reflection
