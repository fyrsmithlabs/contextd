// Package framework provides the integration test framework for contextd.
package framework

import (
	"context"
	"strings"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSuiteA_Secrets_A4_SecretScrubbingBeforeStorage verifies that secrets
// are detected and redacted AUTOMATICALLY before being stored in the ReasoningBank.
//
// Test A.4: Secret Scrubbing Before Storage
// - Record a memory containing secrets (API keys, passwords) with RAW content
// - Verify the Developer framework automatically scrubs before storage
// - This tests the integration, not manual scrubbing
//
// NOTE: The Developer.RecordMemory method automatically scrubs content,
// simulating the MCP layer behavior in production.
//
// NOTE: Uses SharedStore (mock) because chromem doesn't support $gte filters
// used by ReasoningBank's MinConfidence threshold.
// Each subtest uses a unique ProjectID to avoid cross-contamination.
func TestSuiteA_Secrets_A4_SecretScrubbingBeforeStorage(t *testing.T) {
	// Create shared store for all subtests - mock store supports $gte filters
	shared, err := NewSharedStore(SharedStoreConfig{ProjectID: "test_project_secrets_a4"})
	require.NoError(t, err)
	defer shared.Close()

	t.Run("API key is scrubbed before storage", func(t *testing.T) {
		// Create a developer with shared store (scrubber is initialized inside StartContextd)
		// Use unique project ID for isolation
		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-secrets-a4-1",
			TenantID:  "test-tenant-secrets",
			ProjectID: "test_project_secrets_a4_1", // Unique per subtest
		}, shared)
		require.NoError(t, err)

		ctx := context.Background()
		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Content with AWS API key - pass RAW content, let Developer scrub automatically
		contentWithSecret := "To fix the AWS connection error, use this key: AKIAIOSFODNN7EXAMPLE"

		// Record with RAW content - Developer.RecordMemory will scrub automatically
		memoryID, err := dev.RecordMemory(ctx, MemoryRecord{
			Title:   "AWS Connection Fix",
			Content: contentWithSecret, // RAW content with secret
			Outcome: "success",
			Tags:    []string{"aws", "troubleshooting"},
		})
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)

		// Search and retrieve the memory
		results, err := dev.SearchMemory(ctx, "AWS Connection Fix", 5)
		require.NoError(t, err)
		require.NotEmpty(t, results, "Should find the recorded memory")

		// Verify the secret was AUTOMATICALLY scrubbed
		assert.NotContains(t, results[0].Content, "AKIAIOSFODNN7EXAMPLE",
			"API key should be automatically scrubbed by Developer.RecordMemory")
		assert.Contains(t, results[0].Content, "[REDACTED]",
			"Redaction marker should be present after automatic scrubbing")
	})

	t.Run("GitHub token is scrubbed before storage", func(t *testing.T) {
		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-secrets-a4-2",
			TenantID:  "test-tenant-secrets",
			ProjectID: "test_project_secrets_a4_2", // Unique per subtest
		}, shared)
		require.NoError(t, err)

		ctx := context.Background()
		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Content with GitHub personal access token - RAW content
		contentWithSecret := "Use this token for CI: ghp_1234567890abcdefghijklmnopqrstuv123456"

		// Record with RAW content - automatic scrubbing
		memoryID, err := dev.RecordMemory(ctx, MemoryRecord{
			Title:   "CI Setup Instructions",
			Content: contentWithSecret, // RAW content
			Outcome: "success",
			Tags:    []string{"ci", "github"},
		})
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)

		// Search and retrieve
		results, err := dev.SearchMemory(ctx, "CI Setup Instructions", 5)
		require.NoError(t, err)
		require.NotEmpty(t, results, "Should find the recorded memory")

		assert.NotContains(t, results[0].Content, "ghp_1234567890abcdefghijklmnopqrstuv123456",
			"GitHub token should not be present in stored memory")
		assert.Contains(t, results[0].Content, "[REDACTED]",
			"Redaction marker should be present")
	})

	t.Run("multiple secrets are all scrubbed", func(t *testing.T) {
		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-secrets-a4-3",
			TenantID:  "test-tenant-secrets",
			ProjectID: "test_project_secrets_a4_3", // Unique per subtest
		}, shared)
		require.NoError(t, err)

		ctx := context.Background()
		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Content with multiple types of secrets - RAW content
		contentWithSecrets := `Configuration for production:
		- API_KEY=sk-1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMN
		- PASSWORD=SuperSecret123!
		- GitHub token: ghp_abcdefghijklmnopqrstuvwxyz1234567890`

		// Record with RAW content - automatic scrubbing handles all secrets
		memoryID, err := dev.RecordMemory(ctx, MemoryRecord{
			Title:   "Production Configuration",
			Content: contentWithSecrets, // RAW content with multiple secrets
			Outcome: "success",
			Tags:    []string{"production", "config"},
		})
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)

		// Search and retrieve
		results, err := dev.SearchMemory(ctx, "Production Configuration", 5)
		require.NoError(t, err)
		require.NotEmpty(t, results, "Should find the recorded memory")

		// Verify all secrets were automatically scrubbed
		assert.NotContains(t, results[0].Content, "sk-1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMN",
			"API key should be automatically scrubbed")
		assert.NotContains(t, results[0].Content, "ghp_abcdefghijklmnopqrstuvwxyz1234567890",
			"GitHub token should be automatically scrubbed")
		// Verify redaction markers are present
		assert.Contains(t, results[0].Content, "[REDACTED]",
			"Redaction markers should be present after automatic scrubbing")
	})

	t.Run("non-secrets are preserved", func(t *testing.T) {
		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-secrets-a4-4",
			TenantID:  "test-tenant-secrets",
			ProjectID: "test_project_secrets_a4_4", // Unique per subtest
		}, shared)
		require.NoError(t, err)

		ctx := context.Background()
		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Content without secrets - should pass through unchanged
		contentNoSecret := "The error occurred because the API endpoint was wrong. Fix by updating the URL to https://api.example.com/v2"

		// Record with RAW content - automatic scrubbing should not alter non-secrets
		memoryID, err := dev.RecordMemory(ctx, MemoryRecord{
			Title:   "API Endpoint Fix",
			Content: contentNoSecret, // RAW content with no secrets
			Outcome: "success",
			Tags:    []string{"api", "bug-fix"},
		})
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)

		// Search and retrieve
		results, err := dev.SearchMemory(ctx, "API Endpoint Fix", 5)
		require.NoError(t, err)
		require.NotEmpty(t, results, "Should find the recorded memory")

		// Content should be unchanged (no [REDACTED])
		assert.NotContains(t, results[0].Content, "[REDACTED]",
			"No redaction should occur when there are no secrets")
		assert.Contains(t, results[0].Content, "api.example.com",
			"Normal URLs should be preserved")
	})
}

