package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectBranch(t *testing.T) {
	tests := []struct {
		name       string
		setupRepo  func(t *testing.T) string
		want       string
		wantErr    bool
		errMessage string
	}{
		{
			name: "main branch",
			setupRepo: func(t *testing.T) string {
				dir := t.TempDir()
				gitDir := filepath.Join(dir, ".git")
				require.NoError(t, os.Mkdir(gitDir, 0755))
				headFile := filepath.Join(gitDir, "HEAD")
				require.NoError(t, os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0644))
				return dir
			},
			want:    "main",
			wantErr: false,
		},
		{
			name: "feature branch",
			setupRepo: func(t *testing.T) string {
				dir := t.TempDir()
				gitDir := filepath.Join(dir, ".git")
				require.NoError(t, os.Mkdir(gitDir, 0755))
				headFile := filepath.Join(gitDir, "HEAD")
				require.NoError(t, os.WriteFile(headFile, []byte("ref: refs/heads/feature/v3-rebuild\n"), 0644))
				return dir
			},
			want:    "feature/v3-rebuild",
			wantErr: false,
		},
		{
			name: "master branch",
			setupRepo: func(t *testing.T) string {
				dir := t.TempDir()
				gitDir := filepath.Join(dir, ".git")
				require.NoError(t, os.Mkdir(gitDir, 0755))
				headFile := filepath.Join(gitDir, "HEAD")
				require.NoError(t, os.WriteFile(headFile, []byte("ref: refs/heads/master\n"), 0644))
				return dir
			},
			want:    "master",
			wantErr: false,
		},
		{
			name: "detached HEAD",
			setupRepo: func(t *testing.T) string {
				dir := t.TempDir()
				gitDir := filepath.Join(dir, ".git")
				require.NoError(t, os.Mkdir(gitDir, 0755))
				headFile := filepath.Join(gitDir, "HEAD")
				require.NoError(t, os.WriteFile(headFile, []byte("abc123def456789\n"), 0644))
				return dir
			},
			want:    "detached",
			wantErr: false,
		},
		{
			name: "non-git directory",
			setupRepo: func(t *testing.T) string {
				return t.TempDir()
			},
			want:       "",
			wantErr:    true,
			errMessage: "not a git repository",
		},
		{
			name: "missing HEAD file",
			setupRepo: func(t *testing.T) string {
				dir := t.TempDir()
				gitDir := filepath.Join(dir, ".git")
				require.NoError(t, os.Mkdir(gitDir, 0755))
				return dir
			},
			want:       "",
			wantErr:    true,
			errMessage: "HEAD file not found",
		},
		{
			name: "empty HEAD file",
			setupRepo: func(t *testing.T) string {
				dir := t.TempDir()
				gitDir := filepath.Join(dir, ".git")
				require.NoError(t, os.Mkdir(gitDir, 0755))
				headFile := filepath.Join(gitDir, "HEAD")
				require.NoError(t, os.WriteFile(headFile, []byte(""), 0644))
				return dir
			},
			want:    "detached",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectPath := tt.setupRepo(t)
			got, err := DetectBranch(projectPath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIsMainBranch(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		want   bool
	}{
		{"main", "main", true},
		{"master", "master", true},
		{"develop", "develop", false},
		{"feature branch", "feature/auth", false},
		{"bugfix branch", "bugfix/security", false},
		{"detached", "detached", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMainBranch(tt.branch)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectBranchRealRepo(t *testing.T) {
	// Integration test: detect branch in actual contextd repo
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Get project root (3 levels up from pkg/git/)
	projectRoot := filepath.Join("..", "..")
	branch, err := DetectBranch(projectRoot)
	require.NoError(t, err)
	assert.NotEmpty(t, branch)
	t.Logf("Current branch: %s", branch)
}
