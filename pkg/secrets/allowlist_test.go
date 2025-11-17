package secrets

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAllowlists_ProjectOnly(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, ".gitleaks.toml")

	// Write valid project allowlist
	content := `[allowlist]
paths = [
  '''test/fixtures/.*\.env''',
  '''docs/examples/.*'''
]
regexes = [
  '''DEMO_API_KEY''',
  '''EXAMPLE_SECRET_.*'''
]
`
	if err := os.WriteFile(projectFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load allowlists
	allowlist, err := LoadAllowlists(tmpDir, "")
	if err != nil {
		t.Fatalf("LoadAllowlists() error = %v", err)
	}

	// Verify project patterns loaded
	if len(allowlist.Paths) != 2 {
		t.Errorf("got %d paths, want 2", len(allowlist.Paths))
	}
	if len(allowlist.Regexes) != 2 {
		t.Errorf("got %d regexes, want 2", len(allowlist.Regexes))
	}
}

func TestLoadAllowlists_UserOnly(t *testing.T) {
	tmpDir := t.TempDir()
	userFile := filepath.Join(tmpDir, "allowlist.toml")

	content := `[allowlist]
paths = [
  '''.*/demo-projects/.*'''
]
regexes = [
  '''MY_PERSONAL_DEMO_KEY'''
]
`
	if err := os.WriteFile(userFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load with empty project path
	allowlist, err := LoadAllowlists("", userFile)
	if err != nil {
		t.Fatalf("LoadAllowlists() error = %v", err)
	}

	if len(allowlist.Paths) != 1 {
		t.Errorf("got %d paths, want 1", len(allowlist.Paths))
	}
	if len(allowlist.Regexes) != 1 {
		t.Errorf("got %d regexes, want 1", len(allowlist.Regexes))
	}
}

func TestLoadAllowlists_BothMerged(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, ".gitleaks.toml")
	userFile := filepath.Join(tmpDir, "user-allowlist.toml")

	projectContent := `[allowlist]
paths = ['''project-path''']
regexes = ['''PROJECT_REGEX''']
`
	userContent := `[allowlist]
paths = ['''user-path''']
regexes = ['''USER_REGEX''']
`

	if err := os.WriteFile(projectFile, []byte(projectContent), 0600); err != nil {
		t.Fatalf("Failed to write project file: %v", err)
	}
	if err := os.WriteFile(userFile, []byte(userContent), 0600); err != nil {
		t.Fatalf("Failed to write user file: %v", err)
	}

	// Load both
	allowlist, err := LoadAllowlists(tmpDir, userFile)
	if err != nil {
		t.Fatalf("LoadAllowlists() error = %v", err)
	}

	// Verify union merge (both patterns present)
	if len(allowlist.Paths) != 2 {
		t.Errorf("got %d paths, want 2 (union merge)", len(allowlist.Paths))
	}
	if len(allowlist.Regexes) != 2 {
		t.Errorf("got %d regexes, want 2 (union merge)", len(allowlist.Regexes))
	}
}

func TestLoadAllowlists_MissingProjectFile(t *testing.T) {
	tmpDir := t.TempDir()
	// No .gitleaks.toml file created

	// Should succeed with empty allowlist (missing project file is OK)
	allowlist, err := LoadAllowlists(tmpDir, "")
	if err != nil {
		t.Fatalf("LoadAllowlists() should not error on missing project file: %v", err)
	}

	if len(allowlist.Paths) != 0 {
		t.Errorf("got %d paths, want 0 for missing file", len(allowlist.Paths))
	}
}

func TestLoadAllowlists_MissingUserFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.toml")

	// Should succeed with empty allowlist (missing user file is OK)
	allowlist, err := LoadAllowlists("", nonExistentFile)
	if err != nil {
		t.Fatalf("LoadAllowlists() should not error on missing user file: %v", err)
	}

	if len(allowlist.Paths) != 0 {
		t.Errorf("got %d paths, want 0 for missing file", len(allowlist.Paths))
	}
}

func TestLoadAllowlists_BothMissing(t *testing.T) {
	tmpDir := t.TempDir()

	// Both files missing - should return empty allowlist
	allowlist, err := LoadAllowlists(tmpDir, filepath.Join(tmpDir, "nonexistent.toml"))
	if err != nil {
		t.Fatalf("LoadAllowlists() should not error when both files missing: %v", err)
	}

	if allowlist == nil {
		t.Fatal("allowlist should not be nil")
	}
	if len(allowlist.Paths) != 0 || len(allowlist.Regexes) != 0 {
		t.Error("empty allowlist should have no patterns")
	}
}

func TestLoadAllowlists_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, ".gitleaks.toml")

	// Write invalid TOML
	invalidContent := `[allowlist
paths = "not a list"  # Missing closing bracket and wrong type
`
	if err := os.WriteFile(projectFile, []byte(invalidContent), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Should return error
	_, err := LoadAllowlists(tmpDir, "")
	if err == nil {
		t.Fatal("LoadAllowlists() should error on invalid TOML")
	}

	// Error should wrap ErrInvalidTOML
	if !errors.Is(err, ErrInvalidTOML) {
		t.Errorf("error should wrap ErrInvalidTOML, got: %v", err)
	}
}

