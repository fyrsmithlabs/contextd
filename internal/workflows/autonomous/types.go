// Package autonomous provides Temporal workflows for autonomous AI development teams.
//
// This package implements an autonomous AI development team that handles the complete
// development cycle from GitHub issue to production-ready pull request, using:
// - Temporal for durable workflow orchestration
// - Go activities for agent crew execution
// - Contextd MCP for shared learning via ReasoningBank and context folding
//
// See docs/plans/2025-12-27-autonomous-dev-team-design.md for architecture details.
package autonomous

import (
	"context"
	"time"
)

// Agent represents an AI agent that performs specific tasks.
type Agent interface {
	// Execute runs the agent with given input and returns output.
	Execute(ctx context.Context, input AgentInput) (AgentOutput, error)

	// Name returns the agent's name for logging and observability.
	Name() string
}

// AgentInput provides context and configuration for agent execution.
type AgentInput struct {
	// Task describes what the agent should do
	Task string

	// Context provides additional context (feature spec, code analysis, etc.)
	Context map[string]interface{}

	// MCPClient is the Contextd MCP client for ReasoningBank and collections
	MCPClient MCPClient

	// CollectionName is the short-lived collection for this feature
	CollectionName string

	// ProjectPath is the local path to the project repository
	ProjectPath string

	// IssueNumber is the GitHub issue number being worked on
	IssueNumber int
}

// AgentOutput contains the results of agent execution.
type AgentOutput struct {
	// Result is the primary output (can be any type)
	Result interface{}

	// Artifacts are files/data produced by the agent
	Artifacts []Artifact

	// Metrics tracks agent performance
	Metrics AgentMetrics

	// Error indicates if the agent encountered an error
	Error error
}

// Artifact represents a file or data produced by an agent.
type Artifact struct {
	Type     string                 // "code", "test", "doc", "report"
	Path     string                 // File path relative to project root
	Content  string                 // File content or data
	Metadata map[string]interface{} // Additional metadata
}

// AgentMetrics tracks performance and behavior of agent execution.
type AgentMetrics struct {
	Duration      time.Duration // How long the agent took
	TokensUsed    int           // LLM tokens consumed
	MemoriesUsed  int           // ReasoningBank memories retrieved
	MemoriesAdded int           // New memories recorded
	FilesModified int           // Files created or edited
	TestsRun      int           // Tests executed
	TestsPassed   int           // Tests that passed
}

// MCPClient defines the interface for Contextd MCP operations.
// This wraps the actual MCP client to make testing easier.
type MCPClient interface {
	// ReasoningBank operations
	MemorySearch(ctx context.Context, projectID, query string, limit int) ([]Memory, error)
	MemoryRecord(ctx context.Context, memory Memory) (string, error)
	MemoryOutcome(ctx context.Context, memoryID string, succeeded bool) error

	// Short-lived collection operations
	RepositoryIndex(ctx context.Context, path, tenantID string) (*IndexResult, error)
	RepositorySearch(ctx context.Context, query, collectionName string, limit int) ([]SearchResult, error)
	CollectionDelete(ctx context.Context, collectionName string) error

	// Checkpoint operations
	CheckpointSave(ctx context.Context, checkpoint Checkpoint) (string, error)
	CheckpointResume(ctx context.Context, checkpointID, level string) (*CheckpointState, error)
}

// Memory represents a ReasoningBank memory entry.
type Memory struct {
	ID        string
	Title     string
	Content   string
	Outcome   string   // "success" or "failure"
	Tags      []string
	Timestamp time.Time
}

// IndexResult contains the result of repository indexing.
type IndexResult struct {
	CollectionName string
	FilesIndexed   int
	Timestamp      time.Time
}

// SearchResult represents a semantic search result.
type SearchResult struct {
	FilePath string
	Content  string
	Score    float64
	Metadata map[string]interface{}
}

// Checkpoint represents a workflow checkpoint for recovery.
type Checkpoint struct {
	SessionID   string
	TenantID    string
	ProjectPath string
	Name        string
	Description string
	Summary     string
	Context     string
	FullState   string
	TokenCount  int
	Threshold   float64
	AutoCreated bool
}

// CheckpointState represents the state loaded from a checkpoint.
type CheckpointState struct {
	Summary   string
	Context   string
	FullState string
}

