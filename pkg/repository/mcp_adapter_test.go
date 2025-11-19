package repository

import (
	"context"
	"testing"

	"github.com/fyrsmithlabs/contextd/pkg/checkpoint"
	"github.com/fyrsmithlabs/contextd/pkg/mcp"
)

// TestMCPAdapter_IndexRepository tests the MCP adapter.
func TestMCPAdapter_IndexRepository(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "test.txt", "test content")

	mockCheckpoint := &mockCheckpointService{
		savedCheckpoints: make([]*checkpoint.Checkpoint, 0),
	}
	svc := NewService(mockCheckpoint)
	adapter := NewMCPAdapter(svc)

	opts := mcp.RepositoryIndexOptions{
		IncludePatterns: []string{"*.txt"},
		MaxFileSize:     1024 * 1024,
	}

	result, err := adapter.IndexRepository(context.Background(), tmpDir, opts)

	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	if result.FilesIndexed != 1 {
		t.Errorf("FilesIndexed = %d, want 1", result.FilesIndexed)
	}

	if result.Path != tmpDir {
		t.Errorf("Path = %s, want %s", result.Path, tmpDir)
	}
}

// TestMCPAdapter_IndexRepository_Error tests error handling.
func TestMCPAdapter_IndexRepository_Error(t *testing.T) {
	mockCheckpoint := &mockCheckpointService{}
	svc := NewService(mockCheckpoint)
	adapter := NewMCPAdapter(svc)

	opts := mcp.RepositoryIndexOptions{}
	_, err := adapter.IndexRepository(context.Background(), "/nonexistent", opts)

	if err == nil {
		t.Fatal("IndexRepository() error = nil, want error")
	}
}
