# Test Suites

**Status**: Active Development
**Last Updated**: 2025-12-11

---

## Suite A: Policy Compliance

Tests that recorded policies influence future developer behavior.

### Test Functions

| Function | File | What It Validates |
|----------|------|-------------------|
| `TestSuiteA_Policy_TDDEnforcement` | suite_a_policy_test.go | TDD policy search with high confidence |
| `TestSuiteA_Policy_ConventionalCommits` | suite_a_policy_test.go | Commit message policy search |
| `TestSuiteA_Policy_NoSecrets` | suite_a_policy_test.go | Security policy search |
| `TestSuiteA_Policy_CrossDeveloperPolicySharing` | suite_a_policy_test.go | Dev B finds Dev A's policies |
| `TestSuiteA_Secrets_AutomaticScrubbing` | suite_a_secrets_test.go | Secrets removed before storage |
| `TestSuiteA_Secrets_DefenseInDepth` | suite_a_secrets_test.go | Scrubbing on both store and search |

### Setup Pattern

```go
func TestSuiteA_Policy_TDDEnforcement(t *testing.T) {
    t.Run("searches TDD policy with confidence >= 0.7", func(t *testing.T) {
        // 1. Create shared store with unique ProjectID
        sharedStore, err := NewSharedStore(SharedStoreConfig{
            ProjectID: "test_project_tdd_policy",  // UNIQUE per test
        })
        require.NoError(t, err)
        defer sharedStore.Close()

        // 2. Create developer with store
        dev, err := NewDeveloperWithStore(DeveloperConfig{
            ID:        "dev-alice",
            TenantID:  "test-tenant",
            ProjectID: "test_project_tdd_policy",
        }, sharedStore)
        require.NoError(t, err)

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        // 3. Start services
        err = dev.StartContextd(ctx)
        require.NoError(t, err)
        defer dev.StopContextd(ctx)

        // 4. Record memory, search, assert
        // ...
    })
}
```

### Anti-Patterns

```go
// BAD: Reusing ProjectID across tests
sharedStore, _ := NewSharedStore(SharedStoreConfig{
    ProjectID: "test_project",  // Will cause cross-contamination
})

// GOOD: Unique ProjectID per test
sharedStore, _ := NewSharedStore(SharedStoreConfig{
    ProjectID: "test_project_tdd_a1",  // Unique
})
```

### Pass Criteria

| Check | Threshold | Type |
|-------|-----------|------|
| Results found | >= 1 | Binary |
| Confidence score | >= 0.7 | Threshold |
| Content matches | Contains policy text | Binary |

---

## Suite C: Bug-Fix Learning

Tests that Developer B benefits from Developer A's recorded fixes.

### Test Functions

| Function | What It Validates |
|----------|-------------------|
| `TestSuiteC_BugFix_SameBugSearch` | Exact bug match returns high-confidence fix |
| `TestSuiteC_BugFix_SimilarBugAdaptation` | Similar bug returns adaptable fix |
| `TestSuiteC_BugFix_FalsePositivePrevention` | Unrelated queries return low confidence |
| `TestSuiteC_BugFix_ConfidenceDecayOnNegativeFeedback` | Negative feedback reduces confidence |
| `TestSuiteC_BugFix_KnowledgeTransferWorkflow` | Senior dev fix found by junior dev |

### Setup Pattern

```go
func TestSuiteC_BugFix_SameBugSearch(t *testing.T) {
    t.Run("finds exact bug fix with high confidence", func(t *testing.T) {
        sharedStore, err := NewSharedStore(SharedStoreConfig{
            ProjectID: "test_project_bugfix_c1",
        })
        require.NoError(t, err)
        defer sharedStore.Close()

        dev, err := NewDeveloperWithStore(DeveloperConfig{
            ID:        "dev-c1",
            TenantID:  "test-tenant",
            ProjectID: "test_project_bugfix_c1",
        }, sharedStore)
        require.NoError(t, err)

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        err = dev.StartContextd(ctx)
        require.NoError(t, err)
        defer dev.StopContextd(ctx)

        // Record bug fix
        memoryID, err := dev.RecordMemory(ctx, MemoryRecord{
            Title:   "nil pointer dereference in user service",
            Content: "Bug: nil pointer when user.Profile accessed...",
            Tags:    []string{"bug", "nil-pointer"},
            Outcome: "success",
        })
        require.NoError(t, err)

        // Search for same bug
        results, err := dev.SearchMemory(ctx, "nil pointer dereference in user service", 5)
        require.NoError(t, err)

        // Assert
        assert.GreaterOrEqual(t, len(results), 1)
        if len(results) > 0 {
            assert.GreaterOrEqual(t, results[0].Confidence, 0.7)
        }
    })
}
```