// FeatureDevelopmentInput provides input for the feature development workflow.
type FeatureDevelopmentInput struct {
	// GitHub issue information
	IssueNumber int
	IssueTitle  string
	IssueBody   string
	Repository  string // "owner/repo"

	// Project information
	ProjectPath string
	TenantID    string

	// Configuration
	SkipUsageTests     bool // Skip usage test generation (for testing)
	SkipBenchmarks     bool // Skip performance benchmarks
	SkipSecurityScan   bool // Skip security scanning
	SkipConsensus      bool // Skip consensus review
	AllowParallel      bool // Allow parallel feature development
	MaxParallelFeatures int // Max concurrent features (default: 1)
}

// FeatureDevelopmentResult contains the result of the feature development workflow.
type FeatureDevelopmentResult struct {
	// Workflow status
	Success   bool
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration

	// Phase results
	AnalysisResult       *AnalysisResult
	ImplementationResult *ImplementationResult
	QualityResult        *QualityResult
	ReviewResult         *ReviewResult

	// Outputs
	PRNumber   int
	PRURL      string
	BranchName string

	// Metrics
	TotalTokens      int
	MemoriesRetrieved int
	MemoriesRecorded int
	FilesModified    int
	TestsAdded       int
	CommitsCreated   int

	// Errors (if any)
	Errors []string
}

// AnalysisResult contains output from the Analysis crew.
type AnalysisResult struct {
	// Feature specification
	FeatureSpec       string
	AcceptanceCriteria []string
	EdgeCases         []string

	// Architecture plan
	ArchitecturePlan string
	AffectedFiles    []string
	NewFilesNeeded   []string

	// Research findings
	SimilarPatterns []Memory
	RecommendedApproach string

	// Collection info
	CollectionName string

	Metrics AgentMetrics
}

// ImplementationResult contains output from the Implementation crew.
type ImplementationResult struct {
	// Code artifacts
	CodeFiles    []Artifact
	TestFiles    []Artifact
	DocFiles     []Artifact

	// Git information
	BranchName string
	Commits    []CommitInfo

	// Collection info
	CollectionName string

	Metrics AgentMetrics
}

// CommitInfo represents a git commit.
type CommitInfo struct {
	SHA     string
	Message string
	Files   []string
}

// QualityResult contains output from the Quality crew.
type QualityResult struct {
	// Usage tests
	UsageTestFiles []Artifact
	UsageTestsPassed bool
	EdgeCasesFound   []string

	// Performance benchmarks
	BenchmarkResults []BenchmarkResult
	RegressionDetected bool

	// Security scan
	SecurityReport SecurityReport
	VulnerabilitiesFound bool

	// Collection info
	CollectionName string

	Metrics AgentMetrics
}

// BenchmarkResult represents a performance benchmark result.
type BenchmarkResult struct {
	Name       string
	Duration   time.Duration
	Iterations int
	Baseline   time.Duration
	Regression float64 // Percentage change from baseline
}

// SecurityReport contains security scan results.
type SecurityReport struct {
	Vulnerabilities  []Vulnerability
	DependencyIssues []DependencyIssue
	SecretsFound     []string
	Passed           bool
}

// Vulnerability represents a security vulnerability found.
type Vulnerability struct {
	Severity    string // "critical", "high", "medium", "low"
	Description string
	FilePath    string
	LineNumber  int
	Fix         string
}

// DependencyIssue represents a dependency security issue.
type DependencyIssue struct {
	Dependency  string
	Version     string
	Severity    string
	Description string
	Fix         string
}

// ReviewResult contains output from the Review & Ship crew.
type ReviewResult struct {
	// Technical review
	TechnicalReviews []TechnicalReview
	TechnicalApproved bool

	// UX persona validation
	PersonaReviews   []PersonaReview
	PersonaApproved  bool

	// Overall consensus
	ConsensusReached bool
	Approved         bool
	Reason           string

	// PR information
	PRNumber   int
	PRURL      string
	PRBody     string

	// Collection info
	CollectionName string

	Metrics AgentMetrics
}

// TechnicalReview represents feedback from a technical reviewer.
type TechnicalReview struct {
	Reviewer string // "code", "architecture", "security"
	Approved bool
	Comments []string
	Issues   []ReviewIssue
}

// PersonaReview represents feedback from a UX persona validator.
type PersonaReview struct {
	Persona          string // "marcus", "sarah", "alex", "jordan"
	Approved         bool
	UXBreakingChanges bool
	Comments         []string
	Issues           []ReviewIssue
}

// ReviewIssue represents a specific issue found during review.
type ReviewIssue struct {
	Severity    string // "blocking", "major", "minor"
	Category    string // "code-quality", "architecture", "security", "ux"
	Description string
	FilePath    string
	LineNumber  int
	Suggestion  string
}
