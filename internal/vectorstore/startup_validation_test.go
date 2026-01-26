package vectorstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestValidateStartup_NilChecker(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	result, err := ValidateStartup(ctx, nil, nil, logger)

	require.NoError(t, err)
	assert.True(t, result.Passed)
	assert.Contains(t, result.Messages[0], "validation skipped")
}

func TestValidateStartup_AllHealthy(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	// Create healthy collection
	collectionDir := filepath.Join(tmpDir, "abc12345")
	require.NoError(t, os.MkdirAll(collectionDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, "00000000.gob"), []byte("metadata"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, "00000001.gob"), []byte("document"), 0644))

	checker := NewMetadataHealthChecker(tmpDir, logger)
	result, err := ValidateStartup(ctx, checker, nil, logger)

	require.NoError(t, err)
	assert.True(t, result.Passed)
	assert.Equal(t, 0, result.ErrorCount)
	assert.Equal(t, 0, result.WarningCount)
	assert.NotNil(t, result.Health)
	assert.Equal(t, 1, result.Health.HealthyCount)
}

func TestValidateStartup_CorruptCollection_NoBlock(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	// Create corrupt collection (documents but no metadata)
	collectionDir := filepath.Join(tmpDir, "corrupt1")
	require.NoError(t, os.MkdirAll(collectionDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, "00000001.gob"), []byte("document"), 0644))

	checker := NewMetadataHealthChecker(tmpDir, logger)
	cfg := &StartupValidationConfig{
		FailOnCorruption: false, // Don't block
	}

	result, err := ValidateStartup(ctx, checker, cfg, logger)

	require.NoError(t, err) // No error because FailOnCorruption=false
	assert.True(t, result.Passed)
	assert.Equal(t, 1, result.ErrorCount) // Corruption is logged as error
	assert.Equal(t, 1, result.Health.CorruptCount)
}

func TestValidateStartup_CorruptCollection_Block(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	// Create corrupt collection
	collectionDir := filepath.Join(tmpDir, "corrupt1")
	require.NoError(t, os.MkdirAll(collectionDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, "00000001.gob"), []byte("document"), 0644))

	checker := NewMetadataHealthChecker(tmpDir, logger)
	cfg := &StartupValidationConfig{
		FailOnCorruption: true, // Block on corruption
	}

	result, err := ValidateStartup(ctx, checker, cfg, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "startup blocked")
	assert.False(t, result.Passed)
}

func TestValidateStartup_EmptyCollection_Warning(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	// Create empty collection (just directory, no files)
	collectionDir := filepath.Join(tmpDir, "empty123")
	require.NoError(t, os.MkdirAll(collectionDir, 0755))

	checker := NewMetadataHealthChecker(tmpDir, logger)
	result, err := ValidateStartup(ctx, checker, nil, logger)

	require.NoError(t, err)
	assert.True(t, result.Passed)
	assert.Equal(t, 1, result.WarningCount) // Empty collection is a warning
	assert.Equal(t, 1, len(result.Health.Empty))
}

func TestValidateStartup_FailOnDegraded(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	// Create corrupt collection
	collectionDir := filepath.Join(tmpDir, "corrupt1")
	require.NoError(t, os.MkdirAll(collectionDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, "00000001.gob"), []byte("document"), 0644))

	checker := NewMetadataHealthChecker(tmpDir, logger)
	cfg := &StartupValidationConfig{
		FailOnCorruption: false,
		FailOnDegraded:   true, // Block on any degraded state
	}

	result, err := ValidateStartup(ctx, checker, cfg, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "degraded state")
	assert.False(t, result.Passed)
}

func TestMetrics_UpdateHealthMetrics(t *testing.T) {
	health := &MetadataHealth{
		Healthy:       []string{"a", "b", "c"},
		Corrupt:       []string{"d"},
		Empty:         []string{"e"},
		HealthyCount:  3,
		CorruptCount:  1,
	}

	// Should not panic
	UpdateHealthMetrics(health)
	UpdateHealthMetrics(nil) // Should handle nil gracefully
}

func TestMetrics_RecordResults(t *testing.T) {
	// Should not panic
	RecordHealthCheckResult(true)
	RecordHealthCheckResult(false)
	RecordQuarantineResult(true)
	RecordQuarantineResult(false)
}
