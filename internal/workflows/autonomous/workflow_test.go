package autonomous_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"

	"github.com/fyrsmithlabs/contextd/internal/workflows/autonomous"
)

// TestFeatureDevelopmentWorkflow_Success tests the happy path workflow execution.
func TestFeatureDevelopmentWorkflow_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock activity responses for success case
	env.OnActivity(autonomous.AnalyzeFeature, mock.Anything, mock.Anything).Return(&autonomous.AnalysisResult{
		FeatureSpec:         "Add user authentication",
		AcceptanceCriteria:  []string{"Users can log in", "Sessions persist"},
		EdgeCases:           []string{"Invalid credentials", "Expired sessions"},
		ArchitecturePlan:    "Add auth middleware",
		AffectedFiles:       []string{"internal/http/middleware.go"},
		NewFilesNeeded:      []string{"internal/auth/service.go"},
		SimilarPatterns:     []autonomous.Memory{{Title: "JWT auth pattern"}},
		RecommendedApproach: "Use JWT tokens",
		CollectionName:      "test-collection-analysis",
		Metrics: autonomous.AgentMetrics{
			Duration:   5 * time.Minute,
			TokensUsed: 1000,
		},
	}, nil)

	env.OnActivity(autonomous.ImplementFeature, mock.Anything, mock.Anything).Return(&autonomous.ImplementationResult{
		CodeFiles: []autonomous.Artifact{
			{Type: "code", Path: "internal/auth/service.go", Content: "package auth"},
		},
		TestFiles: []autonomous.Artifact{
			{Type: "test", Path: "internal/auth/service_test.go", Content: "package auth"},
		},
		DocFiles: []autonomous.Artifact{
			{Type: "doc", Path: "README.md", Content: "# Auth"},
		},
		BranchName: "feature/auth-67",
		Commits: []autonomous.CommitInfo{
			{SHA: "abc123", Message: "Add auth service", Files: []string{"internal/auth/service.go"}},
		},
		CollectionName: "test-collection-impl",
		Metrics: autonomous.AgentMetrics{
			Duration:   15 * time.Minute,
			TokensUsed: 3000,
		},
	}, nil)

	env.OnActivity(autonomous.ValidateQuality, mock.Anything, mock.Anything).Return(&autonomous.QualityResult{
		UsageTestFiles: []autonomous.Artifact{
			{Type: "test", Path: "test/usage/auth_test.go", Content: "package usage"},
		},
		UsageTestsPassed: true,
		EdgeCasesFound:   []string{"Concurrent login attempts"},
		BenchmarkResults: []autonomous.BenchmarkResult{
			{Name: "BenchmarkAuth", Duration: 100 * time.Millisecond, Regression: 0.02},
		},
		RegressionDetected: false,
		SecurityReport: autonomous.SecurityReport{
			Vulnerabilities:  []autonomous.Vulnerability{},
			DependencyIssues: []autonomous.DependencyIssue{},
			Passed:           true,
		},
		VulnerabilitiesFound: false,
		CollectionName:       "test-collection-qa",
		Metrics: autonomous.AgentMetrics{
			Duration:   10 * time.Minute,
			TokensUsed: 2000,
		},
	}, nil)

	env.OnActivity(autonomous.ConsensusReview, mock.Anything, mock.Anything).Return(&autonomous.ReviewResult{
		TechnicalReviews: []autonomous.TechnicalReview{
			{Reviewer: "code", Approved: true, Comments: []string{"LGTM"}},
			{Reviewer: "architecture", Approved: true, Comments: []string{"Good design"}},
			{Reviewer: "security", Approved: true, Comments: []string{"No issues"}},
		},
		TechnicalApproved: true,
		PersonaReviews: []autonomous.PersonaReview{
			{Persona: "marcus", Approved: true, UXBreakingChanges: false},
			{Persona: "sarah", Approved: true, UXBreakingChanges: false},
			{Persona: "alex", Approved: true, UXBreakingChanges: false},
			{Persona: "jordan", Approved: true, UXBreakingChanges: false},
		},
		PersonaApproved:  true,
		ConsensusReached: true,
		Approved:         true,
		Reason:           "All reviewers approved",
		PRNumber:         123,
		PRURL:            "https://github.com/owner/repo/pull/123",
		PRBody:           "## Summary\nAdds auth",
		CollectionName:   "test-collection-review",
		Metrics: autonomous.AgentMetrics{
			Duration:   5 * time.Minute,
			TokensUsed: 1500,
		},
	}, nil)

	env.OnActivity(autonomous.Cleanup, mock.Anything, mock.Anything).Return(nil)

	// Execute workflow
	input := autonomous.FeatureDevelopmentInput{
		IssueNumber: 67,
		IssueTitle:  "Add user authentication",
		IssueBody:   "We need user login functionality",
		Repository:  "owner/repo",
		ProjectPath: "/path/to/project",
		TenantID:    "test-tenant",
	}

	env.ExecuteWorkflow(autonomous.FeatureDevelopmentWorkflow, input)

	// Verify workflow succeeded
	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	// Verify result
	var result autonomous.FeatureDevelopmentResult
	err := env.GetWorkflowResult(&result)
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 123, result.PRNumber)
	assert.Equal(t, "https://github.com/owner/repo/pull/123", result.PRURL)
	assert.Equal(t, "feature/auth-67", result.BranchName)
	assert.Equal(t, 2, result.FilesModified) // 1 code + 1 test
	assert.Equal(t, 1, result.TestsAdded)
	assert.Equal(t, 1, result.CommitsCreated)
	assert.Equal(t, 7500, result.TotalTokens) // 1000 + 3000 + 2000 + 1500
}

