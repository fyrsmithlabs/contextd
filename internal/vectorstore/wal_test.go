package vectorstore

import (
	"context"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWAL_WriteEntry(t *testing.T) {
	logger := zap.NewNop()
	scrubber := &secrets.NoopScrubber{}
	ctx := context.Background()

	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	entry := WALEntry{
		ID:        "test-entry-1",
		Operation: "add",
		Docs: []Document{
			{ID: "doc1", Content: "test content", Metadata: map[string]interface{}{"key": "value"}},
		},
		Timestamp: time.Now(),
		Synced:    false,
	}

	err = wal.WriteEntry(ctx, entry)
	assert.NoError(t, err)

	// Verify entry was written
	pending := wal.PendingEntries()
	assert.Len(t, pending, 1)
	assert.Equal(t, "test-entry-1", pending[0].ID)
}

func TestWAL_WriteEntry_InvalidOperation(t *testing.T) {
	logger := zap.NewNop()
	scrubber := &secrets.NoopScrubber{}
	ctx := context.Background()

	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	entry := WALEntry{
		ID:        "test-entry-2",
		Operation: "invalid",
		Timestamp: time.Now(),
	}

	err = wal.WriteEntry(ctx, entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid WAL operation")
}

func TestWAL_MarkSynced(t *testing.T) {
	logger := zap.NewNop()
	scrubber := &secrets.NoopScrubber{}
	ctx := context.Background()

	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	// Write entry
	entry := WALEntry{
		ID:        "test-entry-3",
		Operation: "add",
		Docs: []Document{
			{ID: "doc2", Content: "test", Metadata: map[string]interface{}{}},
		},
		Timestamp: time.Now(),
		Synced:    false,
	}

	err = wal.WriteEntry(ctx, entry)
	require.NoError(t, err)

	// Mark as synced
	err = wal.MarkSynced("test-entry-3")
	assert.NoError(t, err)

	// Verify no longer pending
	pending := wal.PendingEntries()
	assert.Len(t, pending, 0)
}

func TestWAL_RecordSyncAttempt(t *testing.T) {
	logger := zap.NewNop()
	scrubber := &secrets.NoopScrubber{}
	ctx := context.Background()

	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	// Write entry
	entry := WALEntry{
		ID:        "test-entry-4",
		Operation: "add",
		Docs: []Document{
			{ID: "doc3", Content: "test", Metadata: map[string]interface{}{}},
		},
		Timestamp: time.Now(),
		Synced:    false,
	}

	err = wal.WriteEntry(ctx, entry)
	require.NoError(t, err)

	// Record sync attempt
	testErr := assert.AnError
	err = wal.RecordSyncAttempt("test-entry-4", testErr)
	assert.NoError(t, err)

	// Verify sync attempt recorded
	pending := wal.PendingEntries()
	require.Len(t, pending, 1)
	assert.Equal(t, 1, pending[0].SyncAttempts)
	assert.NotEmpty(t, pending[0].SyncError)
}

func TestWAL_PendingEntries(t *testing.T) {
	logger := zap.NewNop()
	scrubber := &secrets.NoopScrubber{}
	ctx := context.Background()

	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	// Initially no pending entries
	pending := wal.PendingEntries()
	assert.Len(t, pending, 0)

	// Write unsynced entry
	entry := WALEntry{
		ID:        "test-entry-5",
		Operation: "add",
		Docs: []Document{
			{ID: "doc4", Content: "test", Metadata: map[string]interface{}{}},
		},
		Timestamp: time.Now(),
		Synced:    false,
	}

	err = wal.WriteEntry(ctx, entry)
	require.NoError(t, err)

	pending = wal.PendingEntries()
	assert.Len(t, pending, 1)
}

func TestWAL_Compact(t *testing.T) {
	logger := zap.NewNop()
	scrubber := &secrets.NoopScrubber{}
	ctx := context.Background()

	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	// Write synced entry with old timestamp
	oldEntry := WALEntry{
		ID:        "old-entry",
		Operation: "add",
		Docs: []Document{
			{ID: "doc5", Content: "old", Metadata: map[string]interface{}{}},
		},
		Timestamp: time.Now().AddDate(0, 0, -10), // 10 days ago
		Synced:    true,
	}

	err = wal.WriteEntry(ctx, oldEntry)
	require.NoError(t, err)

	err = wal.MarkSynced("old-entry")
	require.NoError(t, err)

	// Write recent synced entry
	recentEntry := WALEntry{
		ID:        "recent-entry",
		Operation: "add",
		Docs: []Document{
			{ID: "doc6", Content: "recent", Metadata: map[string]interface{}{}},
		},
		Timestamp: time.Now(),
		Synced:    true,
	}

	err = wal.WriteEntry(ctx, recentEntry)
	require.NoError(t, err)

	err = wal.MarkSynced("recent-entry")
	require.NoError(t, err)

	// Compact with 7 day retention
	err = wal.Compact(7)
	assert.NoError(t, err)
}

func TestWAL_ValidateEntrySize(t *testing.T) {
	logger := zap.NewNop()
	scrubber := &secrets.NoopScrubber{}
	ctx := context.Background()

	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	// Too many docs
	largeDocs := make([]Document, maxDocsPerEntry+1)
	for i := range largeDocs {
		largeDocs[i] = Document{ID: "doc", Content: "test", Metadata: map[string]interface{}{}}
	}

	entry := WALEntry{
		ID:        "large-entry",
		Operation: "add",
		Docs:      largeDocs,
		Timestamp: time.Now(),
	}

	err = wal.WriteEntry(ctx, entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max documents")
}

func TestWAL_SecretScrubbing(t *testing.T) {
	logger := zap.NewNop()

	// Create real scrubber
	scrubber, err := secrets.New(nil)
	require.NoError(t, err)

	ctx := context.Background()

	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	// Write entry with potential secret
	entry := WALEntry{
		ID:        "secret-entry",
		Operation: "add",
		Docs: []Document{
			{
				ID:      "doc7",
				Content: "This is a test with a fake AWS key: AKIAIOSFODNN7EXAMPLE",
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
		},
		Timestamp: time.Now(),
	}

	err = wal.WriteEntry(ctx, entry)
	assert.NoError(t, err)

	// Verify content was scrubbed
	pending := wal.PendingEntries()
	require.Len(t, pending, 1)
	assert.NotContains(t, pending[0].Docs[0].Content, "AKIAIOSFODNN7EXAMPLE")
}

func TestValidOperations(t *testing.T) {
	assert.True(t, ValidOperations["add"])
	assert.True(t, ValidOperations["delete"])
	assert.False(t, ValidOperations["invalid"])
}
