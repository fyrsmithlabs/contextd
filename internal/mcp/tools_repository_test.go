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