// TestFeatureDevelopmentWorkflow_UsageTestsFail tests workflow when usage tests fail.
func TestFeatureDevelopmentWorkflow_UsageTestsFail(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock successful analysis and implementation
	env.OnActivity(autonomous.AnalyzeFeature, mock.Anything, mock.Anything).Return(&autonomous.AnalysisResult{
		FeatureSpec:      "Add feature",
		CollectionName:   "test-analysis",
		Metrics:          autonomous.AgentMetrics{TokensUsed: 1000},
	}, nil)

	env.OnActivity(autonomous.ImplementFeature, mock.Anything, mock.Anything).Return(&autonomous.ImplementationResult{
		BranchName:     "feature/test",
		CollectionName: "test-impl",
		Metrics:        autonomous.AgentMetrics{TokensUsed: 2000},
	}, nil)

	// Mock failing quality assurance
	env.OnActivity(autonomous.ValidateQuality, mock.Anything, mock.Anything).Return(&autonomous.QualityResult{
		UsageTestsPassed:     false, // Tests failed!
		RegressionDetected:   false,
		VulnerabilitiesFound: false,
		CollectionName:       "test-qa",
		Metrics:              autonomous.AgentMetrics{TokensUsed: 1500},
	}, nil)

	// Mock posting feedback
	env.OnActivity(autonomous.PostReviewFeedback, mock.Anything, mock.Anything).Return(nil)

	// Execute workflow
	input := autonomous.FeatureDevelopmentInput{
		IssueNumber: 67,
		Repository:  "owner/repo",
		ProjectPath: "/path/to/project",
		TenantID:    "test-tenant",
	}

	env.ExecuteWorkflow(autonomous.FeatureDevelopmentWorkflow, input)

	// Verify workflow completed (but with error)
	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())

	// Verify result shows failure
	var result autonomous.FeatureDevelopmentResult
	_ = env.GetWorkflowResult(&result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Errors, "Usage tests failed")
}

// TestFeatureDevelopmentWorkflow_PerformanceRegression tests workflow when regression detected.
func TestFeatureDevelopmentWorkflow_PerformanceRegression(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnActivity(autonomous.AnalyzeFeature, mock.Anything, mock.Anything).Return(&autonomous.AnalysisResult{
		CollectionName: "test-analysis",
		Metrics:        autonomous.AgentMetrics{TokensUsed: 1000},
	}, nil)

	env.OnActivity(autonomous.ImplementFeature, mock.Anything, mock.Anything).Return(&autonomous.ImplementationResult{
		BranchName:     "feature/test",
		CollectionName: "test-impl",
		Metrics:        autonomous.AgentMetrics{TokensUsed: 2000},
	}, nil)

	env.OnActivity(autonomous.ValidateQuality, mock.Anything, mock.Anything).Return(&autonomous.QualityResult{
		UsageTestsPassed:     true,
		RegressionDetected:   true, // Regression detected!
		VulnerabilitiesFound: false,
		BenchmarkResults: []autonomous.BenchmarkResult{
			{Name: "SlowFunction", Regression: 0.15}, // 15% slower
		},
		CollectionName: "test-qa",
		Metrics:        autonomous.AgentMetrics{TokensUsed: 1500},
	}, nil)

	env.OnActivity(autonomous.PostReviewFeedback, mock.Anything, mock.Anything).Return(nil)

	input := autonomous.FeatureDevelopmentInput{
		IssueNumber: 67,
		Repository:  "owner/repo",
		ProjectPath: "/path/to/project",
		TenantID:    "test-tenant",
	}

	env.ExecuteWorkflow(autonomous.FeatureDevelopmentWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())

	var result autonomous.FeatureDevelopmentResult
	_ = env.GetWorkflowResult(&result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Errors, "Performance regression detected")
}

