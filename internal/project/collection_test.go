package project

import (
	"testing"

	"github.com/google/uuid"
)

func TestGetCollectionName(t *testing.T) {
	projectID := uuid.New().String()

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
			want:           projectID + "_memories",
			wantErr:        false,
		},
		{
			name:           "checkpoints collection",
			projectID:      projectID,
			collectionType: CollectionCheckpoints,
			want:           projectID + "_checkpoints",
			wantErr:        false,
		},
		{
			name:           "remediations collection",
			projectID:      projectID,
			collectionType: CollectionRemediations,
			want:           projectID + "_remediations",
			wantErr:        false,
		},
		{
			name:           "sessions collection",
			projectID:      projectID,
			collectionType: CollectionSessions,
			want:           projectID + "_sessions",
			wantErr:        false,
		},
		{
			name:           "codebase collection",
			projectID:      projectID,
			collectionType: CollectionCodebase,
			want:           projectID + "_codebase",
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

				// Verify all expected collections are present
				expectedSuffixes := []string{"memories", "checkpoints", "remediations", "sessions", "codebase"}
				for _, suffix := range expectedSuffixes {
					expected := projectID + "_" + suffix
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
