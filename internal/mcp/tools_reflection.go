package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/fyrsmithlabs/contextd/internal/reflection"
)

// ===== REFLECTION TOOLS =====

type reflectReportInput struct {
	ProjectID           string `json:"project_id" jsonschema:"required,Project identifier"`
	ProjectPath         string `json:"project_path,omitempty" jsonschema:"Project path for repository context"`
	PeriodDays          int    `json:"period_days,omitempty" jsonschema:"Number of days to analyze (default: 30)"`
	IncludePatterns     bool   `json:"include_patterns,omitempty" jsonschema:"Include pattern analysis (default: true)"`
	IncludeCorrelations bool   `json:"include_correlations,omitempty" jsonschema:"Include correlation analysis (default: true)"`
	IncludeInsights     bool   `json:"include_insights,omitempty" jsonschema:"Include insights (default: true)"`
	MaxInsights         int    `json:"max_insights,omitempty" jsonschema:"Maximum insights to include (default: 10)"`
	Format              string `json:"format,omitempty" jsonschema:"Output format: json, text, markdown (default: json)"`
}

type reflectReportOutput struct {
	ReportID      string                        `json:"report_id" jsonschema:"Report identifier"`
	ProjectID     string                        `json:"project_id" jsonschema:"Project analyzed"`
	GeneratedAt   time.Time                     `json:"generated_at" jsonschema:"Report generation time"`
	PeriodDays    int                           `json:"period_days" jsonschema:"Days analyzed"`
	Summary       string                        `json:"summary" jsonschema:"High-level summary"`
	Statistics    reflection.ReportStatistics   `json:"statistics" jsonschema:"Numerical statistics"`
	PatternCount  int                           `json:"pattern_count" jsonschema:"Number of patterns identified"`
	InsightCount  int                           `json:"insight_count" jsonschema:"Number of insights generated"`
	Format        string                        `json:"format" jsonschema:"Output format used"`
	FormattedText string                        `json:"formatted_text,omitempty" jsonschema:"Formatted report (for text/markdown)"`
}

type reflectAnalyzeInput struct {
	ProjectID     string   `json:"project_id" jsonschema:"required,Project identifier"`
	MinConfidence float64  `json:"min_confidence,omitempty" jsonschema:"Minimum confidence threshold (default: 0.3)"`
	MinFrequency  int      `json:"min_frequency,omitempty" jsonschema:"Minimum pattern frequency (default: 2)"`
	IncludeTags   []string `json:"include_tags,omitempty" jsonschema:"Filter to specific tags"`
	ExcludeTags   []string `json:"exclude_tags,omitempty" jsonschema:"Exclude specific tags"`
	MaxPatterns   int      `json:"max_patterns,omitempty" jsonschema:"Maximum patterns to return (default: 20)"`
}

type reflectAnalyzeOutput struct {
	ProjectID    string               `json:"project_id" jsonschema:"Project analyzed"`
	PatternCount int                  `json:"pattern_count" jsonschema:"Number of patterns found"`
	Patterns     []reflection.Pattern `json:"patterns" jsonschema:"Identified patterns"`
}

