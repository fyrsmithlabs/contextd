// Package framework provides the integration test framework for contextd.
package framework

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Suite A: Policy Retrieval Tests
//
// This file implements Suite A from the integration test framework design:
// Policy Compliance tests that validate recorded policies are retrieved
// when relevant context arises.
//
// Test Coverage:
//
// A.1: TDD Policy Enforcement
//   - Developer records a TDD policy memory
//   - Later searches for development practices
//   - Verifies policy is retrieved with confidence >= 0.7
//   - Validates policy content contains TDD-specific patterns
//
// A.2: Conventional Commits Policy
//   - Developer records conventional commits policy
//   - Searches for commit message guidance
//   - Verifies policy content matches expected commit format patterns
//
// A.3: No Secrets Policy
//   - Developer records "never commit secrets" policy
//   - Searches for security/credentials guidance
//   - Verifies policy retrieval and content validation
//   - Additional test for comprehensive preventive/detective/corrective controls
//
// A.4: Cross-Developer Policy Sharing (bonus test)
//   - Dev A records a code review policy
//   - Dev B searches and retrieves the same policy
//   - Validates shared knowledge in team scenarios
//
// Design Notes:
//
// - Uses SharedStore with mockVectorStore for deterministic test behavior
// - mockVectorStore returns all documents, simulating perfect semantic search
// - In production, real embeddings would provide semantic similarity matching
// - Tests use AssertionSet framework for structured validation
// - Assertions include binary (pass/fail), threshold (numeric), and behavioral (pattern-based)
//
// See: docs/plans/2025-12-10-integration-test-framework-design.md

// TestSuiteA_Policy tests policy retrieval and enforcement scenarios.
// These tests validate that recorded policies are retrieved when relevant
// context arises, ensuring team knowledge is applied consistently.

// TestSuiteA_Policy_TDDEnforcement tests that a recorded TDD policy is
// retrieved when a developer starts new work.
//
// Test A.1: TDD Policy Enforcement
// - Developer records a TDD policy memory
// - Later, when starting new work, search should retrieve the TDD policy
// - Verify the policy is found with confidence >= 0.7
func TestSuiteA_Policy_TDDEnforcement(t *testing.T) {
	t.Run("retrieves TDD policy when starting new feature work", func(t *testing.T) {
		// Setup: Create shared store for better test reliability
		// The mockVectorStore returns all documents, simulating perfect semantic search
		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "test_project_tdd",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		// Setup: Create developer and start contextd
		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-tdd-test",
			TenantID:  "test-tenant",
			ProjectID: "test_project_tdd",
		}, sharedStore)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Step 1: Developer records TDD policy
		policyMemoryID, err := dev.RecordMemory(ctx, MemoryRecord{
			Title: "Always use TDD",
			Content: "When implementing new features, always write a failing test first, " +
				"then implement the minimum code to pass, then refactor. " +
				"Never write implementation before tests.",
			Outcome: "success",
			Tags:    []string{"policy", "tdd", "development-practice"},
		})
		require.NoError(t, err, "failed to record TDD policy")
		assert.NotEmpty(t, policyMemoryID, "policy memory ID should not be empty")

		// Step 2: Later, developer searches for guidance on new feature work
		// Simulate: "I'm starting a new feature, what development practices should I follow?"
		// Note: With mockVectorStore, any query will return all memories in the collection
		// In production, semantic search would find relevant policies based on embedding similarity
		results, err := dev.SearchMemory(ctx, "TDD development practices", 5)
		require.NoError(t, err, "failed to search for TDD policy")

		// Create session result for assertion evaluation
		sessionResult := &SessionResult{
			Developer: DeveloperConfig{
				ID:        dev.ID(),
				TenantID:  dev.TenantID(),
				ProjectID: "test_project_tdd",
			},
			MemoryIDs:     []string{policyMemoryID},
			SearchResults: [][]MemoryResult{results},
			Errors:        []string{},
		}

		// Assertions
		assertions := AssertionSet{
			Binary: []BinaryAssertion{
				{
					Check:  "search_has_results",
					Method: "memory_search",
				},
			},
			Threshold: []ThresholdAssertion{
				{
					Check:     "confidence",
					Method:    "first_result",
					Threshold: 0.7,
					Operator:  ">=",
				},
			},
			Behavioral: []BehavioralAssertion{
				{
					Check:  "content_pattern",
					Method: "regex_match",
					Patterns: []string{
						"(?i)test.*first",
						"(?i)TDD",
						"(?i)failing test",
					},
				},
			},
		}

		// Evaluate all assertions
		assertionResults := EvaluateAssertionSet(assertions, sessionResult)

		// Check if all assertions passed
		if !AllPassed(assertionResults) {
			t.Log("Assertion failures:")
			for _, ar := range FailedAssertions(assertionResults) {
				t.Logf("  - %s: %s", ar.Check, ar.Message)
			}
			t.Fatalf("Assertions failed: %s", SummaryMessage(assertionResults))
		}

		t.Logf("All assertions passed: %s", SummaryMessage(assertionResults))
	})
}

