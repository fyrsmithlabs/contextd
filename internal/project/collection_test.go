package project

import (
	"strings"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/sanitize"
	"github.com/google/uuid"
)

func TestGetCollectionName(t *testing.T) {
	projectID := uuid.New().String()
	sanitizedID := sanitize.Identifier(projectID)

	tests := []struct {
		name           string
		projectID      string
		collectionType CollectionType
		want           string
		wantErr        bool
	}{
		{
			name:           "memories collection",
			projectID:      projectID,
			collectionType: CollectionMemories,
			want:           sanitizedID + "_memories",
			wantErr:        false,
		},
		{
			name:           "checkpoints collection",
			projectID:      projectID,
			collectionType: CollectionCheckpoints,
			want:           sanitizedID + "_checkpoints",
			wantErr:        false,
		},
		{
			name:           "remediations collection",
			projectID:      projectID,
			collectionType: CollectionRemediations,
			want:           sanitizedID + "_remediations",
			wantErr:        false,
		},
		{
			name:           "sessions collection",
			projectID:      projectID,
			collectionType: CollectionSessions,
			want:           sanitizedID + "_sessions",
			wantErr:        false,
		},
		{
			name:           "codebase collection",
			projectID:      projectID,
			collectionType: CollectionCodebase,
			want:           sanitizedID + "_codebase",
			wantErr:        false,
		},
		{
			name:           "hyphenated project ID",
			projectID:      "simple-ctl",
			collectionType: CollectionMemories,
			want:           "simple_ctl_memories",
			wantErr:        false,
		},
		{
			name:           "empty project ID",
			projectID:      "",
			collectionType: CollectionMemories,
			want:           "",
			wantErr:        true,
		},
		{
			name:           "empty collection type",
			projectID:      projectID,
			collectionType: "",
			want:           "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCollectionName(tt.projectID, tt.collectionType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCollectionName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetCollectionName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAllCollectionNames(t *testing.T) {
	projectID := uuid.New().String()
	sanitizedID := sanitize.Identifier(projectID)

	tests := []struct {
		name      string
		projectID string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "valid project ID",
			projectID: projectID,
			wantCount: 5, // memories, checkpoints, remediations, sessions, codebase
			wantErr:   false,
		},
		{
			name:      "empty project ID",
			projectID: "",
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAllCollectionNames(tt.projectID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllCollectionNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != tt.wantCount {
					t.Errorf("GetAllCollectionNames() returned %d collections, want %d", len(got), tt.wantCount)
				}

				// Verify all expected collections are present (using sanitized ID)
				expectedSuffixes := []string{"memories", "checkpoints", "remediations", "sessions", "codebase"}
				for _, suffix := range expectedSuffixes {
					expected := sanitizedID + "_" + suffix
					found := false
					for _, name := range got {
						if name == expected {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("GetAllCollectionNames() missing expected collection: %s", expected)
					}
				}
			}
		})
	}
}

func TestGetCollectionName_Sanitization(t *testing.T) {
	// Test that various project IDs are properly sanitized
	tests := []struct {
		projectID string
		wantBase  string
	}{
		{"simple-ctl", "simple_ctl"},
		{"my-cool-project", "my_cool_project"},
		{"Project.Name", "project_name"},
		{"user/repo", "user_repo"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.projectID, func(t *testing.T) {
			got, err := GetCollectionName(tt.projectID, CollectionMemories)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := tt.wantBase + "_memories"
			if got != want {
				t.Errorf("GetCollectionName(%q) = %q, want %q", tt.projectID, got, want)
			}

			// Verify result matches collection name pattern
			if strings.Contains(got, "-") {
				t.Errorf("GetCollectionName(%q) = %q contains hyphen", tt.projectID, got)
			}
		})
	}
}
