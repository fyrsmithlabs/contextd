package workflows

import (
	"errors"
	"strings"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionValidationConfig_Validate tests validation of VersionValidationConfig
func TestVersionValidationConfig_Validate(t *testing.T) {
	validToken := config.Secret("ghp_test1234567890")
	emptyToken := config.Secret("")

	tests := []struct {
		name    string
		config  VersionValidationConfig
		wantErr error
	}{
		{
			name: "valid config",
			config: VersionValidationConfig{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				PRNumber:    123,
				HeadSHA:     "abc123def456",
				GitHubToken: validToken,
			},
			wantErr: nil,
		},
		{
			name: "valid with full SHA",
			config: VersionValidationConfig{
				Owner:       "test",
				Repo:        "repo",
				PRNumber:    1,
				HeadSHA:     "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				GitHubToken: validToken,
			},
			wantErr: nil,
		},
		{
			name: "empty Owner",
			config: VersionValidationConfig{
				Owner:       "",
				Repo:        "contextd",
				PRNumber:    123,
				HeadSHA:     "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrEmptyField,
		},
		{
			name: "invalid Owner - special chars",
			config: VersionValidationConfig{
				Owner:       "owner@#$",
				Repo:        "contextd",
				PRNumber:    123,
				HeadSHA:     "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidGitHubIdentifier,
		},
		{
			name: "invalid Owner - too long",
			config: VersionValidationConfig{
				Owner:       "a" + strings.Repeat("b", 40),
				Repo:        "contextd",
				PRNumber:    123,
				HeadSHA:     "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidGitHubIdentifier,
		},
		{
			name: "empty Repo",
			config: VersionValidationConfig{
				Owner:       "fyrsmithlabs",
				Repo:        "",
				PRNumber:    123,
				HeadSHA:     "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrEmptyField,
		},
		{
			name: "invalid Repo - special chars",
			config: VersionValidationConfig{
				Owner:       "fyrsmithlabs",
				Repo:        "repo@#$%",
				PRNumber:    123,
				HeadSHA:     "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidGitHubIdentifier,
		},
		{
			name: "invalid PRNumber - zero",
			config: VersionValidationConfig{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				PRNumber:    0,
				HeadSHA:     "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidPRNumber,
		},
		{
			name: "invalid PRNumber - negative",
			config: VersionValidationConfig{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				PRNumber:    -1,
				HeadSHA:     "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidPRNumber,
		},
		{
			name: "empty HeadSHA",
			config: VersionValidationConfig{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				PRNumber:    123,
				HeadSHA:     "",
				GitHubToken: validToken,
			},
			wantErr: ErrEmptyField,
		},
		{
			name: "invalid HeadSHA - too short",
			config: VersionValidationConfig{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				PRNumber:    123,
				HeadSHA:     "abc12",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "invalid HeadSHA - non-hex",
			config: VersionValidationConfig{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				PRNumber:    123,
				HeadSHA:     "ghijklm",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "missing GitHubToken",
			config: VersionValidationConfig{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				PRNumber:    123,
				HeadSHA:     "abc1234",
				GitHubToken: emptyToken,
			},
			wantErr: ErrMissingToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateComprehensive()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "expected error %v, got %v", tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestPluginUpdateValidationConfig_Validate tests validation of PluginUpdateValidationConfig
func TestPluginUpdateValidationConfig_Validate(t *testing.T) {
	validToken := config.Secret("ghp_test1234567890")

	tests := []struct {
		name    string
		config  PluginUpdateValidationConfig
		wantErr error
	}{
		{
			name: "valid config",
			config: PluginUpdateValidationConfig{
				Owner:      "fyrsmithlabs",
				Repo:       "contextd",
				PRNumber:   123,
				BaseBranch: "main",
				HeadBranch: "feature/validation",
				HeadSHA:    "abc123def456",
				GitHubToken: validToken,
			},
			wantErr: nil,
		},
		{
			name: "valid with namespaced branch",
			config: PluginUpdateValidationConfig{
				Owner:      "test",
				Repo:       "repo",
				PRNumber:   1,
				BaseBranch: "release/v1.0",
				HeadBranch: "hotfix/security-patch",
				HeadSHA:    "abc1234",
				GitHubToken: validToken,
			},
			wantErr: nil,
		},
		{
			name: "empty BaseBranch",
			config: PluginUpdateValidationConfig{
				Owner:      "fyrsmithlabs",
				Repo:       "contextd",
				PRNumber:   123,
				BaseBranch: "",
				HeadBranch: "feature/test",
				HeadSHA:    "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrEmptyField,
		},
		{
			name: "invalid BaseBranch - path traversal",
			config: PluginUpdateValidationConfig{
				Owner:      "fyrsmithlabs",
				Repo:       "contextd",
				PRNumber:   123,
				BaseBranch: "../main",
				HeadBranch: "feature/test",
				HeadSHA:    "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrPathTraversal,
		},
		{
			name: "invalid BaseBranch - spaces",
			config: PluginUpdateValidationConfig{
				Owner:      "fyrsmithlabs",
				Repo:       "contextd",
				PRNumber:   123,
				BaseBranch: "main branch",
				HeadBranch: "feature/test",
				HeadSHA:    "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "invalid HeadBranch - starts with slash",
			config: PluginUpdateValidationConfig{
				Owner:      "fyrsmithlabs",
				Repo:       "contextd",
				PRNumber:   123,
				BaseBranch: "main",
				HeadBranch: "/feature/test",
				HeadSHA:    "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "invalid HeadBranch - forbidden char",
			config: PluginUpdateValidationConfig{
				Owner:      "fyrsmithlabs",
				Repo:       "contextd",
				PRNumber:   123,
				BaseBranch: "main",
				HeadBranch: "feature*test",
				HeadSHA:    "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateComprehensive()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "expected error %v, got %v", tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestFetchFileContentInput_Validate tests validation of FetchFileContentInput
func TestFetchFileContentInput_Validate(t *testing.T) {
	validToken := config.Secret("ghp_test1234567890")

	tests := []struct {
		name    string
		input   FetchFileContentInput
		wantErr error
	}{
		{
			name: "valid input",
			input: FetchFileContentInput{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				Path:        "VERSION",
				Ref:         "abc123",
				GitHubToken: validToken,
			},
			wantErr: nil,
		},
		{
			name: "valid input with nested path",
			input: FetchFileContentInput{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				Path:        ".claude-plugin/plugin.json",
				Ref:         "main",
				GitHubToken: validToken,
			},
			wantErr: nil,
		},
		{
			name: "invalid Path - path traversal",
			input: FetchFileContentInput{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				Path:        "../../etc/passwd",
				Ref:         "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrPathTraversal,
		},
		{
			name: "invalid Path - absolute path",
			input: FetchFileContentInput{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				Path:        "/etc/passwd",
				Ref:         "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "invalid Path - starts with ./",
			input: FetchFileContentInput{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				Path:        "./VERSION",
				Ref:         "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "empty Path",
			input: FetchFileContentInput{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				Path:        "",
				Ref:         "abc123",
				GitHubToken: validToken,
			},
			wantErr: ErrEmptyField,
		},
		{
			name: "invalid Ref - not SHA or branch",
			input: FetchFileContentInput{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				Path:        "VERSION",
				Ref:         "invalid*ref",
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.ValidateComprehensive()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "expected error %v, got %v", tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestPostVersionCommentInput_Validate tests validation of PostVersionCommentInput
func TestPostVersionCommentInput_Validate(t *testing.T) {
	validToken := config.Secret("ghp_test1234567890")

	tests := []struct {
		name    string
		input   PostVersionCommentInput
		wantErr error
	}{
		{
			name: "valid input",
			input: PostVersionCommentInput{
				Owner:         "fyrsmithlabs",
				Repo:          "contextd",
				PRNumber:      123,
				VersionFile:   "1.2.3",
				PluginVersion: "0.0.0",
				GitHubToken:   validToken,
			},
			wantErr: nil,
		},
		{
			name: "valid input with empty versions (for removal)",
			input: PostVersionCommentInput{
				Owner:         "fyrsmithlabs",
				Repo:          "contextd",
				PRNumber:      123,
				VersionFile:   "",
				PluginVersion: "",
				GitHubToken:   validToken,
			},
			wantErr: nil,
		},
		{
			name: "invalid Owner",
			input: PostVersionCommentInput{
				Owner:         "",
				Repo:          "contextd",
				PRNumber:      123,
				VersionFile:   "1.2.3",
				PluginVersion: "0.0.0",
				GitHubToken:   validToken,
			},
			wantErr: ErrEmptyField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.ValidateComprehensive()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "expected error %v, got %v", tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestFetchPRFilesInput_Validate tests validation of FetchPRFilesInput
func TestFetchPRFilesInput_Validate(t *testing.T) {
	validToken := config.Secret("ghp_test1234567890")

	tests := []struct {
		name    string
		input   FetchPRFilesInput
		wantErr error
	}{
		{
			name: "valid input",
			input: FetchPRFilesInput{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				PRNumber:    123,
				GitHubToken: validToken,
			},
			wantErr: nil,
		},
		{
			name: "invalid PRNumber",
			input: FetchPRFilesInput{
				Owner:       "fyrsmithlabs",
				Repo:        "contextd",
				PRNumber:    0,
				GitHubToken: validToken,
			},
			wantErr: ErrInvalidPRNumber,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.ValidateComprehensive()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "expected error %v, got %v", tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test helper functions

func TestIsValidGitSHA(t *testing.T) {
	tests := []struct {
		name  string
		sha   string
		valid bool
	}{
		{"valid short SHA", "abc123d", true},
		{"valid full SHA", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", true},
		{"invalid - too short", "abc12", false},
		{"invalid - too long", "a" + strings.Repeat("b", 41), false},
		{"invalid - non-hex", "ghijklm", false},
		{"valid - mixed case", "AbC1234", true},
		{"invalid - spaces", "abc 123", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, isValidGitSHA(tt.sha))
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{"valid simple path", "VERSION", nil},
		{"valid nested path", ".claude-plugin/plugin.json", nil},
		{"valid deep path", "internal/workflows/types.go", nil},
		{"invalid - path traversal", "../../etc/passwd", ErrPathTraversal},
		{"invalid - absolute path", "/etc/passwd", ErrInvalidInput},
		{"invalid - starts with ./", "./VERSION", ErrInvalidInput},
		{"invalid - empty", "", ErrInvalidInput},
		{"invalid - hidden traversal", "foo/../bar", ErrPathTraversal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.path)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name    string
		branch  string
		wantErr error
	}{
		{"valid simple", "main", nil},
		{"valid with slash", "feature/test", nil},
		{"valid with hyphen", "release-1.0", nil},
		{"valid with underscore", "feature_test", nil},
		{"invalid - path traversal", "../main", ErrPathTraversal},
		{"invalid - spaces", "main branch", ErrInvalidInput},
		{"invalid - starts with slash", "/main", ErrInvalidInput},
		{"invalid - ends with slash", "main/", ErrInvalidInput},
		{"invalid - double slash", "feature//test", ErrInvalidInput},
		{"invalid - asterisk", "feature*", ErrInvalidInput},
		{"invalid - tilde", "feature~1", ErrInvalidInput},
		{"invalid - caret", "feature^", ErrInvalidInput},
		{"invalid - colon", "feature:test", ErrInvalidInput},
		{"invalid - question", "feature?", ErrInvalidInput},
		{"invalid - bracket", "feature[test]", ErrInvalidInput},
		{"invalid - backslash", "feature\\test", ErrInvalidInput},
		{"invalid - @{", "feature@{test}", ErrInvalidInput},
		{"invalid - empty", "", ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchName(tt.branch)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				require.NoError(t, err)
			}
		})
	}
}