// TestSuiteA_Secrets_A5_SecretScrubbingInSearchResults verifies that search
// results are scrubbed even if secrets somehow made it into storage.
// This is a defense-in-depth test.
//
// Test A.5: Secret Scrubbing in Search Results
// - Pass RAW content with secrets to RecordMemory
// - Developer.RecordMemory scrubs on write
// - Developer.SearchMemory scrubs on read (defense-in-depth)
// - Both layers protect against secret leakage
//
// NOTE: The Developer framework implements two-layer scrubbing:
// 1. RecordMemory scrubs content before storage
// 2. SearchMemory scrubs results before returning (defense-in-depth)
//
// NOTE: Uses SharedStore (mock) because chromem doesn't support $gte filters
// used by ReasoningBank's MinConfidence threshold.
func TestSuiteA_Secrets_A5_SecretScrubbingInSearchResults(t *testing.T) {
	// Create shared store for all subtests - mock store supports $gte filters
	shared, err := NewSharedStore(SharedStoreConfig{ProjectID: "test_project_secrets_a5"})
	require.NoError(t, err)
	defer shared.Close()

	t.Run("search results are scrubbed on retrieval", func(t *testing.T) {
		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-secrets-a5-1",
			TenantID:  "test-tenant-secrets",
			ProjectID: "test_project_secrets_a5_1", // Unique per subtest
		}, shared)
		require.NoError(t, err)

		ctx := context.Background()
		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Content with database connection string - pass RAW
		contentWithSecret := "Database connection: postgres://admin:P@ssw0rd123@db.example.com:5432/mydb"

		// Record with RAW content - automatic scrubbing on both record and search
		memoryID, err := dev.RecordMemory(ctx, MemoryRecord{
			Title:   "Database Connection String",
			Content: contentWithSecret, // RAW content with secret
			Outcome: "success",
			Tags:    []string{"database", "config"},
		})
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)

		// Search and retrieve - SearchMemory also scrubs (defense-in-depth)
		results, err := dev.SearchMemory(ctx, "Database Connection String", 5)
		require.NoError(t, err)
		require.NotEmpty(t, results, "Should find the recorded memory")

		// Verify results are scrubbed (by both layers)
		assert.NotContains(t, results[0].Content, "P@ssw0rd123",
			"Password should not appear in search results")
		assert.Contains(t, results[0].Content, "[REDACTED]",
			"Redaction marker should be present in search results")
	})

	t.Run("defense in depth - both layers scrub secrets", func(t *testing.T) {
		// This test verifies that even if one layer failed, the other would catch secrets
		// In production: MCP layer scrubs on record AND on response
		// In tests: Developer scrubs on RecordMemory AND SearchMemory
		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-secrets-a5-2",
			TenantID:  "test-tenant-secrets",
			ProjectID: "test_project_secrets_a5_2", // Unique per subtest
		}, shared)
		require.NoError(t, err)

		ctx := context.Background()
		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Content with JWT token - pass RAW
		contentWithJWT := "Authentication token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

		// Record RAW content - RecordMemory scrubs automatically
		memoryID, err := dev.RecordMemory(ctx, MemoryRecord{
			Title:   "JWT Authentication",
			Content: contentWithJWT, // RAW content with JWT
			Outcome: "success",
			Tags:    []string{"auth", "jwt"},
		})
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)

		// Search - SearchMemory ALSO scrubs (defense-in-depth)
		results, err := dev.SearchMemory(ctx, "JWT Authentication", 5)
		require.NoError(t, err)
		require.NotEmpty(t, results, "Should find the recorded memory")

		// The JWT should be redacted by BOTH layers
		assert.NotContains(t, results[0].Content, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			"JWT header should be redacted (defense-in-depth)")
		assert.NotContains(t, results[0].Content, "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			"JWT signature should be redacted (defense-in-depth)")
	})

	t.Run("multiple searches return consistently scrubbed results", func(t *testing.T) {
		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-secrets-a5-3",
			TenantID:  "test-tenant-secrets",
			ProjectID: "test_project_secrets_a5_3", // Unique per subtest
		}, shared)
		require.NoError(t, err)

		ctx := context.Background()
		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Content with Anthropic API key - pass RAW
		contentWithSecret := "To use Claude API, set your key: sk-ant-api03-abc123def456ghi789jkl012mno345pqr678stu901vwx234yz5678901234567890123456789012345678901234567890"

		// Record RAW content - automatic scrubbing
		memoryID, err := dev.RecordMemory(ctx, MemoryRecord{
			Title:   "Claude API Setup",
			Content: contentWithSecret, // RAW content
			Outcome: "success",
			Tags:    []string{"claude", "api"},
		})
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)

		// Perform multiple searches - each should consistently scrub
		for i := 0; i < 3; i++ {
			results, err := dev.SearchMemory(ctx, "Claude API Setup", 5)
			require.NoError(t, err)
			require.NotEmpty(t, results, "Should find the recorded memory on search %d", i+1)

			// Each search should consistently scrub (defense-in-depth layer)
			assert.NotContains(t, results[0].Content, "sk-ant-api03-",
				"Anthropic API key prefix should be redacted on search %d", i+1)
			assert.Contains(t, results[0].Content, "[REDACTED]",
				"Redaction marker should be present on search %d", i+1)
		}
	})
}

