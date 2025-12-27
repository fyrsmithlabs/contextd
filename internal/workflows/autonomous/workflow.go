package autonomous

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// FeatureDevelopmentWorkflow implements the autonomous AI development workflow.
//
// This workflow orchestrates the complete development cycle:
//  1. Analysis Crew - Extract requirements, plan architecture, research patterns
//  2. Implementation Crew - Write code, tests, documentation
//  3. Quality Crew - Run usage tests, benchmarks, security scans
//  4. Review & Ship Crew - Technical review + UX persona validation
//
// The workflow is durable and can handle:
// - Long-running features (hours/days)
// - Worker crashes (automatic recovery via Temporal)
// - Checkpointing (save state via Contextd MCP)
//
// See docs/plans/2025-12-27-autonomous-dev-team-design.md for architecture.
func FeatureDevelopmentWorkflow(ctx workflow.Context, input FeatureDevelopmentInput) (*FeatureDevelopmentResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting feature development workflow",
		"issue", input.IssueNumber,
		"title", input.IssueTitle,
		"repository", input.Repository,
	)

	result := &FeatureDevelopmentResult{
		StartTime: workflow.Now(ctx),
	}

	// Configure activity options with retries
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Phase 1: Analysis
	logger.Info("Starting analysis phase")
	var analysisResult AnalysisResult
	err := workflow.ExecuteActivity(ctx, AnalyzeFeature, AnalyzeFeatureInput{
		IssueNumber: input.IssueNumber,
		IssueTitle:  input.IssueTitle,
		IssueBody:   input.IssueBody,
		ProjectPath: input.ProjectPath,
		TenantID:    input.TenantID,
	}).Get(ctx, &analysisResult)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Analysis phase failed: %v", err))
		return result, err
	}
	result.AnalysisResult = &analysisResult
	result.MemoriesRetrieved += len(analysisResult.SimilarPatterns)
	logger.Info("Analysis phase complete",
		"affected_files", len(analysisResult.AffectedFiles),
		"memories_retrieved", len(analysisResult.SimilarPatterns),
	)

	// Phase 2: Implementation
	logger.Info("Starting implementation phase")
	var implResult ImplementationResult
	err = workflow.ExecuteActivity(ctx, ImplementFeature, ImplementFeatureInput{
		FeatureSpec:        analysisResult.FeatureSpec,
		ArchitecturePlan:   analysisResult.ArchitecturePlan,
		AffectedFiles:      analysisResult.AffectedFiles,
		NewFilesNeeded:     analysisResult.NewFilesNeeded,
		SimilarPatterns:    analysisResult.SimilarPatterns,
		ProjectPath:        input.ProjectPath,
		TenantID:           input.TenantID,
		IssueNumber:        input.IssueNumber,
		CollectionName:     analysisResult.CollectionName,
	}).Get(ctx, &implResult)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Implementation phase failed: %v", err))
		return result, err
	}
	result.ImplementationResult = &implResult
	result.FilesModified = len(implResult.CodeFiles) + len(implResult.TestFiles)
	result.CommitsCreated = len(implResult.Commits)
	result.BranchName = implResult.BranchName
	logger.Info("Implementation phase complete",
		"code_files", len(implResult.CodeFiles),
		"test_files", len(implResult.TestFiles),
		"commits", len(implResult.Commits),
	)

	// Phase 3: Quality Assurance
	logger.Info("Starting quality assurance phase")
	var qaResult QualityResult
	err = workflow.ExecuteActivity(ctx, ValidateQuality, ValidateQualityInput{
		BranchName:         implResult.BranchName,
		CodeFiles:          implResult.CodeFiles,
		TestFiles:          implResult.TestFiles,
		ProjectPath:        input.ProjectPath,
		TenantID:           input.TenantID,
		IssueNumber:        input.IssueNumber,
		CollectionName:     implResult.CollectionName,
		SkipUsageTests:     input.SkipUsageTests,
		SkipBenchmarks:     input.SkipBenchmarks,
		SkipSecurityScan:   input.SkipSecurityScan,
	}).Get(ctx, &qaResult)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Quality assurance phase failed: %v", err))
		return result, err
	}
	result.QualityResult = &qaResult
	result.TestsAdded = len(qaResult.UsageTestFiles)

	// Check if quality gates passed
	if !qaResult.UsageTestsPassed {
		result.Success = false
		result.Errors = append(result.Errors, "Usage tests failed")
		// Post feedback to issue
		_ = workflow.ExecuteActivity(ctx, PostReviewFeedback, PostReviewFeedbackInput{
			Repository:  input.Repository,
			IssueNumber: input.IssueNumber,
			Feedback:    "Quality assurance failed: usage tests did not pass",
			Details:     qaResult,
		}).Get(ctx, nil)
		return result, fmt.Errorf("usage tests failed")
	}
	if qaResult.RegressionDetected {
		result.Success = false
		result.Errors = append(result.Errors, "Performance regression detected")
		_ = workflow.ExecuteActivity(ctx, PostReviewFeedback, PostReviewFeedbackInput{
			Repository:  input.Repository,
			IssueNumber: input.IssueNumber,
			Feedback:    "Quality assurance failed: performance regression detected",
			Details:     qaResult,
		}).Get(ctx, nil)
		return result, fmt.Errorf("performance regression detected")
	}
	if qaResult.VulnerabilitiesFound {
		result.Success = false
		result.Errors = append(result.Errors, "Security vulnerabilities found")
		_ = workflow.ExecuteActivity(ctx, PostReviewFeedback, PostReviewFeedbackInput{
			Repository:  input.Repository,
			IssueNumber: input.IssueNumber,
			Feedback:    "Quality assurance failed: security vulnerabilities found",
			Details:     qaResult,
		}).Get(ctx, nil)
		return result, fmt.Errorf("security vulnerabilities found")
	}
	logger.Info("Quality assurance phase complete",
		"usage_tests_passed", qaResult.UsageTestsPassed,
		"regression_detected", qaResult.RegressionDetected,
		"vulnerabilities_found", qaResult.VulnerabilitiesFound,
	)

	// Phase 4: Consensus Review
	if !input.SkipConsensus {
		logger.Info("Starting consensus review phase")
		var reviewResult ReviewResult
		err = workflow.ExecuteActivity(ctx, ConsensusReview, ConsensusReviewInput{
			BranchName:     implResult.BranchName,
			CodeFiles:      implResult.CodeFiles,
			TestFiles:      implResult.TestFiles,
			UsageTestFiles: qaResult.UsageTestFiles,
			ProjectPath:    input.ProjectPath,
			TenantID:       input.TenantID,
			IssueNumber:    input.IssueNumber,
			IssueTitle:     input.IssueTitle,
			IssueBody:      input.IssueBody,
			Repository:     input.Repository,
			CollectionName: qaResult.CollectionName,
		}).Get(ctx, &reviewResult)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("Consensus review phase failed: %v", err))
			return result, err
		}
		result.ReviewResult = &reviewResult

		// Check if consensus was reached
		if !reviewResult.Approved {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("Consensus review rejected: %s", reviewResult.Reason))
			// Feedback already posted by ConsensusReview activity
			return result, fmt.Errorf("consensus review rejected: %s", reviewResult.Reason)
		}

		result.PRNumber = reviewResult.PRNumber
		result.PRURL = reviewResult.PRURL
		logger.Info("Consensus review phase complete",
			"technical_approved", reviewResult.TechnicalApproved,
			"persona_approved", reviewResult.PersonaApproved,
			"pr_number", reviewResult.PRNumber,
		)
	} else {
		logger.Info("Skipping consensus review (skip_consensus=true)")
	}

	// Phase 5: Cleanup
	logger.Info("Starting cleanup phase")
	err = workflow.ExecuteActivity(ctx, Cleanup, CleanupInput{
		CollectionNames: []string{
			analysisResult.CollectionName,
			implResult.CollectionName,
			qaResult.CollectionName,
		},
		IssueNumber:    input.IssueNumber,
		Repository:     input.Repository,
		PRNumber:       result.PRNumber,
		PRURL:          result.PRURL,
		AnalysisResult: &analysisResult,
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Cleanup phase failed (non-fatal)", "error", err)
		// Cleanup failure is non-fatal
	}

	// Calculate final metrics
	result.Success = true
	result.EndTime = workflow.Now(ctx)
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.TotalTokens = analysisResult.Metrics.TokensUsed +
		implResult.Metrics.TokensUsed +
		qaResult.Metrics.TokensUsed
	if result.ReviewResult != nil {
		result.TotalTokens += result.ReviewResult.Metrics.TokensUsed
	}
	result.MemoriesRecorded = analysisResult.Metrics.MemoriesAdded +
		implResult.Metrics.MemoriesAdded +
		qaResult.Metrics.MemoriesAdded

	logger.Info("Feature development workflow complete",
		"duration", result.Duration,
		"pr_number", result.PRNumber,
		"pr_url", result.PRURL,
		"files_modified", result.FilesModified,
		"tests_added", result.TestsAdded,
		"total_tokens", result.TotalTokens,
	)

	return result, nil
}