// TestSuiteA_Policy_ConventionalCommits tests that a recorded conventional
// commits policy is retrieved when developers work with git commits.
//
// Test A.2: Conventional Commits Policy
// - Developer records a conventional commits policy
// - Search for commit-related guidance should retrieve it
// - Verify policy content matches expected patterns
func TestSuiteA_Policy_ConventionalCommits(t *testing.T) {
	t.Run("retrieves conventional commits policy for commit guidance", func(t *testing.T) {
		// Setup: Create shared store for better test reliability
		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "test_project_commits",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		// Setup: Create developer and start contextd
		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-commits-test",
			TenantID:  "test-tenant",
			ProjectID: "test_project_commits",
		}, sharedStore)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Step 1: Developer records conventional commits policy
		policyMemoryID, err := dev.RecordMemory(ctx, MemoryRecord{
			Title: "Use conventional commits",
			Content: "All commit messages must follow conventional commits format: " +
				"type(scope): description. " +
				"Types include: feat, fix, docs, refactor, test, chore. " +
				"Example: 'feat(auth): add login endpoint'",
			Outcome: "success",
			Tags:    []string{"policy", "git", "commits", "conventional-commits"},
		})
		require.NoError(t, err, "failed to record conventional commits policy")
		assert.NotEmpty(t, policyMemoryID, "policy memory ID should not be empty")

		// Step 2: Developer searches for commit message guidance
		// Simulate: "How should I format my commit messages?"
		// Note: With mockVectorStore, any query will return all memories
		results, err := dev.SearchMemory(ctx, "conventional commits", 5)
		require.NoError(t, err, "failed to search for commit policy")

		// Create session result for assertion evaluation
		sessionResult := &SessionResult{
			Developer: DeveloperConfig{
				ID:        dev.ID(),
				TenantID:  dev.TenantID(),
				ProjectID: "test_project_commits",
			},
			MemoryIDs:     []string{policyMemoryID},
			SearchResults: [][]MemoryResult{results},
			Errors:        []string{},
		}

		// Assertions - includes threshold for confidence >= 0.7
		assertions := AssertionSet{
			Binary: []BinaryAssertion{
				{
					Check:  "search_has_results",
					Method: "memory_search",
				},
			},
			Threshold: []ThresholdAssertion{
				{
					Check:     "confidence",
					Method:    "first_result",
					Threshold: 0.7,
					Operator:  ">=",
				},
			},
			Behavioral: []BehavioralAssertion{
				{
					Check:  "content_pattern",
					Method: "regex_match",
					Patterns: []string{
						"(?i)conventional.*commit",
						"(?i)feat|fix|docs|refactor",
						"(?i)type.*scope.*description",
					},
				},
			},
		}

		// Evaluate all assertions
		assertionResults := EvaluateAssertionSet(assertions, sessionResult)

		// Check if all assertions passed
		if !AllPassed(assertionResults) {
			t.Log("Assertion failures:")
			for _, ar := range FailedAssertions(assertionResults) {
				t.Logf("  - %s: %s", ar.Check, ar.Message)
			}
			t.Fatalf("Assertions failed: %s", SummaryMessage(assertionResults))
		}

		t.Logf("All assertions passed: %s", SummaryMessage(assertionResults))

		// Additional verification: Check that retrieved policy contains expected content
		if len(results) > 0 {
			firstResult := results[0]
			assert.Contains(t, firstResult.Content, "conventional commits",
				"retrieved policy should mention conventional commits")
			assert.Contains(t, firstResult.Content, "feat",
				"retrieved policy should mention 'feat' commit type")
			assert.GreaterOrEqual(t, firstResult.Confidence, 0.7,
				"policy confidence should be >= 0.7 (MinConfidence threshold)")
			t.Logf("Retrieved policy: %s (confidence: %.2f)",
				firstResult.Title, firstResult.Confidence)
		}
	})
}

