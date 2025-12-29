// Package framework provides the integration test harness for contextd.
package framework

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// UserSimulationConfig configures the user simulation test.
type UserSimulationConfig struct {
	NumDevelopers int
	Duration      time.Duration
	ProjectID     string
}

// SimulationStats tracks statistics from the simulation.
type SimulationStats struct {
	TotalOperations     int64
	MemoriesRecorded    int64
	MemoriesSearched    int64
	CheckpointsSaved    int64
	CheckpointsResumed  int64
	FeedbackGiven       int64
	Errors              int64
	AvgOperationTimeMs  float64
}

// TestUserSimulation runs a multi-developer simulation to stress test contextd.
// This test simulates multiple developers working concurrently on a shared project.
func TestUserSimulation(t *testing.T) {
	// Get configuration from environment
	numDevs := getEnvInt("SIM_DEVELOPERS", 3)
	duration := getEnvDuration("SIM_DURATION", 30*time.Second)

	config := UserSimulationConfig{
		NumDevelopers: numDevs,
		Duration:      duration,
		ProjectID:     fmt.Sprintf("sim-%d", time.Now().Unix()),
	}

	t.Logf("Starting user simulation: %d developers for %v", config.NumDevelopers, config.Duration)

	// Create shared store for cross-developer testing
	harness, err := NewTestHarness(config.ProjectID)
	if err != nil {
		t.Fatalf("Failed to create test harness: %v", err)
	}
	defer harness.Cleanup(context.Background())

	// Create developers
	var developers []*Developer
	for i := 0; i < config.NumDevelopers; i++ {
		dev, err := harness.CreateDeveloper(
			fmt.Sprintf("sim-dev-%d", i),
			"sim-tenant",
		)
		if err != nil {
			t.Fatalf("Failed to create developer %d: %v", i, err)
		}
		developers = append(developers, dev)
	}

	// Start all developers
	ctx := context.Background()
	for _, dev := range developers {
		if err := dev.StartContextd(ctx); err != nil {
			t.Fatalf("Failed to start contextd for %s: %v", dev.ID(), err)
		}
	}

	// Run simulation
	stats := runSimulation(t, ctx, developers, config.Duration)

	// Report results
	t.Logf("")
	t.Logf("========================================")
	t.Logf("Simulation Results")
	t.Logf("========================================")
	t.Logf("Duration: %v", config.Duration)
	t.Logf("Developers: %d", config.NumDevelopers)
	t.Logf("Total Operations: %d", stats.TotalOperations)
	t.Logf("  Memories Recorded: %d", stats.MemoriesRecorded)
	t.Logf("  Memories Searched: %d", stats.MemoriesSearched)
	t.Logf("  Checkpoints Saved: %d", stats.CheckpointsSaved)
	t.Logf("  Checkpoints Resumed: %d", stats.CheckpointsResumed)
	t.Logf("  Feedback Given: %d", stats.FeedbackGiven)
	t.Logf("  Errors: %d", stats.Errors)
	t.Logf("Avg Operation Time: %.2fms", stats.AvgOperationTimeMs)
	t.Logf("Operations/sec: %.2f", float64(stats.TotalOperations)/config.Duration.Seconds())
	t.Logf("========================================")

	// Fail if error rate is too high (>5%)
	if stats.TotalOperations > 0 {
		errorRate := float64(stats.Errors) / float64(stats.TotalOperations) * 100
		if errorRate > 5.0 {
			t.Errorf("Error rate too high: %.2f%% (expected < 5%%)", errorRate)
		}
	}
}

func runSimulation(t *testing.T, ctx context.Context, developers []*Developer, duration time.Duration) *SimulationStats {
	stats := &SimulationStats{}
	var wg sync.WaitGroup
	var totalTimeNs int64

	// Create cancellation context for simulation
	simCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	// Start a goroutine for each developer
	for _, dev := range developers {
		wg.Add(1)
		go func(d *Developer) {
			defer wg.Done()
			simulateDeveloper(t, simCtx, d, stats, &totalTimeNs)
		}(dev)
	}

	// Wait for all developers to finish
	wg.Wait()

	// Calculate average operation time
	if stats.TotalOperations > 0 {
		stats.AvgOperationTimeMs = float64(totalTimeNs) / float64(stats.TotalOperations) / 1e6
	}

	return stats
}