// MultiFeatureWorkflow handles multiple features with optional parallelism.
//
// This workflow allows parallel feature development with configurable limits.
// When AllowParallel is false, features are processed sequentially.
// When AllowParallel is true, up to MaxParallelFeatures run concurrently.
func MultiFeatureWorkflow(ctx workflow.Context, inputs []FeatureDevelopmentInput, config FeatureDevelopmentConfig) ([]*FeatureDevelopmentResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting multi-feature workflow",
		"feature_count", len(inputs),
		"allow_parallel", config.AllowParallel,
		"max_parallel", config.MaxParallelFeatures,
	)

	results := make([]*FeatureDevelopmentResult, len(inputs))

	if !config.AllowParallel || config.MaxParallelFeatures <= 1 {
		// Sequential execution
		for i, input := range inputs {
			childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf("feature-%d", input.IssueNumber),
			})

			var result FeatureDevelopmentResult
			err := workflow.ExecuteChildWorkflow(childCtx, FeatureDevelopmentWorkflow, input).Get(ctx, &result)
			if err != nil {
				logger.Error("Feature workflow failed", "issue", input.IssueNumber, "error", err)
				results[i] = &FeatureDevelopmentResult{
					Success: false,
					Errors:  []string{err.Error()},
				}
			} else {
				results[i] = &result
			}
		}
		return results, nil
	}

	// Parallel execution with limit
	var futures []workflow.ChildWorkflowFuture
	var futureIndexes []int
	nextIndex := 0

	for nextIndex < len(inputs) {
		// Fill up to max parallel
		for len(futures) < config.MaxParallelFeatures && nextIndex < len(inputs) {
			input := inputs[nextIndex]
			childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf("feature-%d", input.IssueNumber),
			})

			future := workflow.ExecuteChildWorkflow(childCtx, FeatureDevelopmentWorkflow, input)
			futures = append(futures, future)
			futureIndexes = append(futureIndexes, nextIndex)
			nextIndex++
		}

		// Wait for any future to complete
		selector := workflow.NewSelector(ctx)
		for i, future := range futures {
			i := i // capture loop variable
			future := future
			selector.AddFuture(future, func(f workflow.Future) {
				var result FeatureDevelopmentResult
				err := f.Get(ctx, &result)
				if err != nil {
					logger.Error("Feature workflow failed", "index", futureIndexes[i], "error", err)
					results[futureIndexes[i]] = &FeatureDevelopmentResult{
						Success: false,
						Errors:  []string{err.Error()},
					}
				} else {
					results[futureIndexes[i]] = &result
				}
			})
		}
		selector.Select(ctx)

		// Remove completed futures
		newFutures := []workflow.ChildWorkflowFuture{}
		newIndexes := []int{}
		for i, future := range futures {
			if !future.IsReady() {
				newFutures = append(newFutures, future)
				newIndexes = append(newIndexes, futureIndexes[i])
			}
		}
		futures = newFutures
		futureIndexes = newIndexes
	}

	// Wait for remaining futures
	for i, future := range futures {
		var result FeatureDevelopmentResult
		err := future.Get(ctx, &result)
		if err != nil {
			logger.Error("Feature workflow failed", "index", futureIndexes[i], "error", err)
			results[futureIndexes[i]] = &FeatureDevelopmentResult{
				Success: false,
				Errors:  []string{err.Error()},
			}
		} else {
			results[futureIndexes[i]] = &result
		}
	}

	logger.Info("Multi-feature workflow complete", "feature_count", len(inputs))
	return results, nil
}