// TestSuiteA_Secrets_A6_SecretScrubbingBypassDetection is a known failure test
// that would detect if scrubbing was bypassed or disabled.
//
// Test A.6: Secret Scrubbing Bypass Detection (Known Failure)
// - This test is EXPECTED TO FAIL if scrubbing is bypassed
// - Documents what it would detect
//
// This test is currently skipped as it's designed to fail when the system
// is not properly configured. It demonstrates what a bypass would look like.
func TestSuiteA_Secrets_A6_SecretScrubbingBypassDetection(t *testing.T) {
	t.Skip("Known failure test - skip by default. Enable to test bypass detection.")

	t.Run("detects if scrubbing is completely disabled", func(t *testing.T) {
		// Create a disabled scrubber (simulating bypass scenario)
		cfg := secrets.DefaultConfig()
		cfg.Enabled = false
		scrubber, err := secrets.New(cfg)
		require.NoError(t, err)

		dev, err := NewDeveloper(DeveloperConfig{
			ID:        "dev-secrets-a6-1",
			TenantID:  "test-tenant-secrets",
			ProjectID: "test_project_secrets",
		})
		require.NoError(t, err)

		ctx := context.Background()
		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Content with AWS key
		secretContent := "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"

		// If scrubbing is disabled, this won't redact anything
		result := scrubber.Scrub(secretContent)

		// Record the content (potentially unscrubbed)
		_, err = dev.RecordMemory(ctx, MemoryRecord{
			Title:   "AWS Config",
			Content: result.Scrubbed,
			Outcome: "success",
			Tags:    []string{"aws"},
		})
		require.NoError(t, err)

		// Search and retrieve
		results, err := dev.SearchMemory(ctx, "AWS Config", 5)
		require.NoError(t, err)

		if len(results) > 0 {
			// THIS SHOULD FAIL if scrubbing is disabled
			// The secret would be present in the content
			if strings.Contains(results[0].Content, "AKIAIOSFODNN7EXAMPLE") {
				t.Errorf("SECURITY VIOLATION: Secret was not scrubbed! Found: AWS key in content")
			}
		}
	})

	t.Run("detects if scrubbing only happens at one layer", func(t *testing.T) {
		// This test would detect if scrubbing only happens at storage layer
		// but not at retrieval layer (or vice versa)
		scrubber, err := secrets.New(secrets.DefaultConfig())
		require.NoError(t, err)

		dev, err := NewDeveloper(DeveloperConfig{
			ID:        "dev-secrets-a6-2",
			TenantID:  "test-tenant-secrets",
			ProjectID: "test_project_secrets",
		})
		require.NoError(t, err)

		ctx := context.Background()
		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Content with private key
		secretContent := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA1234567890abcdefghijklmnopqrstuvwxyz
-----END RSA PRIVATE KEY-----`

		// Scrub before recording
		result := scrubber.Scrub(secretContent)
		require.True(t, result.HasFindings())

		// Record
		_, err = dev.RecordMemory(ctx, MemoryRecord{
			Title:   "SSH Key",
			Content: result.Scrubbed,
			Outcome: "success",
			Tags:    []string{"ssh"},
		})
		require.NoError(t, err)

		// Retrieve
		results, err := dev.SearchMemory(ctx, "SSH Key", 5)
		require.NoError(t, err)

		if len(results) > 0 {
			// THIS SHOULD FAIL if either layer is bypassed
			if strings.Contains(results[0].Content, "BEGIN RSA PRIVATE KEY") {
				t.Errorf("SECURITY VIOLATION: Private key header was not scrubbed!")
			}
			if strings.Contains(results[0].Content, "MIIEpAIBAAKCAQEA") {
				t.Errorf("SECURITY VIOLATION: Private key content was not scrubbed!")
			}
		}
	})

	t.Run("detects if allow-list is too permissive", func(t *testing.T) {
		// Create scrubber with overly permissive allow-list
		cfg := secrets.DefaultConfig()
		// Add a very broad allow-list pattern (simulating misconfiguration)
		cfg.AllowList = []string{".*"} // This would allow everything!
		scrubber, err := secrets.New(cfg)
		require.NoError(t, err)

		dev, err := NewDeveloper(DeveloperConfig{
			ID:        "dev-secrets-a6-3",
			TenantID:  "test-tenant-secrets",
			ProjectID: "test_project_secrets",
		})
		require.NoError(t, err)

		ctx := context.Background()
		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Content with GitHub token
		secretContent := "ghp_1234567890abcdefghijklmnopqrstuv123456"

		// Scrub with permissive allow-list
		result := scrubber.Scrub(secretContent)

		// Record
		_, err = dev.RecordMemory(ctx, MemoryRecord{
			Title:   "GitHub Token",
			Content: result.Scrubbed,
			Outcome: "success",
			Tags:    []string{"github"},
		})
		require.NoError(t, err)

		// Retrieve
		results, err := dev.SearchMemory(ctx, "GitHub Token", 5)
		require.NoError(t, err)

		if len(results) > 0 {
			// THIS SHOULD FAIL if allow-list is too permissive
			if strings.Contains(results[0].Content, "ghp_1234567890") {
				t.Errorf("SECURITY VIOLATION: GitHub token was allowed through due to permissive allow-list!")
			}
		}
	})
}

// TestSecretsScrubbingIntegration verifies the secrets package integration
// works correctly with the test framework.
func TestSecretsScrubbingIntegration(t *testing.T) {
	t.Run("scrubber detects all default rule types", func(t *testing.T) {
		scrubber, err := secrets.New(secrets.DefaultConfig())
		require.NoError(t, err)

		testCases := []struct {
			name     string
			content  string
			shouldFind bool
			ruleID   string
		}{
			{
				name:     "AWS Access Key",
				content:  "AWS access key: AKIAIOSFODNN7EXAMPLE", // Include keyword
				shouldFind: true,
				ruleID:   "aws-access-key-id",
			},
			{
				name:     "GitHub PAT",
				content:  "ghp_1234567890abcdefghijklmnopqrstuv123456",
				shouldFind: true,
				ruleID:   "github-token",
			},
			{
				name:     "JWT",
				content:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0In0.abc123",
				shouldFind: true,
				ruleID:   "jwt",
			},
			{
				name:     "No secrets",
				content:  "Just normal text about AWS services",
				shouldFind: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := scrubber.Scrub(tc.content)
				if tc.shouldFind {
					assert.True(t, result.HasFindings(), "Should detect secret in: %s", tc.content)
					if tc.ruleID != "" {
						assert.Contains(t, result.RuleIDs(), tc.ruleID,
							"Should detect rule %s", tc.ruleID)
					}
					assert.NotEqual(t, result.Original, result.Scrubbed,
						"Content should be modified when secrets found")
				} else {
					assert.False(t, result.HasFindings(), "Should not detect secrets in: %s", tc.content)
					assert.Equal(t, result.Original, result.Scrubbed,
						"Content should be unchanged when no secrets")
				}
			})
		}
	})

	t.Run("scrubber performance is acceptable", func(t *testing.T) {
		scrubber, err := secrets.New(secrets.DefaultConfig())
		require.NoError(t, err)

		// Test with reasonably large content
		largeContent := strings.Repeat("This is a test of scrubbing performance. ", 100)
		largeContent += "API_KEY=sk-1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMN"

		result := scrubber.Scrub(largeContent)

		// Scrubbing should be fast (under 100ms for this size)
		assert.Less(t, result.Duration.Milliseconds(), int64(100),
			"Scrubbing should complete in under 100ms")
		assert.True(t, result.HasFindings(), "Should find the API key in large content")
	})
}
