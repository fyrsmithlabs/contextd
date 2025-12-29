package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid alphanumeric", "myproject", false},
		{"valid with hyphen", "my-project", false},
		{"valid with underscore", "my_project", false},
		{"valid with dot", "my.project", false},
		{"valid with numbers", "project123", false},
		{"valid mixed", "My-Project_123.v2", false},
		{"empty", "", true},
		{"starts with hyphen", "-project", true},
		{"starts with dot", ".project", true},
		{"path traversal dot", ".", true},
		{"path traversal dotdot", "..", true},
		{"contains slash", "my/project", true},
		{"contains backslash", "my\\project", true},
		{"contains space", "my project", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestRegistry_RegisterTenant(t *testing.T) {
	tmpDir := t.TempDir()

	r, err := NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Register new tenant
	entry, err := r.RegisterTenant("acme")
	if err != nil {
		t.Fatalf("RegisterTenant failed: %v", err)
	}
	if entry.Name != "acme" {
		t.Errorf("entry.Name = %q, want %q", entry.Name, "acme")
	}
	if entry.UUID == "" {
		t.Error("entry.UUID is empty")
	}

	// Verify directory created
	tenantPath := filepath.Join(tmpDir, "acme")
	if _, err := os.Stat(tenantPath); os.IsNotExist(err) {
		t.Errorf("tenant directory not created: %s", tenantPath)
	}

	// Register same tenant again (idempotent)
	entry2, err := r.RegisterTenant("acme")
	if err != nil {
		t.Fatalf("RegisterTenant (idempotent) failed: %v", err)
	}
	if entry2.UUID != entry.UUID {
		t.Errorf("UUID changed on re-registration: %s != %s", entry2.UUID, entry.UUID)
	}

	// Register invalid tenant
	_, err = r.RegisterTenant("../evil")
	if err == nil {
		t.Error("expected error for path traversal, got nil")
	}
}

func TestRegistry_RegisterProject(t *testing.T) {
	tmpDir := t.TempDir()

	r, err := NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Register tenant first
	_, err = r.RegisterTenant("acme")
	if err != nil {
		t.Fatalf("RegisterTenant failed: %v", err)
	}

	// Register project directly under tenant (no team)
	entry, err := r.RegisterProject("acme", "", "contextd")
	if err != nil {
		t.Fatalf("RegisterProject failed: %v", err)
	}
	if entry.Name != "contextd" {
		t.Errorf("entry.Name = %q, want %q", entry.Name, "contextd")
	}

	// Verify directory created
	projectPath := filepath.Join(tmpDir, "acme", "contextd")
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Errorf("project directory not created: %s", projectPath)
	}

	// Register project without tenant should fail
	_, err = r.RegisterProject("nonexistent", "", "myproject")
	if err != ErrTenantNotFound {
		t.Errorf("expected ErrTenantNotFound, got %v", err)
	}
}

func TestRegistry_RegisterTeamAndProject(t *testing.T) {
	tmpDir := t.TempDir()

	r, err := NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Register tenant
	_, err = r.RegisterTenant("acme")
	if err != nil {
		t.Fatalf("RegisterTenant failed: %v", err)
	}

	// Register team
	team, err := r.RegisterTeam("acme", "platform")
	if err != nil {
		t.Fatalf("RegisterTeam failed: %v", err)
	}
	if team.Name != "platform" {
		t.Errorf("team.Name = %q, want %q", team.Name, "platform")
	}

	// Register project under team
	proj, err := r.RegisterProject("acme", "platform", "contextd")
	if err != nil {
		t.Fatalf("RegisterProject with team failed: %v", err)
	}
	if proj.Name != "contextd" {
		t.Errorf("proj.Name = %q, want %q", proj.Name, "contextd")
	}

	// Verify directory structure
	projectPath := filepath.Join(tmpDir, "acme", "platform", "contextd")
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Errorf("project directory not created: %s", projectPath)
	}

	// Register project with nonexistent team should fail
	_, err = r.RegisterProject("acme", "nonexistent", "myproject")
	if err != ErrTeamNotFound {
		t.Errorf("expected ErrTeamNotFound, got %v", err)
	}
}