// TestSuiteA_Policy_NoSecrets tests that a "never commit secrets" policy
// is retrieved when relevant security context arises.
//
// Test A.3: No Secrets Policy
// - Developer records a "never commit secrets" policy
// - Search should retrieve this when relevant context arises
// - Test the policy retrieval mechanism
func TestSuiteA_Policy_NoSecrets(t *testing.T) {
	t.Run("retrieves no secrets policy for security guidance", func(t *testing.T) {
		// Setup: Create shared store for better test reliability
		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "test_project_secrets",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		// Setup: Create developer and start contextd
		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-secrets-test",
			TenantID:  "test-tenant",
			ProjectID: "test_project_secrets",
		}, sharedStore)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Step 1: Developer records no secrets policy
		policyMemoryID, err := dev.RecordMemory(ctx, MemoryRecord{
			Title: "Never commit secrets",
			Content: "Never commit .env files, API keys, passwords, or any credentials to git. " +
				"Always ensure .env files are in .gitignore. " +
				"Use environment variables or secret management systems for sensitive data. " +
				"Review changes before committing to catch accidental secret exposure.",
			Outcome: "success",
			Tags:    []string{"policy", "security", "secrets", "git"},
		})
		require.NoError(t, err, "failed to record no secrets policy")
		assert.NotEmpty(t, policyMemoryID, "policy memory ID should not be empty")

		// Step 2: Developer searches for guidance on handling sensitive data
		// Simulate: "How should I handle API keys and credentials in my code?"
		// Note: With mockVectorStore, any query will return all memories
		results, err := dev.SearchMemory(ctx, "secrets security credentials", 5)
		require.NoError(t, err, "failed to search for secrets policy")

		// Create session result for assertion evaluation
		sessionResult := &SessionResult{
			Developer: DeveloperConfig{
				ID:        dev.ID(),
				TenantID:  dev.TenantID(),
				ProjectID: "test_project_secrets",
			},
			MemoryIDs:     []string{policyMemoryID},
			SearchResults: [][]MemoryResult{results},
			Errors:        []string{},
		}

		// Assertions - includes threshold for confidence >= 0.7
		assertions := AssertionSet{
			Binary: []BinaryAssertion{
				{
					Check:  "search_has_results",
					Method: "memory_search",
				},
				{
					Check:  "no_errors",
					Method: "session_completion",
				},
			},
			Threshold: []ThresholdAssertion{
				{
					Check:     "confidence",
					Method:    "first_result",
					Threshold: 0.7,
					Operator:  ">=",
				},
			},
			Behavioral: []BehavioralAssertion{
				{
					Check:  "content_pattern",
					Method: "regex_match",
					Patterns: []string{
						"(?i)never.*commit.*secret",
						"(?i)\\.env|API.*key|password|credential",
						"(?i)\\.gitignore",
					},
					NegativePatterns: []string{
						// Ensure the policy doesn't accidentally contain actual secrets
						"sk-[a-zA-Z0-9]{32,}",
						"AKIA[0-9A-Z]{16}",
					},
				},
			},
		}

		// Evaluate all assertions
		assertionResults := EvaluateAssertionSet(assertions, sessionResult)

		// Check if all assertions passed
		if !AllPassed(assertionResults) {
			t.Log("Assertion failures:")
			for _, ar := range FailedAssertions(assertionResults) {
				t.Logf("  - %s: %s", ar.Check, ar.Message)
			}
			t.Fatalf("Assertions failed: %s", SummaryMessage(assertionResults))
		}

		t.Logf("All assertions passed: %s", SummaryMessage(assertionResults))

		// Additional verification: Check that policy is actionable
		if len(results) > 0 {
			firstResult := results[0]
			assert.Contains(t, firstResult.Content, "gitignore",
				"policy should mention .gitignore as a concrete action")
			assert.NotContains(t, firstResult.Content, "sk-",
				"policy should not contain actual API keys")
			assert.GreaterOrEqual(t, firstResult.Confidence, 0.7,
				"security policy confidence should be >= 0.7 (MinConfidence threshold)")
			t.Logf("Retrieved security policy: %s (confidence: %.2f)",
				firstResult.Title, firstResult.Confidence)
		}
	})

	t.Run("policy includes both preventive and detective controls", func(t *testing.T) {
		// This test verifies that the policy is comprehensive
		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "test_project_secrets_comprehensive",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-secrets-comprehensive",
			TenantID:  "test-tenant",
			ProjectID: "test_project_secrets_comprehensive",
		}, sharedStore)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Record a comprehensive secrets policy
		_, err = dev.RecordMemory(ctx, MemoryRecord{
			Title: "Comprehensive secrets management policy",
			Content: "PREVENTIVE: Always add .env to .gitignore before first commit. " +
				"Use environment variables for all secrets. " +
				"DETECTIVE: Run git diff before committing to review for accidental secrets. " +
				"Use pre-commit hooks with gitleaks to scan for secrets. " +
				"CORRECTIVE: If a secret is committed, rotate it immediately and use git filter-repo to remove from history.",
			Outcome: "success",
			Tags:    []string{"policy", "security", "secrets", "comprehensive"},
		})
		require.NoError(t, err)

		// Search for the policy
		results, err := dev.SearchMemory(ctx, "prevent detect secrets management", 5)
		require.NoError(t, err)

		// Verify the policy covers multiple aspects
		if len(results) > 0 {
			content := results[0].Content
			assert.Contains(t, content, "PREVENTIVE", "should have preventive controls")
			assert.Contains(t, content, "DETECTIVE", "should have detective controls")
			assert.Contains(t, content, "CORRECTIVE", "should have corrective actions")
		}
	})
}

