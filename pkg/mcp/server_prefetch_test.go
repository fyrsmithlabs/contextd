package mcp

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap/zaptest"

	"github.com/fyrsmithlabs/contextd/pkg/config"
	"github.com/fyrsmithlabs/contextd/pkg/prefetch"
)

// TestServer_InitializePrefetch tests prefetch initialization.
func TestServer_InitializePrefetch(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.PreFetchConfig
		expectEnabled bool
	}{
		{
			name: "enabled",
			cfg: &config.PreFetchConfig{
				Enabled:         true,
				CacheTTL:        5 * time.Minute,
				CacheMaxEntries: 100,
			},
			expectEnabled: true,
		},
		{
			name:          "nil config",
			cfg:           nil,
			expectEnabled: false,
		},
		{
			name: "disabled",
			cfg: &config.PreFetchConfig{
				Enabled: false,
			},
			expectEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			server := &Server{echo: e}
			logger := zaptest.NewLogger(t)

			err := server.InitializePrefetch(tt.cfg, logger)
			if err != nil {
				t.Fatalf("InitializePrefetch() error = %v", err)
			}

			if server.prefetchEnabled != tt.expectEnabled {
				t.Errorf("prefetchEnabled = %v, want %v", server.prefetchEnabled, tt.expectEnabled)
			}

			if tt.expectEnabled {
				if server.prefetchCache == nil {
					t.Error("expected non-nil prefetchCache")
				}
				if server.prefetchExecutor == nil {
					t.Error("expected non-nil prefetchExecutor")
				}
				if server.prefetchDetectors == nil {
					t.Error("expected non-nil prefetchDetectors map")
				}
			}
		})
	}
}

// TestServer_StartStopDetector tests detector lifecycle.
func TestServer_StartStopDetector(t *testing.T) {
	e := echo.New()
	server := &Server{echo: e}
	logger := zaptest.NewLogger(t)

	cfg := &config.PreFetchConfig{
		Enabled:         true,
		CacheTTL:        5 * time.Minute,
		CacheMaxEntries: 100,
	}

	err := server.InitializePrefetch(cfg, logger)
	if err != nil {
		t.Fatalf("InitializePrefetch() error = %v", err)
	}

	// Create test git repo
	tmpDir, cleanup := createTestGitRepo(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start detector
	err = server.StartPrefetchDetector(ctx, tmpDir)
	if err != nil {
		t.Fatalf("StartPrefetchDetector() error = %v", err)
	}

	// Verify detector exists
	server.prefetchMu.RLock()
	detector, exists := server.prefetchDetectors[tmpDir]
	server.prefetchMu.RUnlock()

	if !exists {
		t.Fatal("detector not found after StartPrefetchDetector")
	}
	if detector == nil {
		t.Fatal("detector is nil")
	}

	// Stop detector
	server.StopPrefetchDetector(tmpDir)

	// Verify detector removed
	server.prefetchMu.RLock()
	_, exists = server.prefetchDetectors[tmpDir]
	server.prefetchMu.RUnlock()

	if exists {
		t.Error("detector still exists after StopPrefetchDetector")
	}

	// Should be safe to stop again
	server.StopPrefetchDetector(tmpDir)
}

// TestServer_GetPrefetchResults tests cache retrieval.
func TestServer_GetPrefetchResults(t *testing.T) {
	e := echo.New()
	server := &Server{echo: e}
	logger := zaptest.NewLogger(t)

	cfg := &config.PreFetchConfig{
		Enabled:         true,
		CacheTTL:        5 * time.Minute,
		CacheMaxEntries: 100,
	}

	err := server.InitializePrefetch(cfg, logger)
	if err != nil {
		t.Fatalf("InitializePrefetch() error = %v", err)
	}

	projectPath := "/test/project"

	// Test cache miss
	results := server.GetPrefetchResults(projectPath)
	if len(results) != 0 {
		t.Errorf("expected 0 results for cache miss, got %d", len(results))
	}

	// Populate cache
	testResults := []prefetch.PreFetchResult{
		{
			Type: "branch_diff",
			Data: map[string]interface{}{
				"summary": "test diff",
			},
			Metadata:   map[string]string{"branch": "main"},
			Confidence: 1.0,
		},
	}
	server.prefetchCache.Set(projectPath, testResults)

	// Test cache hit
	results = server.GetPrefetchResults(projectPath)
	if len(results) != 1 {
		t.Errorf("expected 1 result for cache hit, got %d", len(results))
	}

	if len(results) > 0 && results[0].Type != "branch_diff" {
		t.Errorf("expected result type 'branch_diff', got '%s'", results[0].Type)
	}
}

// TestServer_Shutdown tests shutdown cleans up detectors.
func TestServer_Shutdown(t *testing.T) {
	e := echo.New()
	server := &Server{echo: e}
	logger := zaptest.NewLogger(t)

	cfg := &config.PreFetchConfig{
		Enabled:         true,
		CacheTTL:        5 * time.Minute,
		CacheMaxEntries: 100,
	}

	err := server.InitializePrefetch(cfg, logger)
	if err != nil {
		t.Fatalf("InitializePrefetch() error = %v", err)
	}

	// Create test git repos and start detectors
	tmpDir1, cleanup1 := createTestGitRepo(t)
	defer cleanup1()

	tmpDir2, cleanup2 := createTestGitRepo(t)
	defer cleanup2()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = server.StartPrefetchDetector(ctx, tmpDir1)
	_ = server.StartPrefetchDetector(ctx, tmpDir2)

	// Verify 2 detectors running
	server.prefetchMu.RLock()
	count := len(server.prefetchDetectors)
	server.prefetchMu.RUnlock()

	if count != 2 {
		t.Errorf("expected 2 detectors, got %d", count)
	}

	// Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	err = server.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	// Verify all detectors stopped
	server.prefetchMu.RLock()
	count = len(server.prefetchDetectors)
	server.prefetchMu.RUnlock()

	if count != 0 {
		t.Errorf("expected 0 detectors after shutdown, got %d", count)
	}
}

// Helper functions

func createTestGitRepo(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "mcp-prefetch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	// Initialize git repo
	if err := initGitRepo(tmpDir); err != nil {
		cleanup()
		t.Fatalf("failed to init git repo: %v", err)
	}

	return tmpDir, cleanup
}

func initGitRepo(dir string) error {
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.name", "Test User"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "commit", "--allow-empty", "-m", "Initial commit"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}