### Pass Criteria

| Test | Pass Condition |
|------|----------------|
| Same bug | Confidence >= 0.7, fix found |
| Similar bug | Confidence >= 0.5, adaptable pattern found |
| False positive | Confidence < 0.5 OR no results |
| Negative feedback | Confidence decreases after feedback |
| Knowledge transfer | Junior finds senior's fix |

---

## Suite D: Multi-Session Continuity

Tests checkpoint save/resume functionality.

### Test Functions

| Function | What It Validates |
|----------|-------------------|
| `TestSuiteD_MultiSession_CleanResume` | Checkpoint saves and restores context |
| `TestSuiteD_MultiSession_CheckpointListAndSelection` | Multiple checkpoints can be listed and selected |
| `TestSuiteD_MultiSession_PartialWorkResume` | Partial progress preserved in checkpoint |
| `TestSuiteD_MultiSession_CrossSessionMemoryAccumulation` | Memories persist across sessions |
| `TestSuiteD_MultiSession_CheckpointStats` | Stats track checkpoint operations |
| `TestSuiteD_MultiSession_SessionIDPreservation` | Session IDs can be set and found |

### Setup Pattern

```go
func TestSuiteD_MultiSession_CleanResume(t *testing.T) {
    t.Run("checkpoint can be saved and resumed", func(t *testing.T) {
        sharedStore, err := NewSharedStore(SharedStoreConfig{
            ProjectID: "test_project_multisession_d1",
        })
        require.NoError(t, err)
        defer sharedStore.Close()

        // Session 1: Create and save checkpoint
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

        // Save checkpoint
        checkpointID, err := dev1.SaveCheckpoint(ctx, CheckpointSaveRequest{
            Name:    "feature-complete",
            Summary: "Auth feature: User model complete",
            Context: "Working on authentication...",
        })
        require.NoError(t, err)

        dev1.StopContextd(ctx)

        // Session 2: Resume from checkpoint
        dev2, err := NewDeveloperWithStore(DeveloperConfig{
            ID:        "dev-d1-session2",
            TenantID:  "test-tenant-d1",
            ProjectID: "test_project_multisession_d1",
        }, sharedStore)
        require.NoError(t, err)

        err = dev2.StartContextd(ctx)
        require.NoError(t, err)
        defer dev2.StopContextd(ctx)

        resumed, err := dev2.ResumeCheckpoint(ctx, checkpointID)
        require.NoError(t, err)

        assert.Contains(t, resumed.Summary, "Auth feature")
    })
}
```

### Pass Criteria

| Test | Pass Condition |
|------|----------------|
| Clean resume | Summary matches saved content |
| List checkpoints | >= 3 checkpoints returned |
| Partial work | Progress indicators preserved |
| Memory accumulation | Session 1 memories found in session 2 |
| Stats tracking | Checkpoint count increments |

---

## Cross-Developer Scenarios

Both Suite A and Suite C test cross-developer knowledge transfer. The pattern:

```go
// Create shared store for BOTH developers
sharedProject := "test_project_shared"
sharedStore, _ := NewSharedStore(SharedStoreConfig{
    ProjectID: sharedProject,
})

// Developer A records knowledge
devA, _ := NewDeveloperWithStore(DeveloperConfig{
    ID:        "senior-dev",
    ProjectID: sharedProject,
}, sharedStore)

devA.RecordMemory(ctx, MemoryRecord{...})

// Developer B searches and finds it
devB, _ := NewDeveloperWithStore(DeveloperConfig{
    ID:        "junior-dev",
    ProjectID: sharedProject,  // SAME project
}, sharedStore)

results, _ := devB.SearchMemory(ctx, "related query", 5)
// results contains devA's knowledge
```

The shared store and matching ProjectID enable cross-developer visibility.