// TestSuiteA_Policy_CrossDeveloperPolicySharing tests that policies
// recorded by one developer are available to another developer in the
// same project (when using shared store).
func TestSuiteA_Policy_CrossDeveloperPolicySharing(t *testing.T) {
	t.Run("dev B retrieves policy recorded by dev A", func(t *testing.T) {
		// Setup: Create shared store
		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "shared_test_project",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Dev A records a policy
		devA, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-a-policy",
			TenantID:  "test-tenant",
			ProjectID: "shared_test_project",
		}, sharedStore)
		require.NoError(t, err)

		err = devA.StartContextd(ctx)
		require.NoError(t, err)
		defer devA.StopContextd(ctx)

		policyID, err := devA.RecordMemory(ctx, MemoryRecord{
			Title:   "Code review policy",
			Content: "All PRs must have at least one approval before merging. Run linter and tests before requesting review.",
			Outcome: "success",
			Tags:    []string{"policy", "code-review", "team"},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, policyID)

		// Dev B searches for the policy
		devB, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-b-policy",
			TenantID:  "test-tenant",
			ProjectID: "shared_test_project",
		}, sharedStore)
		require.NoError(t, err)

		err = devB.StartContextd(ctx)
		require.NoError(t, err)
		defer devB.StopContextd(ctx)

		results, err := devB.SearchMemory(ctx, "code review pull request approval", 5)
		require.NoError(t, err)

		// Verify Dev B can retrieve Dev A's policy
		sessionResult := &SessionResult{
			SearchResults: [][]MemoryResult{results},
		}

		// Assertions - includes threshold for confidence >= 0.7
		assertions := AssertionSet{
			Binary: []BinaryAssertion{
				{
					Check:  "search_has_results",
					Method: "memory_search",
				},
			},
			Threshold: []ThresholdAssertion{
				{
					Check:     "confidence",
					Method:    "first_result",
					Threshold: 0.7,
					Operator:  ">=",
				},
			},
			Behavioral: []BehavioralAssertion{
				{
					Check:  "content_pattern",
					Method: "regex_match",
					Patterns: []string{
						"(?i)approval",
						"(?i)review",
					},
				},
			},
		}

		assertionResults := EvaluateAssertionSet(assertions, sessionResult)
		assert.True(t, AllPassed(assertionResults),
			"Dev B should be able to retrieve Dev A's policy: %s",
			SummaryMessage(assertionResults))

		// Additional verification: explicitly check confidence threshold
		if len(results) > 0 {
			assert.GreaterOrEqual(t, results[0].Confidence, 0.7,
				"cross-developer policy confidence should be >= 0.7")
			t.Logf("Cross-developer policy retrieved: %s (confidence: %.2f)",
				results[0].Title, results[0].Confidence)
		}
	})
}
