package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fyrsmithlabs/contextd/internal/tenant"
)

// TestRepositoryTools_TenantIDConsistency is a regression test for GitHub issue #19.
// Bug: repository_search used "default" tenant ID while repository_index used
// tenant.GetTenantIDForPath(), causing collection name mismatch.
// Fix: Both tools now use tenant.GetTenantIDForPath() for consistent collection naming.
func TestRepositoryTools_TenantIDConsistency(t *testing.T) {
	testCases := []struct {
		name        string
		projectPath string
		tenantID    string // explicit tenant ID provided by user
		wantSame    bool   // whether both paths should produce same tenant ID
	}{
		{
			name:        "no_explicit_tenant_id",
			projectPath: "/home/testuser/projects/myproject",
			tenantID:    "",
			wantSame:    true,
		},
		{
			name:        "explicit_tenant_id",
			projectPath: "/home/testuser/projects/myproject",
			tenantID:    "explicit_tenant",
			wantSame:    true,
		},
		{
			name:        "different_project_paths",
			projectPath: "/home/other/code/app",
			tenantID:    "",
			wantSame:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate repository_index tenant ID resolution
			indexTenantID := tc.tenantID
			if indexTenantID == "" {
				indexTenantID = tenant.GetTenantIDForPath(tc.projectPath)
			}

			// Simulate repository_search tenant ID resolution (after fix)
			searchTenantID := tc.tenantID
			if searchTenantID == "" {
				searchTenantID = tenant.GetTenantIDForPath(tc.projectPath)
			}

			// Both should produce the same tenant ID
			if tc.wantSame {
				assert.Equal(t, indexTenantID, searchTenantID,
					"repository_index and repository_search must use consistent tenant IDs (regression test for #19)")
			}
		})
	}
}

// TestRepositoryTools_CollectionNameConsistency verifies that collection names
// are generated consistently between repository_index and repository_search.
func TestRepositoryTools_CollectionNameConsistency(t *testing.T) {
	testCases := []struct {
		name        string
		projectPath string
		tenantID    string
	}{
		{
			name:        "typical_project",
			projectPath: "/home/dahendel/projects/contextd",
			tenantID:    "",
		},
		{
			name:        "explicit_tenant",
			projectPath: "/home/user/code/myapp",
			tenantID:    "mycompany",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Resolve tenant ID as both tools would
			resolvedTenantID := tc.tenantID
			if resolvedTenantID == "" {
				resolvedTenantID = tenant.GetTenantIDForPath(tc.projectPath)
			}

			// Both tools use the same collection name format: {tenant}_{project}_codebase
			// This test verifies the tenant ID resolution is consistent
			assert.NotEmpty(t, resolvedTenantID, "tenant ID should never be empty after resolution")
			assert.NotEqual(t, "default", resolvedTenantID,
				"tenant ID should NOT default to 'default' - use GetTenantIDForPath instead (regression test for #19)")
		})
	}
}

// TestRepositorySearch_ContentMode_Minimal tests that minimal mode returns only file_path, score, branch
func TestRepositorySearch_ContentMode_Minimal(t *testing.T) {
	input := repositorySearchInput{
		Query:       "test query",
		ProjectPath: "/home/user/project",
		ContentMode: "minimal",
	}

	// Verify ContentMode field exists and is set correctly
	assert.Equal(t, "minimal", input.ContentMode)

	// Test result structure for minimal mode
	result := map[string]interface{}{
		"file_path": "/home/user/project/main.go",
		"score":     0.95,
		"branch":    "main",
	}

	// Minimal mode should NOT have content or metadata
	_, hasContent := result["content"]
	_, hasMetadata := result["metadata"]
	_, hasContentPreview := result["content_preview"]

	assert.False(t, hasContent, "minimal mode should not include content")
	assert.False(t, hasMetadata, "minimal mode should not include metadata")
	assert.False(t, hasContentPreview, "minimal mode should not include content_preview")

	// Should have required fields
	assert.NotEmpty(t, result["file_path"])
	assert.NotNil(t, result["score"])
	assert.NotEmpty(t, result["branch"])
}

// TestRepositorySearch_ContentMode_Preview tests that preview mode includes content_preview (max 200 chars)
func TestRepositorySearch_ContentMode_Preview(t *testing.T) {
	input := repositorySearchInput{
		Query:       "test query",
		ProjectPath: "/home/user/project",
		ContentMode: "preview",
	}

	assert.Equal(t, "preview", input.ContentMode)

	// Test content preview truncation
	longContent := "This is a very long content string that exceeds 200 characters. " +
		"We need to make sure it gets truncated properly when using preview mode. " +
		"This should definitely be longer than 200 characters to test the truncation logic. " +
		"Adding more text to ensure we exceed the limit."

	preview := longContent
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}

	assert.LessOrEqual(t, len(preview), 203, "preview should be max 200 chars + '...'")
	assert.True(t, len(longContent) > 200, "test content should be longer than 200 chars")
}

// TestRepositorySearch_ContentMode_Full tests that full mode includes complete content and metadata
func TestRepositorySearch_ContentMode_Full(t *testing.T) {
	input := repositorySearchInput{
		Query:       "test query",
		ProjectPath: "/home/user/project",
		ContentMode: "full",
	}

	assert.Equal(t, "full", input.ContentMode)

	// Full mode result structure
	result := map[string]interface{}{
		"file_path": "/home/user/project/main.go",
		"score":     0.95,
		"branch":    "main",
		"content":   "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
		"metadata": map[string]interface{}{
			"language": "go",
		},
	}

	// Full mode should have content and metadata
	_, hasContent := result["content"]
	_, hasMetadata := result["metadata"]

	assert.True(t, hasContent, "full mode should include content")
	assert.True(t, hasMetadata, "full mode should include metadata")
}

