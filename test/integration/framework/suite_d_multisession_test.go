// Package framework provides the integration test framework for contextd.
package framework

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Suite D: Multi-Session Tests
//
// Tests the system's ability to preserve context across sessions using
// checkpoints. Validates that work can be resumed without losing progress.
//
// Test Coverage:
//
// D.1: Clean Resume
//   - Developer saves checkpoint mid-work
//   - New session resumes from checkpoint
//   - Verifies context is preserved
//
// D.2: Checkpoint List and Selection
//   - Developer saves multiple checkpoints
//   - Can list and select specific checkpoint
//   - Verifies correct checkpoint is resumed
//
// D.3: Partial Work Resume
//   - Developer records work progress in checkpoint summary
//   - Resume provides accurate summary of completed tasks
//   - Verifies summary matches what was saved
//
// D.4: Cross-Session Memory Accumulation
//   - Session 1 records memories and checkpoint
//   - Session 2 resumes and can still access memories
//   - Verifies memories persist across sessions

// TestSuiteD_MultiSession_CleanResume tests that a checkpoint can be saved
// and resumed cleanly in a new session.
//
// Test D.1: Clean Resume
func TestSuiteD_MultiSession_CleanResume(t *testing.T) {
	t.Run("checkpoint can be saved and resumed", func(t *testing.T) {
		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "test_project_multisession_d1",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		// Session 1: Developer starts work and saves checkpoint
		dev1, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-d1",
			TenantID:  "test-tenant-d1",
			ProjectID: "test_project_multisession_d1",
		}, sharedStore)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = dev1.StartContextd(ctx)
		require.NoError(t, err)

		// Record some work context
		_, err = dev1.RecordMemory(ctx, MemoryRecord{
			Title:   "Auth feature implementation started",
			Content: "Created User model in pkg/auth/models.go with fields: ID, Email, PasswordHash, CreatedAt",
			Tags:    []string{"feature", "auth", "implementation"},
			Outcome: "success",
		})
		require.NoError(t, err)

		// Save checkpoint
		checkpointID, err := dev1.SaveCheckpoint(ctx, CheckpointSaveRequest{
			Name:    "auth-feature-models-complete",
			Summary: "Auth feature: User model complete. Next: implement handlers.",
			Context: "Working on authentication feature. User model done, need to implement login/logout handlers.",
		})
		require.NoError(t, err)
		require.NotEmpty(t, checkpointID)

		// Stop session 1
		err = dev1.StopContextd(ctx)
		require.NoError(t, err)

		// Session 2: New developer instance resumes from checkpoint
		dev2, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-d1-session2",
			TenantID:  "test-tenant-d1",
			ProjectID: "test_project_multisession_d1",
		}, sharedStore)
		require.NoError(t, err)

		err = dev2.StartContextd(ctx)
		require.NoError(t, err)
		defer dev2.StopContextd(ctx)

		// Resume from checkpoint
		resumed, err := dev2.ResumeCheckpoint(ctx, checkpointID)
		require.NoError(t, err)

		// Verify checkpoint context is preserved
		assert.Equal(t, checkpointID, resumed.ID, "checkpoint ID should match")
		assert.Contains(t, resumed.Summary, "Auth feature", "summary should contain feature name")
		assert.Contains(t, resumed.Summary, "User model complete", "summary should indicate models are complete")
		assert.Contains(t, resumed.Summary, "handlers", "summary should mention next task")

		// Binary assertion: checkpoint found and resumed
		assert.NotEmpty(t, resumed.Context, "context should not be empty")

		// Threshold assertion: context contains useful information
		assert.Greater(t, len(resumed.Context), 50, "context should have meaningful content")
	})
}