func simulateDeveloper(t *testing.T, ctx context.Context, dev *Developer, stats *SimulationStats, totalTimeNs *int64) {
	// Seed random with developer-specific seed for reproducibility
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Memory IDs we've recorded (for feedback)
	var recordedMemoryIDs []string
	// Checkpoint IDs we've saved (for resume)
	var savedCheckpointIDs []string

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Pick a random operation
			op := r.Intn(100)

			start := time.Now()
			var err error

			switch {
			case op < 40: // 40% - Record memory
				memory := generateRandomMemory(r, dev.ID())
				var memID string
				memID, err = dev.RecordMemory(ctx, memory)
				if err == nil && memID != "" {
					recordedMemoryIDs = append(recordedMemoryIDs, memID)
					atomic.AddInt64(&stats.MemoriesRecorded, 1)
				}

			case op < 70: // 30% - Search memory
				query := generateRandomQuery(r)
				_, err = dev.SearchMemory(ctx, query, 5)
				if err == nil {
					atomic.AddInt64(&stats.MemoriesSearched, 1)
				}

			case op < 80: // 10% - Save checkpoint
				req := CheckpointSaveRequest{
					Name:    fmt.Sprintf("checkpoint-%d", r.Int63()),
					Summary: fmt.Sprintf("Simulation checkpoint from %s", dev.ID()),
				}
				var cpID string
				cpID, err = dev.SaveCheckpoint(ctx, req)
				if err == nil && cpID != "" {
					savedCheckpointIDs = append(savedCheckpointIDs, cpID)
					atomic.AddInt64(&stats.CheckpointsSaved, 1)
				}

			case op < 90: // 10% - Resume checkpoint (if any saved)
				if len(savedCheckpointIDs) > 0 {
					cpID := savedCheckpointIDs[r.Intn(len(savedCheckpointIDs))]
					_, err = dev.ResumeCheckpoint(ctx, cpID)
					if err == nil {
						atomic.AddInt64(&stats.CheckpointsResumed, 1)
					}
				}

			default: // 10% - Give feedback (if any memories recorded)
				if len(recordedMemoryIDs) > 0 {
					memID := recordedMemoryIDs[r.Intn(len(recordedMemoryIDs))]
					helpful := r.Intn(2) == 1
					err = dev.GiveFeedback(ctx, memID, helpful, "Simulation feedback")
					if err == nil {
						atomic.AddInt64(&stats.FeedbackGiven, 1)
					}
				}
			}

			elapsed := time.Since(start)
			atomic.AddInt64(totalTimeNs, elapsed.Nanoseconds())
			atomic.AddInt64(&stats.TotalOperations, 1)

			if err != nil {
				// Don't count context cancellation as errors
				if ctx.Err() == nil {
					atomic.AddInt64(&stats.Errors, 1)
					t.Logf("Developer %s operation error: %v", dev.ID(), err)
				}
			}

			// Small delay between operations
			time.Sleep(time.Duration(r.Intn(50)) * time.Millisecond)
		}
	}
}

func generateRandomMemory(r *rand.Rand, devID string) MemoryRecord {
	titles := []string{
		"Fixed null pointer exception",
		"Implemented new feature",
		"Refactored authentication module",
		"Added input validation",
		"Updated API endpoint",
		"Fixed race condition",
		"Improved error handling",
		"Added logging",
		"Optimized database query",
		"Fixed memory leak",
	}

	contents := []string{
		"Found a null pointer when accessing user.profile. Added nil check before accessing nested properties.",
		"Created new REST endpoint for user preferences. Uses JWT authentication.",
		"Moved auth logic to separate package. Now uses interface for testability.",
		"Added validation for email format and password strength requirements.",
		"Changed response format to include pagination metadata.",
		"Added mutex to protect shared map access in concurrent handlers.",
		"Wrapped external API calls with retry logic and circuit breaker.",
		"Added structured logging with correlation IDs for request tracing.",
		"Added index on user_id column, reduced query time from 500ms to 10ms.",
		"Fixed goroutine leak by properly closing channels in worker pool.",
	}

	tags := [][]string{
		{"bugfix", "null-pointer"},
		{"feature", "implementation"},
		{"refactor", "auth"},
		{"validation", "security"},
		{"api", "breaking-change"},
		{"bugfix", "concurrency"},
		{"reliability", "api"},
		{"observability", "logging"},
		{"performance", "database"},
		{"bugfix", "memory"},
	}

	idx := r.Intn(len(titles))
	return MemoryRecord{
		Title:   fmt.Sprintf("[%s] %s", devID, titles[idx]),
		Content: contents[idx],
		Outcome: "success",
		Tags:    tags[idx],
	}
}

func generateRandomQuery(r *rand.Rand) string {
	queries := []string{
		"null pointer exception fix",
		"how to implement authentication",
		"input validation best practices",
		"API pagination",
		"race condition fix",
		"error handling patterns",
		"logging best practices",
		"database optimization",
		"memory leak detection",
		"refactoring strategies",
	}
	return queries[r.Intn(len(queries))]
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