func TestLoadAllowlists_InvalidRegex(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, ".gitleaks.toml")

	// Write allowlist with invalid regex pattern
	content := `[allowlist]
paths = []
regexes = [
  '''[unclosed bracket'''
]
`
	if err := os.WriteFile(projectFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Should fail-fast with clear error
	_, err := LoadAllowlists(tmpDir, "")
	if err == nil {
		t.Fatal("LoadAllowlists() should fail-fast on invalid regex")
	}

	// Error should wrap ErrInvalidRegex
	if !errors.Is(err, ErrInvalidRegex) {
		t.Errorf("error should wrap ErrInvalidRegex, got: %v", err)
	}

	// Error message should mention which pattern failed
	errMsg := err.Error()
	if !stringContains(errMsg, "unclosed bracket") {
		t.Errorf("error message should identify invalid pattern, got: %s", errMsg)
	}
}

func TestLoadAllowlists_EmptySections(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, ".gitleaks.toml")

	content := `[allowlist]
paths = []
regexes = []
`
	if err := os.WriteFile(projectFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	allowlist, err := LoadAllowlists(tmpDir, "")
	if err != nil {
		t.Fatalf("LoadAllowlists() should handle empty sections: %v", err)
	}

	if len(allowlist.Paths) != 0 || len(allowlist.Regexes) != 0 {
		t.Error("empty sections should result in no patterns")
	}
}

func TestLoadAllowlists_DuplicatePatterns(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, ".gitleaks.toml")
	userFile := filepath.Join(tmpDir, "user.toml")

	// Both files have same pattern
	sameContent := `[allowlist]
paths = ['''duplicate-pattern''']
regexes = ['''DUPLICATE_REGEX''']
`
	if err := os.WriteFile(projectFile, []byte(sameContent), 0600); err != nil {
		t.Fatalf("Failed to write project file: %v", err)
	}
	if err := os.WriteFile(userFile, []byte(sameContent), 0600); err != nil {
		t.Fatalf("Failed to write user file: %v", err)
	}

	allowlist, err := LoadAllowlists(tmpDir, userFile)
	if err != nil {
		t.Fatalf("LoadAllowlists() error = %v", err)
	}

	// Union merge will have duplicates (that's OK - no deduplication required)
	if len(allowlist.Paths) != 2 {
		t.Errorf("got %d paths, want 2 (duplicates allowed in union)", len(allowlist.Paths))
	}
}

func TestLoadAllowlists_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, ".gitleaks.toml")

	// Generate 1000 patterns
	var content string
	content += "[allowlist]\n"
	content += "paths = [\n"
	for i := 0; i < 1000; i++ {
		content += "  '''pattern" + string(rune('0'+i%10)) + ".*''',\n"
	}
	content += "]\n"
	content += "regexes = []\n"

	if err := os.WriteFile(projectFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Should handle large file without error
	allowlist, err := LoadAllowlists(tmpDir, "")
	if err != nil {
		t.Fatalf("LoadAllowlists() should handle large files: %v", err)
	}

	if len(allowlist.Paths) != 1000 {
		t.Errorf("got %d paths, want 1000", len(allowlist.Paths))
	}
}

func TestLoadAllowlists_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, ".gitleaks.toml")

	// Create file
	content := `[allowlist]
paths = ['''test''']
`
	if err := os.WriteFile(projectFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Remove read permission
	if err := os.Chmod(projectFile, 0000); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer os.Chmod(projectFile, 0600) // Restore for cleanup

	// Should return error (but not fail-fast - permission errors are runtime issues)
	_, err := LoadAllowlists(tmpDir, "")
	if err == nil {
		t.Fatal("LoadAllowlists() should error on permission denied")
	}
}

func TestAllowlist_Structure(t *testing.T) {
	a := Allowlist{
		Paths:   []string{"path1", "path2"},
		Regexes: []string{"regex1", "regex2"},
	}

	if len(a.Paths) != 2 {
		t.Errorf("Paths length = %d, want 2", len(a.Paths))
	}
	if len(a.Regexes) != 2 {
		t.Errorf("Regexes length = %d, want 2", len(a.Regexes))
	}
}

func TestLoadAllowlists_PathRegexValidation(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, ".gitleaks.toml")

	// Invalid path regex
	content := `[allowlist]
paths = ['''[invalid(regex''']
regexes = []
`
	if err := os.WriteFile(projectFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := LoadAllowlists(tmpDir, "")
	if err == nil {
		t.Fatal("LoadAllowlists() should fail on invalid path regex")
	}

	if !errors.Is(err, ErrInvalidRegex) {
		t.Errorf("error should wrap ErrInvalidRegex for path patterns, got: %v", err)
	}
}