// TestSuiteD_MultiSession_CheckpointListAndSelection tests that multiple
// checkpoints can be saved and the correct one can be selected for resume.
//
// Test D.2: Checkpoint List and Selection
func TestSuiteD_MultiSession_CheckpointListAndSelection(t *testing.T) {
	t.Run("can list and select specific checkpoint", func(t *testing.T) {
		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "test_project_multisession_d2",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-d2",
			TenantID:  "test-tenant-d2",
			ProjectID: "test_project_multisession_d2",
		}, sharedStore)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Save multiple checkpoints
		cp1ID, err := dev.SaveCheckpoint(ctx, CheckpointSaveRequest{
			Name:    "checkpoint-1",
			Summary: "First checkpoint: initial setup",
			Context: "Setting up project structure",
		})
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)

		cp2ID, err := dev.SaveCheckpoint(ctx, CheckpointSaveRequest{
			Name:    "checkpoint-2",
			Summary: "Second checkpoint: core logic implemented",
			Context: "Core business logic complete",
		})
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		_, err = dev.SaveCheckpoint(ctx, CheckpointSaveRequest{
			Name:    "checkpoint-3",
			Summary: "Third checkpoint: tests added",
			Context: "Unit tests written for core logic",
		})
		require.NoError(t, err)

		// List checkpoints
		checkpoints, err := dev.ListCheckpoints(ctx, 10)
		require.NoError(t, err)

		// Binary assertion: multiple checkpoints found
		assert.GreaterOrEqual(t, len(checkpoints), 3, "should have at least 3 checkpoints")

		// Resume from specific checkpoint (the second one)
		resumed, err := dev.ResumeCheckpoint(ctx, cp2ID)
		require.NoError(t, err)

		// Verify correct checkpoint is resumed
		assert.Equal(t, cp2ID, resumed.ID, "should resume from checkpoint 2")
		assert.Contains(t, resumed.Summary, "core logic", "should have checkpoint 2 summary")

		// Verify we can also resume from checkpoint 1
		resumed1, err := dev.ResumeCheckpoint(ctx, cp1ID)
		require.NoError(t, err)
		assert.Contains(t, resumed1.Summary, "initial setup", "should have checkpoint 1 summary")
	})
}

// TestSuiteD_MultiSession_PartialWorkResume tests that partial work progress
// is accurately captured and can be resumed.
//
// Test D.3: Partial Work Resume
func TestSuiteD_MultiSession_PartialWorkResume(t *testing.T) {
	t.Run("partial work progress is preserved in checkpoint", func(t *testing.T) {
		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "test_project_multisession_d3",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-d3",
			TenantID:  "test-tenant-d3",
			ProjectID: "test_project_multisession_d3",
		}, sharedStore)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Simulate partial work with detailed task tracking
		workSummary := `User Registration Feature Progress:
Tasks Completed:
1. [DONE] Create User model
2. [DONE] Create UserRepository interface
3. [IN PROGRESS] Implement PostgresUserRepository (50%)

Remaining Tasks:
4. [ ] Create RegisterUserHandler
5. [ ] Add input validation

Notes:
- User model has ID, Email, PasswordHash, CreatedAt, UpdatedAt fields
- Repository interface includes Create, GetByID, GetByEmail, Update, Delete methods
- Postgres implementation started, need to complete Create and GetByEmail`

		checkpointID, err := dev.SaveCheckpoint(ctx, CheckpointSaveRequest{
			Name:    "registration-partial",
			Summary: "Registration feature: 2/5 tasks complete, 1 in progress",
			Context: workSummary,
		})
		require.NoError(t, err)

		// Resume and verify partial work is preserved
		resumed, err := dev.ResumeCheckpoint(ctx, checkpointID)
		require.NoError(t, err)

		// Binary assertions: key progress indicators present
		assert.Contains(t, resumed.Context, "[DONE] Create User model", "should show completed task 1")
		assert.Contains(t, resumed.Context, "[DONE] Create UserRepository interface", "should show completed task 2")
		assert.Contains(t, resumed.Context, "[IN PROGRESS]", "should show in-progress task")
		assert.Contains(t, resumed.Context, "Remaining Tasks", "should list remaining tasks")

		// Behavioral assertion: summary accurately reflects progress
		assert.Contains(t, resumed.Summary, "2/5", "summary should indicate 2 of 5 tasks complete")
	})
}

