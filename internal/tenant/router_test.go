package tenant

import (
	"testing"
)

func TestNewRouter(t *testing.T) {
	r := NewRouter(true)
	if r == nil {
		t.Fatal("NewRouter returned nil")
	}
}

func TestRouter_GetCollectionName(t *testing.T) {
	r := NewRouter(false)

	tests := []struct {
		name          string
		scope         Scope
		collType      CollectionType
		tenantID      string
		teamID        string
		projectID     string
		expected      string
		expectError   bool
		expectedError error
	}{
		{
			name:        "org scope - memories",
			scope:       ScopeOrg,
			collType:    CollectionMemories,
			tenantID:    "acme",
			expected:    "org_memories",
			expectError: false,
		},
		{
			name:        "team scope - remediations",
			scope:       ScopeTeam,
			collType:    CollectionRemediations,
			tenantID:    "acme",
			teamID:      "platform",
			expected:    "platform_remediations",
			expectError: false,
		},
		{
			name:        "project scope - checkpoints",
			scope:       ScopeProject,
			collType:    CollectionCheckpoints,
			tenantID:    "acme",
			teamID:      "platform",
			projectID:   "api",
			expected:    "platform_api_checkpoints",
			expectError: false,
		},
		{
			name:          "missing tenant ID",
			scope:         ScopeOrg,
			collType:      CollectionMemories,
			tenantID:      "",
			expectError:   true,
			expectedError: ErrInvalidTenantID,
		},
		{
			name:          "team scope missing team ID",
			scope:         ScopeTeam,
			collType:      CollectionMemories,
			tenantID:      "acme",
			teamID:        "",
			expectError:   true,
			expectedError: ErrInvalidTeamID,
		},
		{
			name:          "project scope missing team ID",
			scope:         ScopeProject,
			collType:      CollectionMemories,
			tenantID:      "acme",
			teamID:        "",
			projectID:     "api",
			expectError:   true,
			expectedError: ErrInvalidTeamID,
		},
		{
			name:          "project scope missing project ID",
			scope:         ScopeProject,
			collType:      CollectionMemories,
			tenantID:      "acme",
			teamID:        "platform",
			projectID:     "",
			expectError:   true,
			expectedError: ErrInvalidProjectID,
		},
		{
			name:          "invalid scope",
			scope:         Scope("invalid"),
			collType:      CollectionMemories,
			tenantID:      "acme",
			expectError:   true,
			expectedError: ErrInvalidScope,
		},
		{
			name:        "all collection types - org",
			scope:       ScopeOrg,
			collType:    CollectionCodebase,
			tenantID:    "test",
			expected:    "org_codebase",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.GetCollectionName(tt.scope, tt.collType, tt.tenantID, tt.teamID, tt.projectID)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.expectedError)
				} else if tt.expectedError != nil && err != tt.expectedError {
					t.Errorf("Expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestRouter_ValidateAccess(t *testing.T) {
	r := NewRouter(false)

	// Stub implementation always returns nil
	err := r.ValidateAccess("tenant1", "team1", "project1", "collection1")
	if err != nil {
		t.Errorf("ValidateAccess returned error: %v (stub should return nil)", err)
	}
}

func TestRouter_GetSearchCollections(t *testing.T) {
	r := NewRouter(false)

	tests := []struct {
		name          string
		scope         Scope
		collType      CollectionType
		tenantID      string
		teamID        string
		projectID     string
		expected      []string
		expectError   bool
	}{
		{
			name:      "project scope - searches project, team, org",
			scope:     ScopeProject,
			collType:  CollectionMemories,
			tenantID:  "acme",
			teamID:    "platform",
			projectID: "api",
			expected: []string{
				"platform_api_memories",
				"platform_memories",
				"org_memories",
			},
			expectError: false,
		},
		{
			name:     "team scope - searches team, org",
			scope:    ScopeTeam,
			collType: CollectionRemediations,
			tenantID: "acme",
			teamID:   "frontend",
			expected: []string{
				"frontend_remediations",
				"org_remediations",
			},
			expectError: false,
		},
		{
			name:     "org scope - searches org only",
			scope:    ScopeOrg,
			collType: CollectionCheckpoints,
			tenantID: "acme",
			expected: []string{
				"org_checkpoints",
			},
			expectError: false,
		},
		{
			name:        "invalid scope",
			scope:       Scope("invalid"),
			collType:    CollectionMemories,
			tenantID:    "acme",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.GetSearchCollections(tt.scope, tt.collType, tt.tenantID, tt.teamID, tt.projectID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(result) != len(tt.expected) {
					t.Errorf("Expected %d collections, got %d", len(tt.expected), len(result))
				}
				for i, expected := range tt.expected {
					if i >= len(result) {
						t.Errorf("Missing collection at index %d: %s", i, expected)
						continue
					}
					if result[i] != expected {
						t.Errorf("Collection %d: expected %q, got %q", i, expected, result[i])
					}
				}
			}
		})
	}
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid lowercase", "test", true},
		{"valid with underscore", "test_collection", true},
		{"valid with numbers", "test123", true},
		{"valid mixed", "test_123_collection", true},
		{"uppercase", "Test", true}, // converted to lowercase internally
		{"with hyphen", "test-collection", false},
		{"with space", "test collection", false},
		{"with special chars", "test@collection", false},
		{"empty string", "", false},
		{"just underscore", "_", true},
		{"starts with number", "123test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("isValidIdentifier(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCollectionTypes(t *testing.T) {
	// Verify all collection type constants are defined
	collectionTypes := []CollectionType{
		CollectionMemories,
		CollectionRemediations,
		CollectionCheckpoints,
		CollectionPolicies,
		CollectionSkills,
		CollectionAgents,
		CollectionSessions,
		CollectionCodebase,
		CollectionStandards,
		CollectionRepoStandards,
		CollectionAntiPatterns,
		CollectionFeedback,
	}

	for _, ct := range collectionTypes {
		if string(ct) == "" {
			t.Errorf("Collection type is empty: %v", ct)
		}
		// Verify they're valid identifiers
		if !isValidIdentifier(string(ct)) {
			t.Errorf("Collection type %q is not a valid identifier", ct)
		}
	}
}

func TestScopes(t *testing.T) {
	// Verify scope constants
	scopes := []Scope{ScopeOrg, ScopeTeam, ScopeProject}
	expectedValues := []string{"org", "team", "project"}

	for i, scope := range scopes {
		if string(scope) != expectedValues[i] {
			t.Errorf("Scope constant mismatch: %q != %q", scope, expectedValues[i])
		}
	}
}