// TestFeatureDevelopmentWorkflow_SecurityVulnerabilities tests workflow when vulnerabilities found.
func TestFeatureDevelopmentWorkflow_SecurityVulnerabilities(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnActivity(autonomous.AnalyzeFeature, mock.Anything, mock.Anything).Return(&autonomous.AnalysisResult{
		CollectionName: "test-analysis",
		Metrics:        autonomous.AgentMetrics{TokensUsed: 1000},
	}, nil)

	env.OnActivity(autonomous.ImplementFeature, mock.Anything, mock.Anything).Return(&autonomous.ImplementationResult{
		BranchName:     "feature/test",
		CollectionName: "test-impl",
		Metrics:        autonomous.AgentMetrics{TokensUsed: 2000},
	}, nil)

	env.OnActivity(autonomous.ValidateQuality, mock.Anything, mock.Anything).Return(&autonomous.QualityResult{
		UsageTestsPassed:     true,
		RegressionDetected:   false,
		VulnerabilitiesFound: true, // Security issue!
		SecurityReport: autonomous.SecurityReport{
			Vulnerabilities: []autonomous.Vulnerability{
				{Severity: "high", Description: "SQL injection risk"},
			},
			Passed: false,
		},
		CollectionName: "test-qa",
		Metrics:        autonomous.AgentMetrics{TokensUsed: 1500},
	}, nil)

	env.OnActivity(autonomous.PostReviewFeedback, mock.Anything, mock.Anything).Return(nil)

	input := autonomous.FeatureDevelopmentInput{
		IssueNumber: 67,
		Repository:  "owner/repo",
		ProjectPath: "/path/to/project",
		TenantID:    "test-tenant",
	}

	env.ExecuteWorkflow(autonomous.FeatureDevelopmentWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())

	var result autonomous.FeatureDevelopmentResult
	_ = env.GetWorkflowResult(&result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Errors, "Security vulnerabilities found")
}

// TestFeatureDevelopmentWorkflow_ConsensusRejection tests workflow when consensus review rejects.
func TestFeatureDevelopmentWorkflow_ConsensusRejection(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnActivity(autonomous.AnalyzeFeature, mock.Anything, mock.Anything).Return(&autonomous.AnalysisResult{
		CollectionName: "test-analysis",
		Metrics:        autonomous.AgentMetrics{TokensUsed: 1000},
	}, nil)

	env.OnActivity(autonomous.ImplementFeature, mock.Anything, mock.Anything).Return(&autonomous.ImplementationResult{
		BranchName:     "feature/test",
		CollectionName: "test-impl",
		Metrics:        autonomous.AgentMetrics{TokensUsed: 2000},
	}, nil)

	env.OnActivity(autonomous.ValidateQuality, mock.Anything, mock.Anything).Return(&autonomous.QualityResult{
		UsageTestsPassed:     true,
		RegressionDetected:   false,
		VulnerabilitiesFound: false,
		CollectionName:       "test-qa",
		Metrics:              autonomous.AgentMetrics{TokensUsed: 1500},
	}, nil)

	env.OnActivity(autonomous.ConsensusReview, mock.Anything, mock.Anything).Return(&autonomous.ReviewResult{
		TechnicalApproved: true,
		PersonaReviews: []autonomous.PersonaReview{
			{Persona: "marcus", Approved: false, UXBreakingChanges: true}, // UX breaking change!
		},
		PersonaApproved:  false,
		ConsensusReached: false,
		Approved:         false,
		Reason:           "UX breaking change detected by Marcus persona",
		CollectionName:   "test-review",
		Metrics:          autonomous.AgentMetrics{TokensUsed: 1000},
	}, nil)

	input := autonomous.FeatureDevelopmentInput{
		IssueNumber: 67,
		Repository:  "owner/repo",
		ProjectPath: "/path/to/project",
		TenantID:    "test-tenant",
	}

	env.ExecuteWorkflow(autonomous.FeatureDevelopmentWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())

	var result autonomous.FeatureDevelopmentResult
	_ = env.GetWorkflowResult(&result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Errors, "Consensus review rejected")
}

// TestFeatureDevelopmentWorkflow_SkipConsensus tests workflow with consensus skipped.
func TestFeatureDevelopmentWorkflow_SkipConsensus(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnActivity(autonomous.AnalyzeFeature, mock.Anything, mock.Anything).Return(&autonomous.AnalysisResult{
		CollectionName: "test-analysis",
		Metrics:        autonomous.AgentMetrics{TokensUsed: 1000},
	}, nil)

	env.OnActivity(autonomous.ImplementFeature, mock.Anything, mock.Anything).Return(&autonomous.ImplementationResult{
		BranchName:     "feature/test",
		CollectionName: "test-impl",
		Metrics:        autonomous.AgentMetrics{TokensUsed: 2000},
	}, nil)

	env.OnActivity(autonomous.ValidateQuality, mock.Anything, mock.Anything).Return(&autonomous.QualityResult{
		UsageTestsPassed:     true,
		RegressionDetected:   false,
		VulnerabilitiesFound: false,
		CollectionName:       "test-qa",
		Metrics:              autonomous.AgentMetrics{TokensUsed: 1500},
	}, nil)

	// Consensus activity should NOT be called
	env.OnActivity(autonomous.Cleanup, mock.Anything, mock.Anything).Return(nil)

	input := autonomous.FeatureDevelopmentInput{
		IssueNumber:   67,
		Repository:    "owner/repo",
		ProjectPath:   "/path/to/project",
		TenantID:      "test-tenant",
		SkipConsensus: true, // Skip consensus review
	}

	env.ExecuteWorkflow(autonomous.FeatureDevelopmentWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result autonomous.FeatureDevelopmentResult
	err := env.GetWorkflowResult(&result)
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Nil(t, result.ReviewResult) // No review result when skipped
	assert.Equal(t, 0, result.PRNumber) // No PR created when consensus skipped
}
