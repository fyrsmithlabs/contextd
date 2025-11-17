package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateName(t *testing.T) {
	tests := []struct {
		name       string
		ownerID    string
		projectID  string
		branch     string
		want       string
		wantErr    bool
		errMessage string
	}{
		{
			name:      "main branch collection",
			ownerID:   "owner_abc123",
			projectID: "project_def456",
			branch:    "main",
			want:      "owner_abc123/project_def456/main",
			wantErr:   false,
		},
		{
			name:      "feature branch collection",
			ownerID:   "owner_abc123",
			projectID: "project_def456",
			branch:    "feature/v3-rebuild",
			want:      "owner_abc123/project_def456/feature_v3-rebuild",
			wantErr:   false,
		},
		{
			name:      "master branch collection",
			ownerID:   "owner_xyz789",
			projectID: "project_uvw012",
			branch:    "master",
			want:      "owner_xyz789/project_uvw012/master",
			wantErr:   false,
		},
		{
			name:       "empty owner ID",
			ownerID:    "",
			projectID:  "project_def456",
			branch:     "main",
			want:       "",
			wantErr:    true,
			errMessage: "owner ID required",
		},
		{
			name:       "empty project ID",
			ownerID:    "owner_abc123",
			projectID:  "",
			branch:     "main",
			want:       "",
			wantErr:    true,
			errMessage: "project ID required",
		},
		{
			name:       "empty branch",
			ownerID:    "owner_abc123",
			projectID:  "project_def456",
			branch:     "",
			want:       "",
			wantErr:    true,
			errMessage: "branch required",
		},
		{
			name:      "branch with slashes sanitized",
			ownerID:   "owner_abc",
			projectID: "project_def",
			branch:    "bugfix/security/xss",
			want:      "owner_abc/project_def/bugfix_security_xss",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateName(tt.ownerID, tt.projectID, tt.branch)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSanitizeBranch(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		want   string
	}{
		{"simple branch", "main", "main"},
		{"feature branch with slash", "feature/auth", "feature_auth"},
		{"multiple slashes", "bugfix/security/xss", "bugfix_security_xss"},
		{"already sanitized", "feature_auth", "feature_auth"},
		{"mixed separators", "feature/v3-rebuild", "feature_v3-rebuild"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeBranch(tt.branch)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseCollectionName(t *testing.T) {
	tests := []struct {
		name           string
		collectionName string
		wantOwnerID    string
		wantProjectID  string
		wantBranch     string
		wantErr        bool
		errMessage     string
	}{
		{
			name:           "valid main branch",
			collectionName: "owner_abc123/project_def456/main",
			wantOwnerID:    "owner_abc123",
			wantProjectID:  "project_def456",
			wantBranch:     "main",
			wantErr:        false,
		},
		{
			name:           "valid feature branch",
			collectionName: "owner_abc/project_def/feature_v3-rebuild",
			wantOwnerID:    "owner_abc",
			wantProjectID:  "project_def",
			wantBranch:     "feature_v3-rebuild",
			wantErr:        false,
		},
		{
			name:           "invalid format - too few parts",
			collectionName: "owner_abc/project_def",
			wantErr:        true,
			errMessage:     "invalid collection name format",
		},
		{
			name:           "invalid format - too many parts",
			collectionName: "owner_abc/project_def/main/extra",
			wantErr:        true,
			errMessage:     "invalid collection name format",
		},
		{
			name:           "empty collection name",
			collectionName: "",
			wantErr:        true,
			errMessage:     "collection name required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ownerID, projectID, branch, err := ParseCollectionName(tt.collectionName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantOwnerID, ownerID)
				assert.Equal(t, tt.wantProjectID, projectID)
				assert.Equal(t, tt.wantBranch, branch)
			}
		})
	}
}

func TestGenerateNameIntegration(t *testing.T) {
	// Integration test: Use pkg/auth to generate owner ID
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This will test integration with pkg/auth after we implement auth.DeriveOwnerID
	// For now, we'll skip this test
	t.Skip("waiting for pkg/auth implementation")
}