func TestRegistry_GetPaths(t *testing.T) {
	tmpDir := t.TempDir()

	r, err := NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Test GetProjectPath (direct project)
	path, err := r.GetProjectPath("acme", "", "contextd")
	if err != nil {
		t.Fatalf("GetProjectPath failed: %v", err)
	}
	expected := filepath.Join(tmpDir, "acme", "contextd")
	if path != expected {
		t.Errorf("GetProjectPath = %q, want %q", path, expected)
	}

	// Test GetProjectPath (team-scoped project)
	path, err = r.GetProjectPath("acme", "platform", "contextd")
	if err != nil {
		t.Fatalf("GetProjectPath with team failed: %v", err)
	}
	expected = filepath.Join(tmpDir, "acme", "platform", "contextd")
	if path != expected {
		t.Errorf("GetProjectPath = %q, want %q", path, expected)
	}

	// Test GetTeamPath
	path, err = r.GetTeamPath("acme", "platform")
	if err != nil {
		t.Fatalf("GetTeamPath failed: %v", err)
	}
	expected = filepath.Join(tmpDir, "acme", "platform")
	if path != expected {
		t.Errorf("GetTeamPath = %q, want %q", path, expected)
	}

	// Test GetOrgPath
	path, err = r.GetOrgPath("acme")
	if err != nil {
		t.Fatalf("GetOrgPath failed: %v", err)
	}
	expected = filepath.Join(tmpDir, "acme")
	if path != expected {
		t.Errorf("GetOrgPath = %q, want %q", path, expected)
	}

	// Test invalid paths
	_, err = r.GetProjectPath("../evil", "", "contextd")
	if err == nil {
		t.Error("expected error for path traversal, got nil")
	}
}

func TestRegistry_EnsureProjectExists(t *testing.T) {
	tmpDir := t.TempDir()

	r, err := NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// EnsureProjectExists should auto-create tenant and project
	err = r.EnsureProjectExists("neworg", "", "newproject")
	if err != nil {
		t.Fatalf("EnsureProjectExists failed: %v", err)
	}

	// Verify tenant was created
	tenant, err := r.GetTenant("neworg")
	if err != nil {
		t.Fatalf("GetTenant failed: %v", err)
	}
	if tenant.Name != "neworg" {
		t.Errorf("tenant.Name = %q, want %q", tenant.Name, "neworg")
	}

	// Verify project was created
	proj, err := r.GetProject("neworg", "", "newproject")
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if proj.Name != "newproject" {
		t.Errorf("proj.Name = %q, want %q", proj.Name, "newproject")
	}
}

func TestRegistry_Persistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create registry and add data
	r1, err := NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	_, err = r1.RegisterTenant("acme")
	if err != nil {
		t.Fatalf("RegisterTenant failed: %v", err)
	}

	entry, err := r1.RegisterProject("acme", "", "contextd")
	if err != nil {
		t.Fatalf("RegisterProject failed: %v", err)
	}
	originalUUID := entry.UUID

	// Create new registry instance (simulates restart)
	r2, err := NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("NewRegistry (reload) failed: %v", err)
	}

	// Verify data persisted
	tenant, err := r2.GetTenant("acme")
	if err != nil {
		t.Fatalf("GetTenant (after reload) failed: %v", err)
	}
	if tenant.Name != "acme" {
		t.Errorf("tenant.Name = %q, want %q", tenant.Name, "acme")
	}

	proj, err := r2.GetProject("acme", "", "contextd")
	if err != nil {
		t.Fatalf("GetProject (after reload) failed: %v", err)
	}
	if proj.UUID != originalUUID {
		t.Errorf("UUID changed after reload: %s != %s", proj.UUID, originalUUID)
	}
}

func TestRegistry_ListTenants(t *testing.T) {
	tmpDir := t.TempDir()

	r, err := NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Register some tenants
	r.RegisterTenant("acme")
	r.RegisterTenant("globex")
	r.RegisterTenant("initech")

	tenants := r.ListTenants()
	if len(tenants) != 3 {
		t.Errorf("ListTenants returned %d tenants, want 3", len(tenants))
	}

	// Check all tenants present (order not guaranteed)
	found := make(map[string]bool)
	for _, name := range tenants {
		found[name] = true
	}
	for _, want := range []string{"acme", "globex", "initech"} {
		if !found[want] {
			t.Errorf("tenant %q not found in list", want)
		}
	}
}

func TestRegistry_ListProjects(t *testing.T) {
	tmpDir := t.TempDir()

	r, err := NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Register tenant and projects
	r.RegisterTenant("acme")
	r.RegisterProject("acme", "", "project1")
	r.RegisterProject("acme", "", "project2")

	// Register another tenant with project
	r.RegisterTenant("other")
	r.RegisterProject("other", "", "otherproject")

	// List projects for acme
	projects := r.ListProjects("acme")
	if len(projects) != 2 {
		t.Errorf("ListProjects returned %d projects, want 2", len(projects))
	}

	// Verify other tenant's project not included
	for _, p := range projects {
		if p == "other/otherproject" {
			t.Error("found other tenant's project in acme's list")
		}
	}
}