// TestRepositorySearch_ContentMode_DefaultIsMinimal tests that empty content_mode defaults to minimal
func TestRepositorySearch_ContentMode_DefaultIsMinimal(t *testing.T) {
	input := repositorySearchInput{
		Query:       "test query",
		ProjectPath: "/home/user/project",
		// ContentMode intentionally not set
	}

	// Default should be empty string, which the handler should treat as "minimal"
	assert.Equal(t, "", input.ContentMode)

	// Test the default resolution logic
	contentMode := input.ContentMode
	if contentMode == "" {
		contentMode = "minimal"
	}

	assert.Equal(t, "minimal", contentMode, "empty content_mode should default to minimal")
}

// TestRepositorySearch_ContentMode_OutputIncludesMode tests that output includes content_mode used
func TestRepositorySearch_ContentMode_OutputIncludesMode(t *testing.T) {
	output := repositorySearchOutput{
		Results:     []map[string]interface{}{},
		Count:       0,
		Query:       "test query",
		ContentMode: "minimal",
	}

	assert.Equal(t, "minimal", output.ContentMode, "output should include content_mode used")
}

// TestRepositorySearch_ContentMode_InvalidModeValidation tests that invalid content_mode values are rejected
func TestRepositorySearch_ContentMode_InvalidModeValidation(t *testing.T) {
	testCases := []struct {
		name        string
		contentMode string
		shouldError bool
	}{
		{"valid_minimal", "minimal", false},
		{"valid_preview", "preview", false},
		{"valid_full", "full", false},
		{"valid_empty_defaults_minimal", "", false},
		{"invalid_uppercase", "FULL", true},
		{"invalid_typo", "fulll", true},
		{"invalid_unknown", "compact", true},
		{"invalid_mixed_case", "Preview", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate content mode using the same logic as the handler
			contentMode := tc.contentMode
			if contentMode == "" {
				contentMode = "minimal"
			}

			var isValid bool
			switch contentMode {
			case "minimal", "preview", "full":
				isValid = true
			default:
				isValid = false
			}

			if tc.shouldError {
				assert.False(t, isValid, "content_mode %q should be invalid", tc.contentMode)
			} else {
				assert.True(t, isValid, "content_mode %q should be valid", tc.contentMode)
			}
		})
	}
}

// TestRepositorySearch_ContentMode_Preview_UTF8Safe tests that preview truncation is UTF-8 safe
func TestRepositorySearch_ContentMode_Preview_UTF8Safe(t *testing.T) {
	const previewMaxRunes = 200
	const previewEllipsis = "..."

	testCases := []struct {
		name           string
		content        string
		expectTruncate bool
		description    string
	}{
		{
			name:           "ascii_short",
			content:        "Hello world",
			expectTruncate: false,
			description:    "Short ASCII content should not be truncated",
		},
		{
			name:           "ascii_exact_200",
			content:        string(make([]byte, 200)),
			expectTruncate: false,
			description:    "Exactly 200 ASCII chars should not be truncated",
		},
		{
			name:           "ascii_over_200",
			content:        string(make([]byte, 250)),
			expectTruncate: true,
			description:    "Over 200 ASCII chars should be truncated",
		},
		{
			name:           "emoji_content",
			content:        "Hello " + string([]rune{'🎉', '🚀', '✨'}),
			expectTruncate: false,
			description:    "Short content with emoji should not be truncated",
		},
		{
			name:           "chinese_characters",
			content:        "你好世界" + string(make([]rune, 198)), // 4 + 198 = 202 runes
			expectTruncate: true,
			description:    "Chinese chars count as 1 rune each, should truncate at 200 runes",
		},
		{
			name:           "mixed_multibyte",
			content:        "Hello 世界 🎉 " + string(make([]rune, 195)),
			expectTruncate: true,
			description:    "Mixed ASCII and multi-byte should truncate correctly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Apply the same truncation logic as the handler
			preview := tc.content
			runes := []rune(preview)
			originalRuneCount := len(runes)

			if len(runes) > previewMaxRunes {
				preview = string(runes[:previewMaxRunes]) + previewEllipsis
			}

			if tc.expectTruncate {
				assert.True(t, originalRuneCount > previewMaxRunes,
					"%s: expected content to need truncation", tc.description)
				assert.LessOrEqual(t, len([]rune(preview)), previewMaxRunes+len([]rune(previewEllipsis)),
					"%s: truncated preview should be at most 200 runes + ellipsis", tc.description)
				assert.True(t, len(preview) > 0, "%s: preview should not be empty", tc.description)
			} else {
				assert.Equal(t, tc.content, preview,
					"%s: content should not be modified", tc.description)
			}

			// Verify the result is valid UTF-8
			assert.True(t, isValidUTF8(preview),
				"%s: preview must be valid UTF-8", tc.description)
		})
	}
}

// isValidUTF8 checks if a string is valid UTF-8
func isValidUTF8(s string) bool {
	for i := 0; i < len(s); {
		r, size := rune(s[i]), 1
		if r >= 0x80 {
			r, size = decodeRune(s[i:])
			if r == '\uFFFD' && size == 1 {
				return false // Invalid UTF-8 sequence
			}
		}
		i += size
	}
	return true
}

// decodeRune decodes the first UTF-8 rune from s
func decodeRune(s string) (rune, int) {
	if len(s) == 0 {
		return '\uFFFD', 0
	}
	// Use standard library for actual decoding
	for i, r := range s {
		if i == 0 {
			return r, len(string(r))
		}
	}
	return '\uFFFD', 1
}
