package prefetch

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

// BenchmarkConcurrentGitEvents benchmarks handling of concurrent git events.
func BenchmarkConcurrentGitEvents(b *testing.B) {
	// Create temporary git repository
	tmpDir, err := os.MkdirTemp("", "prefetch-bench-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	if err := initGitRepoForBench(tmpDir); err != nil {
		b.Fatalf("failed to init git repo: %v", err)
	}

	// Create cache
	cache := NewCache(5*time.Minute, 100)

	// Create executor
	executor := NewExecutor(3)

	// Create rule registry
	registry := NewRuleRegistry(tmpDir)

	ctx := context.Background()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate git event
			event := GitEvent{
				Type:        EventTypeBranchSwitch,
				ProjectPath: tmpDir,
				OldBranch:   "main",
				NewBranch:   "feature",
				Timestamp:   time.Now(),
			}

			// Execute rules
			rules := registry.GetRulesForEvent(event.Type)
			results := executor.Execute(ctx, event, rules)

			// Store in cache
			cache.Set(event.ProjectPath, results)
		}
	})
}

// BenchmarkCacheOperations benchmarks cache get/set operations.
func BenchmarkCacheOperations(b *testing.B) {
	cache := NewCache(5*time.Minute, 100)

	results := []PreFetchResult{
		{
			Type: "branch_diff",
			Data: map[string]interface{}{
				"summary": "test diff",
				"files":   []string{"file1.go", "file2.go"},
			},
			Metadata:   map[string]string{"branch": "main"},
			Confidence: 1.0,
		},
	}

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.Set("/test/project", results)
		}
	})

	b.Run("Get-Hit", func(b *testing.B) {
		cache.Set("/test/project", results)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, ok := cache.Get("/test/project")
			if !ok {
				b.Fatal("expected cache hit")
			}
		}
	})

	b.Run("Get-Miss", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, ok := cache.Get("/nonexistent/project")
			if ok {
				b.Fatal("expected cache miss")
			}
		}
	})
}

// BenchmarkRuleExecution benchmarks individual rule execution.
func BenchmarkRuleExecution(b *testing.B) {
	// Create temporary git repository
	tmpDir, err := os.MkdirTemp("", "prefetch-bench-rule-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo with branches
	if initErr := initGitRepoForBench(tmpDir); initErr != nil {
		b.Fatalf("failed to init git repo: %v", initErr)
	}

	// Create feature branch
	cmd := exec.Command("git", "checkout", "-b", "feature")
	cmd.Dir = tmpDir
	if cmdErr := cmd.Run(); cmdErr != nil {
		b.Fatalf("failed to create branch: %v", cmdErr)
	}

	// Create a commit
	cmd = exec.Command("git", "commit", "--allow-empty", "-m", "Test commit")
	cmd.Dir = tmpDir
	if commitErr := cmd.Run(); commitErr != nil {
		b.Fatalf("failed to create commit: %v", commitErr)
	}

	// Get default branch name (could be main or master)
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = tmpDir
	out, err := cmd.Output()
	if err != nil {
		b.Fatalf("failed to get branch name: %v", err)
	}
	defaultBranch := string(out)[:len(out)-1] // trim newline

	ctx := context.Background()

	b.Run("BranchDiffRule", func(b *testing.B) {
		rule := NewBranchDiffRule(tmpDir, 50, 1*time.Second)
		event := GitEvent{
			Type:        EventTypeBranchSwitch,
			ProjectPath: tmpDir,
			OldBranch:   defaultBranch,
			NewBranch:   "feature",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := rule.Execute(ctx, event)
			if err != nil {
				b.Fatalf("rule execution failed: %v", err)
			}
		}
	})

	b.Run("RecentCommitRule", func(b *testing.B) {
		rule := NewRecentCommitRule(tmpDir, 20, 500*time.Millisecond)

		// Get latest commit hash
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = tmpDir
		out, err := cmd.Output()
		if err != nil {
			b.Fatalf("failed to get commit hash: %v", err)
		}
		commitHash := string(out)[:40]

		event := GitEvent{
			Type:        EventTypeNewCommit,
			ProjectPath: tmpDir,
			CommitHash:  commitHash,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := rule.Execute(ctx, event)
			if err != nil {
				b.Fatalf("rule execution failed: %v", err)
			}
		}
	})
}

// BenchmarkMemoryUsage benchmarks memory usage under load.
func BenchmarkMemoryUsage(b *testing.B) {
	cache := NewCache(5*time.Minute, 1000)

	// Create large results
	largeResults := make([]PreFetchResult, 10)
	for i := range largeResults {
		largeResults[i] = PreFetchResult{
			Type: "test_rule",
			Data: map[string]interface{}{
				"large_data": make([]byte, 1024), // 1KB per result
			},
			Metadata:   map[string]string{"index": string(rune(i))},
			Confidence: 1.0,
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		projectPath := "/test/project/" + string(rune(i%100))
		cache.Set(projectPath, largeResults)
	}
}

// Helper functions for benchmarks

func initGitRepoForBench(dir string) error {
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.name", "Bench User"},
		{"git", "config", "user.email", "bench@example.com"},
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