// TestSuiteD_MultiSession_CrossSessionMemoryAccumulation tests that memories
// recorded in one session are accessible after resuming in a new session.
//
// Test D.4: Cross-Session Memory Accumulation
func TestSuiteD_MultiSession_CrossSessionMemoryAccumulation(t *testing.T) {
	t.Run("memories persist across sessions", func(t *testing.T) {
		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "test_project_multisession_d4",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		// Session 1: Record memories and save checkpoint
		dev1, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-d4",
			TenantID:  "test-tenant-d4",
			ProjectID: "test_project_multisession_d4",
		}, sharedStore)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = dev1.StartContextd(ctx)
		require.NoError(t, err)

		// Record a memory about database patterns
		_, err = dev1.RecordMemory(ctx, MemoryRecord{
			Title:   "Database connection pattern",
			Content: "Use connection pooling with max 10 connections, always defer Close(), wrap errors with context",
			Tags:    []string{"pattern", "database", "best-practice"},
			Outcome: "success",
		})
		require.NoError(t, err)

		// Save checkpoint
		checkpointID, err := dev1.SaveCheckpoint(ctx, CheckpointSaveRequest{
			Name:    "db-setup-complete",
			Summary: "Database setup complete with connection pooling",
			Context: "Set up database connection with pooling, documented patterns",
		})
		require.NoError(t, err)

		// Stop session 1
		err = dev1.StopContextd(ctx)
		require.NoError(t, err)

		// Session 2: Resume and verify memories are accessible
		dev2, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-d4-session2",
			TenantID:  "test-tenant-d4",
			ProjectID: "test_project_multisession_d4",
		}, sharedStore)
		require.NoError(t, err)

		err = dev2.StartContextd(ctx)
		require.NoError(t, err)
		defer dev2.StopContextd(ctx)

		// Resume checkpoint
		_, err = dev2.ResumeCheckpoint(ctx, checkpointID)
		require.NoError(t, err)

		// Search for the memory recorded in session 1
		results, err := dev2.SearchMemory(ctx, "database connection pooling pattern", 5)
		require.NoError(t, err)

		// Binary assertion: memory from session 1 is found
		assert.GreaterOrEqual(t, len(results), 1, "should find memory from previous session")

		if len(results) > 0 {
			// Verify it's the right memory
			assert.Contains(t, results[0].Content, "connection pooling",
				"found memory should contain connection pooling info")
			assert.Contains(t, results[0].Content, "defer Close()",
				"found memory should contain defer Close pattern")

			// Threshold assertion: confidence >= 0.7
			assert.GreaterOrEqual(t, results[0].Confidence, 0.7,
				"memory should have high confidence")
		}
	})
}

// TestSuiteD_MultiSession_CheckpointStats tests that checkpoint operations
// are tracked in session statistics.
//
// Test D.5: Checkpoint Statistics
func TestSuiteD_MultiSession_CheckpointStats(t *testing.T) {
	t.Run("checkpoint operations are tracked in stats", func(t *testing.T) {
		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "test_project_multisession_d5",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-d5",
			TenantID:  "test-tenant-d5",
			ProjectID: "test_project_multisession_d5",
		}, sharedStore)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Initial stats
		initialStats := dev.SessionStats()
		assert.Equal(t, 0, initialStats.Checkpoints, "initial checkpoint count should be 0")

		// Save a checkpoint
		_, err = dev.SaveCheckpoint(ctx, CheckpointSaveRequest{
			Name:    "test-checkpoint",
			Summary: "Test checkpoint for stats",
			Context: "Testing checkpoint stats",
		})
		require.NoError(t, err)

		// Check stats after checkpoint
		afterStats := dev.SessionStats()
		assert.Equal(t, 1, afterStats.Checkpoints, "checkpoint count should be 1")
		assert.Greater(t, afterStats.TotalToolCalls, initialStats.TotalToolCalls,
			"total tool calls should increase")

		// Save another checkpoint
		_, err = dev.SaveCheckpoint(ctx, CheckpointSaveRequest{
			Name:    "test-checkpoint-2",
			Summary: "Second test checkpoint",
			Context: "Testing checkpoint stats again",
		})
		require.NoError(t, err)

		finalStats := dev.SessionStats()
		assert.Equal(t, 2, finalStats.Checkpoints, "checkpoint count should be 2")
	})
}

// TestSuiteD_MultiSession_SessionIDPreservation tests that session IDs
// can be set and retrieved correctly.
//
// Test D.6: Session ID Management
func TestSuiteD_MultiSession_SessionIDPreservation(t *testing.T) {
	t.Run("session ID can be set and retrieved", func(t *testing.T) {
		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "test_project_multisession_d6",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		dev, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-d6",
			TenantID:  "test-tenant-d6",
			ProjectID: "test_project_multisession_d6",
		}, sharedStore)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = dev.StartContextd(ctx)
		require.NoError(t, err)
		defer dev.StopContextd(ctx)

		// Session ID should be auto-generated on start
		sessionID := dev.SessionID()
		assert.NotEmpty(t, sessionID, "session ID should be generated on start")
		assert.Contains(t, sessionID, "session_", "session ID should have expected prefix")

		// Should be able to set a custom session ID
		customSessionID := "custom_session_12345"
		dev.SetSessionID(customSessionID)

		retrievedID := dev.SessionID()
		assert.Equal(t, customSessionID, retrievedID, "should be able to set custom session ID")
	})
}
