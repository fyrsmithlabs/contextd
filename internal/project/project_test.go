package project

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewProject(t *testing.T) {
	tests := []struct {
		name     string
		projName string
		path     string
		wantErr  bool
	}{
		{
			name:     "valid project",
			projName: "my-project",
			path:     "/home/user/projects/my-project",
			wantErr:  false,
		},
		{
			name:     "empty name",
			projName: "",
			path:     "/home/user/projects/my-project",
			wantErr:  true,
		},
		{
			name:     "empty path",
			projName: "my-project",
			path:     "",
			wantErr:  true,
		},
		{
			name:     "both empty",
			projName: "",
			path:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := NewProject(tt.projName, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify project fields
				if project.Name != tt.projName {
					t.Errorf("project.Name = %v, want %v", project.Name, tt.projName)
				}
				if project.Path != tt.path {
					t.Errorf("project.Path = %v, want %v", project.Path, tt.path)
				}
				if project.ID == "" {
					t.Error("project.ID should not be empty")
				}
				if _, err := uuid.Parse(project.ID); err != nil {
					t.Errorf("project.ID should be valid UUID: %v", err)
				}
				if project.CreatedAt.IsZero() {
					t.Error("project.CreatedAt should not be zero")
				}
				if project.UpdatedAt.IsZero() {
					t.Error("project.UpdatedAt should not be zero")
				}
			}
		})
	}
}

func TestProject_Validate(t *testing.T) {
	tests := []struct {
		name    string
		project *Project
		wantErr error
	}{
		{
			name: "valid project",
			project: &Project{
				ID:   uuid.New().String(),
				Name: "test-project",
				Path: "/home/user/test",
			},
			wantErr: nil,
		},
		{
			name: "empty ID",
			project: &Project{
				ID:   "",
				Name: "test-project",
				Path: "/home/user/test",
			},
			wantErr: ErrEmptyProjectID,
		},
		{
			name: "invalid UUID",
			project: &Project{
				ID:   "not-a-uuid",
				Name: "test-project",
				Path: "/home/user/test",
			},
			wantErr: ErrInvalidProjectID,
		},
		{
			name: "empty name",
			project: &Project{
				ID:   uuid.New().String(),
				Name: "",
				Path: "/home/user/test",
			},
			wantErr: ErrEmptyProjectName,
		},
		{
			name: "empty path",
			project: &Project{
				ID:   uuid.New().String(),
				Name: "test-project",
				Path: "",
			},
			wantErr: ErrEmptyProjectPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.Validate()
			if err != tt.wantErr {
				t.Errorf("Project.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
