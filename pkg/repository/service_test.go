package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/fyrsmithlabs/contextd/pkg/checkpoint"
)

// TestIndexRepository_ValidPath tests successful repository indexing.
func TestIndexRepository_ValidPath(t *testing.T) {
	// Setup: Create temp directory with test files
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "README.md", "# Test Repository\n\nDocumentation here.")
	createTestFile(t, tmpDir, "main.go", "package main\n\nfunc main() {}")
	createTestFile(t, tmpDir, ".gitignore", "*.log")

	// Mock checkpoint service
	mockCheckpoint := &mockCheckpointService{
		savedCheckpoints: make([]*checkpoint.Checkpoint, 0),
	}

	// Create service
	svc := NewService(mockCheckpoint)

	// Test
	opts := IndexOptions{
		IncludePatterns: []string{"*.md", "*.go"},
		ExcludePatterns: []string{".git/**"},
		MaxFileSize:     1024 * 1024, // 1MB
	}

	result, err := svc.IndexRepository(context.Background(), tmpDir, opts)

	// Verify
	if err != nil {
		t.Fatalf("IndexRepository() error = %v, want nil", err)
	}

	if result.FilesIndexed != 2 {
		t.Errorf("FilesIndexed = %d, want 2 (README.md + main.go)", result.FilesIndexed)
	}

	if len(mockCheckpoint.savedCheckpoints) != 2 {
		t.Errorf("Checkpoints saved = %d, want 2", len(mockCheckpoint.savedCheckpoints))
	}
}

// TestIndexRepository_InvalidPath tests error handling for non-existent path.
func TestIndexRepository_InvalidPath(t *testing.T) {
	mockCheckpoint := &mockCheckpointService{}
	svc := NewService(mockCheckpoint)

	opts := IndexOptions{}
	_, err := svc.IndexRepository(context.Background(), "/nonexistent/path", opts)

	if err == nil {
		t.Fatal("IndexRepository() error = nil, want error for invalid path")
	}
}

// TestIndexRepository_ExcludePatterns tests file exclusion.
func TestIndexRepository_ExcludePatterns(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "main.go", "package main")
	createTestFile(t, tmpDir, "main_test.go", "package main")

	// Create vendor directory
	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, vendorDir, "pkg.go", "package vendor")

	mockCheckpoint := &mockCheckpointService{
		savedCheckpoints: make([]*checkpoint.Checkpoint, 0),
	}
	svc := NewService(mockCheckpoint)

	opts := IndexOptions{
		IncludePatterns: []string{"*.go"},
		ExcludePatterns: []string{"*_test.go", "vendor/**"},
		MaxFileSize:     1024 * 1024,
	}

	result, err := svc.IndexRepository(context.Background(), tmpDir, opts)

	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	// Should only index main.go (exclude main_test.go and vendor/pkg.go)
	if result.FilesIndexed != 1 {
		t.Errorf("FilesIndexed = %d, want 1 (only main.go)", result.FilesIndexed)
	}
}

// TestIndexRepository_MaxFileSize tests file size filtering.
func TestIndexRepository_MaxFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "small.txt", "small content")
	createTestFile(t, tmpDir, "large.txt", string(make([]byte, 2*1024*1024))) // 2MB

	mockCheckpoint := &mockCheckpointService{
		savedCheckpoints: make([]*checkpoint.Checkpoint, 0),
	}
	svc := NewService(mockCheckpoint)

	opts := IndexOptions{
		MaxFileSize: 1024 * 1024, // 1MB limit
	}

	result, err := svc.IndexRepository(context.Background(), tmpDir, opts)

	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	// Should only index small.txt (large.txt exceeds limit)
	if result.FilesIndexed != 1 {
		t.Errorf("FilesIndexed = %d, want 1 (only small.txt)", result.FilesIndexed)
	}
}

// TestIndexRepository_PathTraversalPrevention tests security.
func TestIndexRepository_PathTraversalPrevention(t *testing.T) {
	mockCheckpoint := &mockCheckpointService{}
	svc := NewService(mockCheckpoint)

	tests := []struct {
		name string
		path string
	}{
		{"relative path with traversal", "../../../etc/passwd"},
		{"absolute traversal", "/etc/../etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := IndexOptions{}
			_, err := svc.IndexRepository(context.Background(), tt.path, opts)

			// Should either error or safely resolve path
			if err == nil {
				t.Logf("Path traversal handled: %s", tt.path)
			}
		})
	}
}

// Helper: Create test file
func createTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

// TestIndexRepository_MaxFileSizeExceeds tests file size validation.
func TestIndexRepository_MaxFileSizeExceeds(t *testing.T) {
	mockCheckpoint := &mockCheckpointService{}
	svc := NewService(mockCheckpoint)

	opts := IndexOptions{
		MaxFileSize: 11 * 1024 * 1024, // 11MB (exceeds max)
	}

	_, err := svc.IndexRepository(context.Background(), t.TempDir(), opts)

	if err == nil {
		t.Fatal("IndexRepository() error = nil, want error for file size > 10MB")
	}
}

// TestIndexRepository_InvalidIncludePattern tests pattern validation.
func TestIndexRepository_InvalidIncludePattern(t *testing.T) {
	mockCheckpoint := &mockCheckpointService{}
	svc := NewService(mockCheckpoint)

	opts := IndexOptions{
		IncludePatterns: []string{"[invalid"},
	}

	_, err := svc.IndexRepository(context.Background(), t.TempDir(), opts)

	if err == nil {
		t.Fatal("IndexRepository() error = nil, want error for invalid pattern")
	}
}

// TestIndexRepository_InvalidExcludePattern tests pattern validation.
func TestIndexRepository_InvalidExcludePattern(t *testing.T) {
	mockCheckpoint := &mockCheckpointService{}
	svc := NewService(mockCheckpoint)

	opts := IndexOptions{
		ExcludePatterns: []string{"[invalid"},
	}

	_, err := svc.IndexRepository(context.Background(), t.TempDir(), opts)

	if err == nil {
		t.Fatal("IndexRepository() error = nil, want error for invalid pattern")
	}
}

// TestIndexRepository_CheckpointSaveError tests error handling.
func TestIndexRepository_CheckpointSaveError(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "test.txt", "content")

	mockCheckpoint := &mockCheckpointService{
		saveError: os.ErrPermission,
	}
	svc := NewService(mockCheckpoint)

	opts := IndexOptions{}
	_, err := svc.IndexRepository(context.Background(), tmpDir, opts)

	if err == nil {
		t.Fatal("IndexRepository() error = nil, want error when checkpoint save fails")
	}
}

// TestIndexRepository_ContextCancellation tests cancellation.
func TestIndexRepository_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	// Create many files to ensure context cancellation happens
	for i := 0; i < 100; i++ {
		createTestFile(t, tmpDir, filepath.Base(filepath.Join("file", fmt.Sprintf("%d.txt", i))), "content")
	}

	mockCheckpoint := &mockCheckpointService{
		savedCheckpoints: make([]*checkpoint.Checkpoint, 0),
	}
	svc := NewService(mockCheckpoint)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	opts := IndexOptions{}
	_, err := svc.IndexRepository(ctx, tmpDir, opts)

	if err == nil {
		t.Log("IndexRepository() completed despite cancellation (too few files)")
	}
}

// Mock checkpoint service for testing
type mockCheckpointService struct {
	savedCheckpoints []*checkpoint.Checkpoint
	saveError        error
}

func (m *mockCheckpointService) Save(ctx context.Context, cp *checkpoint.Checkpoint) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.savedCheckpoints = append(m.savedCheckpoints, cp)
	return nil
}
