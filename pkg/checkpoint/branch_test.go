package checkpoint

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/pkg/vectorstore"
)

// TestDetectBranch tests git branch auto-detection
func TestDetectBranch(t *testing.T) {
	tests := []struct {
		name         string
		setupRepo    func(t *testing.T) string // Returns repo path
		expectedBranch string
		wantErr      bool
	}{
		{
			name: "detect main branch",
			setupRepo: func(t *testing.T) string {
				return setupTestGitRepo(t, "main")
			},
			expectedBranch: "main",
			wantErr:        false,
		},
		{
			name: "detect feature branch",
			setupRepo: func(t *testing.T) string {
				repo := setupTestGitRepo(t, "main")
				// Create and checkout feature branch
				runGitCommand(t, repo, "checkout", "-b", "feature/test-branch")
				return repo
			},
			expectedBranch: "feature/test-branch",
			wantErr:        false,
		},
		{
			name: "non-git directory returns empty",
			setupRepo: func(t *testing.T) string {
				// Create non-git directory
				tmpDir := t.TempDir()
				return tmpDir
			},
			expectedBranch: "",
			wantErr:        false, // Not an error, just no git repo
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := tt.setupRepo(t)

			branch, err := detectGitBranch(repoPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("detectGitBranch() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("detectGitBranch() unexpected error: %v", err)
				return
			}

			if branch != tt.expectedBranch {
				t.Errorf("detectGitBranch() = %q, want %q", branch, tt.expectedBranch)
			}
		})
	}
}

// TestService_Save_BranchAutoDetection tests that Save auto-detects branch
func TestService_Save_BranchAutoDetection(t *testing.T) {
	// Setup test git repo
	repoPath := setupTestGitRepo(t, "main")
	runGitCommand(t, repoPath, "checkout", "-b", "feature/auto-detect")

	// Create mock vector store
	mock := &mockVectorStore{
		addDocsFunc: func(ctx context.Context, docs []vectorstore.Document) error {
			// Verify branch was auto-detected
			if len(docs) != 1 {
				t.Fatalf("Expected 1 document, got %d", len(docs))
			}

			branch, ok := docs[0].Metadata["branch"].(string)
			if !ok {
				t.Error("branch metadata not found")
				return nil
			}

			if branch != "feature/auto-detect" {
				t.Errorf("branch = %q, want %q", branch, "feature/auto-detect")
			}

			return nil
		},
		getCollectionInfoFunc: func(ctx context.Context, collectionName string) (*vectorstore.CollectionInfo, error) {
			return &vectorstore.CollectionInfo{PointCount: 0}, nil
		},
	}

	service := &Service{
		vectorStore: mock,
		logger:      zap.NewNop(),
	}

	checkpoint := &Checkpoint{
		ProjectPath: repoPath,
		Summary:     "Test checkpoint with auto-detected branch",
		Content:     "Content here",
	}

	err := service.Save(context.Background(), checkpoint)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify branch was set on checkpoint
	if checkpoint.Branch != "feature/auto-detect" {
		t.Errorf("checkpoint.Branch = %q, want %q", checkpoint.Branch, "feature/auto-detect")
	}
}

// TestService_Search_BranchFilter tests branch filtering in search
func TestService_Search_BranchFilter(t *testing.T) {
	// Create mock with checkpoints with different branches
	mock := &mockVectorStore{
		searchCollectionFunc: func(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
			// Return checkpoints from different branches
			return []vectorstore.SearchResult{
				{
					ID: "1",
					Metadata: map[string]interface{}{
						"id":           "1",
						"project_path": "/test/project",
						"summary":      "Main branch checkpoint",
						"branch":       "main",
					},
					Score: 1.0,
				},
				{
					ID: "2",
					Metadata: map[string]interface{}{
						"id":           "2",
						"project_path": "/test/project",
						"summary":      "Feature branch checkpoint",
						"branch":       "feature/test",
					},
					Score: 1.0,
				},
				{
					ID: "3",
					Metadata: map[string]interface{}{
						"id":           "3",
						"project_path": "/test/project",
						"summary":      "Another main branch",
						"branch":       "main",
					},
					Score: 1.0,
				},
			}, nil
		},
		getCollectionInfoFunc: func(ctx context.Context, collectionName string) (*vectorstore.CollectionInfo, error) {
			return &vectorstore.CollectionInfo{PointCount: 10}, nil
		},
	}

	service := &Service{
		vectorStore: mock,
		logger:      zap.NewNop(),
	}

	// Search with branch filter
	results, err := service.Search(context.Background(), "checkpoint", &SearchOptions{
		ProjectPath: "/test/project",
		Limit:       10,
		Branch:      "main", // Filter to main branch only
	})

	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Should only return main branch checkpoints
	if len(results) != 2 {
		t.Errorf("Search() returned %d results, want 2 (only main branch)", len(results))
	}

	for _, result := range results {
		if result.Checkpoint.Branch != "main" {
			t.Errorf("Search() returned checkpoint with branch %q, want main", result.Checkpoint.Branch)
		}
	}
}

// Helper functions

// setupTestGitRepo creates a temporary git repository for testing
func setupTestGitRepo(t *testing.T, initialBranch string) string {
	t.Helper()

	// Create temp directory
	repoPath := t.TempDir()

	// Initialize git repo
	runGitCommand(t, repoPath, "init", "-b", initialBranch)
	runGitCommand(t, repoPath, "config", "user.name", "Test User")
	runGitCommand(t, repoPath, "config", "user.email", "test@example.com")

	// Create initial commit
	readmePath := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}

	runGitCommand(t, repoPath, "add", "README.md")
	runGitCommand(t, repoPath, "commit", "-m", "Initial commit")

	return repoPath
}

// runGitCommand runs a git command in the specified directory
func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, output)
	}
}