func (s *Server) registerReflectionTools() {
	if s.reasoningbankSvc == nil {
		s.logger.Warn("reasoningbank service not configured, skipping reflection tools")
		return
	}

	reporter := reflection.NewReporter(s.reasoningbankSvc)
	analyzer := reflection.NewAnalyzer(s.reasoningbankSvc)

	// reflect_report - Generate a reflection report
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "reflect_report",
		Description: "Generate a self-reflection report analyzing memories and patterns for a project. Returns insights about behavior patterns, success/failure trends, and recommendations.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args reflectReportInput) (*mcp.CallToolResult, reflectReportOutput, error) {
		// Set defaults
		periodDays := args.PeriodDays
		if periodDays <= 0 {
			periodDays = 30
		}

		includePatterns := args.IncludePatterns
		if !includePatterns && !args.IncludeCorrelations && !args.IncludeInsights {
			// If nothing explicitly requested, include everything
			includePatterns = true
		}

		includeCorrelations := args.IncludeCorrelations
		if !args.IncludeCorrelations && !args.IncludePatterns && !args.IncludeInsights {
			includeCorrelations = true
		}

		includeInsights := args.IncludeInsights
		if !args.IncludeInsights && !args.IncludePatterns && !args.IncludeCorrelations {
			includeInsights = true
		}

		maxInsights := args.MaxInsights
		if maxInsights <= 0 {
			maxInsights = 10
		}

		format := args.Format
		if format == "" {
			format = "json"
		}

		// Calculate period
		now := time.Now()
		period := reflection.ReportPeriod{
			Start:       now.AddDate(0, 0, -periodDays),
			End:         now,
			Description: fmt.Sprintf("Last %d days", periodDays),
		}

		opts := reflection.ReportOptions{
			ProjectID:           args.ProjectID,
			Period:              period,
			IncludePatterns:     includePatterns,
			IncludeCorrelations: includeCorrelations,
			IncludeInsights:     includeInsights,
			MaxInsights:         maxInsights,
			Format:              format,
		}

		report, err := reporter.Generate(ctx, opts)
		if err != nil {
			return nil, reflectReportOutput{}, fmt.Errorf("report generation failed: %w", err)
		}

		output := reflectReportOutput{
			ReportID:     report.ID,
			ProjectID:    report.ProjectID,
			GeneratedAt:  report.GeneratedAt,
			PeriodDays:   periodDays,
			Summary:      report.Summary,
			Statistics:   report.Statistics,
			PatternCount: len(report.Patterns),
			InsightCount: len(report.Insights),
			Format:       format,
		}

		// Generate formatted text for non-JSON formats
		if format == "text" || format == "markdown" {
			output.FormattedText = reflection.FormatReport(report, format)
		}

		// Scrub summary if needed
		if s.scrubber != nil {
			output.Summary = s.scrubber.Scrub(output.Summary).Scrubbed
			if output.FormattedText != "" {
				output.FormattedText = s.scrubber.Scrub(output.FormattedText).Scrubbed
			}
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Generated reflection report: %s - %s", report.ID, report.Summary)},
			},
		}, output, nil
	})

	// reflect_analyze - Analyze patterns in memories
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "reflect_analyze",
		Description: "Analyze memories for behavioral patterns. Returns patterns grouped by category (success, failure, recurring, improving, declining) with confidence scores.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args reflectAnalyzeInput) (*mcp.CallToolResult, reflectAnalyzeOutput, error) {
		// Set defaults
		minConfidence := args.MinConfidence
		if minConfidence <= 0 {
			minConfidence = 0.3
		}

		minFrequency := args.MinFrequency
		if minFrequency <= 0 {
			minFrequency = 2
		}

		maxPatterns := args.MaxPatterns
		if maxPatterns <= 0 {
			maxPatterns = 20
		}

		opts := reflection.AnalyzeOptions{
			ProjectID:     args.ProjectID,
			MinConfidence: minConfidence,
			MinFrequency:  minFrequency,
			IncludeTags:   args.IncludeTags,
			ExcludeTags:   args.ExcludeTags,
			MaxPatterns:   maxPatterns,
		}

		patterns, err := analyzer.Analyze(ctx, opts)
		if err != nil {
			return nil, reflectAnalyzeOutput{}, fmt.Errorf("pattern analysis failed: %w", err)
		}

		output := reflectAnalyzeOutput{
			ProjectID:    args.ProjectID,
			PatternCount: len(patterns),
			Patterns:     patterns,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Found %d patterns for project: %s", output.PatternCount, args.ProjectID)},
			},
		}, output, nil
	})
}

// StoreReflectionReport stores a reflection report to disk for later retrieval.
func StoreReflectionReport(report *reflection.ReflectionReport, projectPath string) (string, error) {
	timestamp := report.GeneratedAt.Format("20060102-150405")
	filename := fmt.Sprintf("reflection-%s.json", timestamp)

	reflectionsDir := filepath.Join(projectPath, ".claude", "reflections")
	reportPath := filepath.Join(reflectionsDir, filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report: %w", err)
	}

	// Create the reflections directory if it doesn't exist
	if err := os.MkdirAll(reflectionsDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create reflections directory: %w", err)
	}

	// Write the report file with restrictive permissions
	if err := os.WriteFile(reportPath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write report: %w", err)
	}

	return reportPath, nil
}