// FeatureDevelopmentConfig configures multi-feature workflow behavior.
type FeatureDevelopmentConfig struct {
	AllowParallel       bool // Whether to allow parallel feature development
	MaxParallelFeatures int  // Maximum concurrent features (default: 1)
}

// Activity input types

type AnalyzeFeatureInput struct {
	IssueNumber int
	IssueTitle  string
	IssueBody   string
	ProjectPath string
	TenantID    string
}

type ImplementFeatureInput struct {
	FeatureSpec      string
	ArchitecturePlan string
	AffectedFiles    []string
	NewFilesNeeded   []string
	SimilarPatterns  []Memory
	ProjectPath      string
	TenantID         string
	IssueNumber      int
	CollectionName   string
}

type ValidateQualityInput struct {
	BranchName       string
	CodeFiles        []Artifact
	TestFiles        []Artifact
	ProjectPath      string
	TenantID         string
	IssueNumber      int
	CollectionName   string
	SkipUsageTests   bool
	SkipBenchmarks   bool
	SkipSecurityScan bool
}

type ConsensusReviewInput struct {
	BranchName     string
	CodeFiles      []Artifact
	TestFiles      []Artifact
	UsageTestFiles []Artifact
	ProjectPath    string
	TenantID       string
	IssueNumber    int
	IssueTitle     string
	IssueBody      string
	Repository     string
	CollectionName string
}

type CleanupInput struct {
	CollectionNames []string
	IssueNumber     int
	Repository      string
	PRNumber        int
	PRURL           string
	AnalysisResult  *AnalysisResult
}

type PostReviewFeedbackInput struct {
	Repository  string
	IssueNumber int
	Feedback    string
	Details     interface{}
}
