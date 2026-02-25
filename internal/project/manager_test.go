package project

import (
	"context"
	"testing"
)

func TestManager_Create(t *testing.T) {
	ctx := context.Background()
	mgr := NewManager()

	tests := []struct {
		name     string
		projName string
		path     string
		wantErr  bool
	}{
		{
			name:     "valid project",
			projName: "test-project",
			path:     "/home/user/test-project",
			wantErr:  false,
		},
		{
			name:     "empty name",
			projName: "",
			path:     "/home/user/test-project",
			wantErr:  true,
		},
		{
			name:     "empty path",
			projName: "test-project",
			path:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := mgr.Create(ctx, tt.projName, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Manager.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if project.Name != tt.projName {
					t.Errorf("project.Name = %v, want %v", project.Name, tt.projName)
				}
				if project.Path != tt.path {
					t.Errorf("project.Path = %v, want %v", project.Path, tt.path)
				}
			}
		})
	}
}

func TestManager_CreateDuplicate(t *testing.T) {
	ctx := context.Background()
	mgr := NewManager()

	// Create first project
	_, err := mgr.Create(ctx, "project1", "/home/user/test")
	if err != nil {
		t.Fatalf("Failed to create first project: %v", err)
	}

	// Try to create second project with same path
	_, err = mgr.Create(ctx, "project2", "/home/user/test")
	if err == nil {
		t.Error("Manager.Create() should fail for duplicate path")
	}
}

func TestManager_Get(t *testing.T) {
	ctx := context.Background()
	mgr := NewManager()

	// Create a project
	created, err := mgr.Create(ctx, "test-project", "/home/user/test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing project",
			id:      created.ID,
			wantErr: false,
		},
		{
			name:    "non-existent project",
			id:      "non-existent-id",
			wantErr: true,
		},
		{
			name:    "empty ID",
			id:      "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := mgr.Get(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Manager.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if project.ID != created.ID {
					t.Errorf("project.ID = %v, want %v", project.ID, created.ID)
				}
			}
		})
	}
}

func TestManager_List(t *testing.T) {
	ctx := context.Background()
	mgr := NewManager()

	// Initially empty
	projects, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("Manager.List() error = %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("Manager.List() returned %d projects, want 0", len(projects))
	}

	// Create multiple projects
	created := make([]*Project, 3)
	for i := 0; i < 3; i++ {
		p, err := mgr.Create(ctx, "project"+string(rune('1'+i)), "/home/user/project"+string(rune('1'+i)))
		if err != nil {
			t.Fatalf("Failed to create project %d: %v", i, err)
		}
		created[i] = p
	}

	// List should return all 3
	projects, err = mgr.List(ctx)
	if err != nil {
		t.Fatalf("Manager.List() error = %v", err)
	}
	if len(projects) != 3 {
		t.Errorf("Manager.List() returned %d projects, want 3", len(projects))
	}

	// Verify all created projects are in the list
	for _, createdProj := range created {
		found := false
		for _, listedProj := range projects {
			if listedProj.ID == createdProj.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Created project %s not found in list", createdProj.ID)
		}
	}
}

func TestManager_Delete(t *testing.T) {
	ctx := context.Background()
	mgr := NewManager()

	// Create a project
	created, err := mgr.Create(ctx, "test-project", "/home/user/test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing project",
			id:      created.ID,
			wantErr: false,
		},
		{
			name:    "non-existent project",
			id:      "non-existent-id",
			wantErr: true,
		},
		{
			name:    "empty ID",
			id:      "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.Delete(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Manager.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If delete succeeded, verify project is gone
			if !tt.wantErr {
				_, err := mgr.Get(ctx, tt.id)
				if err == nil {
					t.Error("Manager.Get() should fail after delete")
				}
			}
		})
	}
}

func TestManager_GetByPath(t *testing.T) {
	ctx := context.Background()
	mgr := NewManager()

	// Create a project
	created, err := mgr.Create(ctx, "test-project", "/home/user/test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "existing path",
			path:    "/home/user/test",
			wantErr: false,
		},
		{
			name:    "non-existent path",
			path:    "/home/user/nonexistent",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := mgr.GetByPath(ctx, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Manager.GetByPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if project.ID != created.ID {
					t.Errorf("project.ID = %v, want %v", project.ID, created.ID)
				}
				if project.Path != tt.path {
					t.Errorf("project.Path = %v, want %v", project.Path, tt.path)
				}
			}
		})
	}
}

func TestManager_ConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	mgr := NewManager()

	// Create a project
	created, err := mgr.Create(ctx, "test-project", "/home/user/test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Run concurrent operations
	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_, err := mgr.Get(ctx, created.ID)
			if err != nil {
				t.Errorf("Concurrent Get() failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestManager_DeleteCleansUpPath(t *testing.T) {
	ctx := context.Background()
	mgr := NewManager()

	path := "/home/user/test"

	// Create project
	created, err := mgr.Create(ctx, "test-project", path)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Verify GetByPath works
	_, err = mgr.GetByPath(ctx, path)
	if err != nil {
		t.Fatalf("GetByPath() failed: %v", err)
	}

	// Delete project
	err = mgr.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Verify GetByPath no longer finds it
	_, err = mgr.GetByPath(ctx, path)
	if err == nil {
		t.Error("GetByPath() should fail after delete")
	}

	// Verify we can create a new project with the same path
	_, err = mgr.Create(ctx, "new-project", path)
	if err != nil {
		t.Errorf("Create() should succeed after delete: %v", err)
	}
}
