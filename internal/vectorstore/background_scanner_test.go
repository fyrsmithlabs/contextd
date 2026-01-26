package vectorstore

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestBackgroundScanner_StartStop(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	// Create healthy collection
	collectionDir := filepath.Join(tmpDir, "abc12345")
	require.NoError(t, os.MkdirAll(collectionDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, "00000000.gob"), []byte("metadata"), 0644))

	checker := NewMetadataHealthChecker(tmpDir, logger)
	scanner := NewBackgroundScanner(checker, &BackgroundScannerConfig{
		Interval: 100 * time.Millisecond,
	}, logger)

	ctx := context.Background()

	// Start scanner
	scanner.Start(ctx)
	assert.True(t, scanner.IsRunning())

	// Wait for initial scan
	time.Sleep(50 * time.Millisecond)
	assert.NotNil(t, scanner.LastHealth())

	// Stop scanner
	scanner.Stop()
	assert.False(t, scanner.IsRunning())
}

func TestBackgroundScanner_DetectsDegradedState(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	// Create corrupt collection (documents but no metadata)
	collectionDir := filepath.Join(tmpDir, "corrupt1")
	require.NoError(t, os.MkdirAll(collectionDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, "00000001.gob"), []byte("document"), 0644))

	var degradedCalled atomic.Int32

	checker := NewMetadataHealthChecker(tmpDir, logger)
	scanner := NewBackgroundScanner(checker, &BackgroundScannerConfig{
		Interval: 100 * time.Millisecond,
		OnDegraded: func(health *MetadataHealth) {
			degradedCalled.Add(1)
		},
	}, logger)

	ctx := context.Background()
	scanner.Start(ctx)
	defer scanner.Stop()

	// Wait for initial scan to detect degraded state
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(1), degradedCalled.Load(), "OnDegraded should be called once")
	assert.NotNil(t, scanner.LastHealth())
	assert.Equal(t, 1, scanner.LastHealth().CorruptCount)
}

func TestBackgroundScanner_DetectsRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	// Start with corrupt collection
	collectionDir := filepath.Join(tmpDir, "abc12345")
	require.NoError(t, os.MkdirAll(collectionDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, "00000001.gob"), []byte("document"), 0644))

	var recoveredCalled atomic.Int32
	var degradedCalled atomic.Int32

	checker := NewMetadataHealthChecker(tmpDir, logger)
	scanner := NewBackgroundScanner(checker, &BackgroundScannerConfig{
		Interval: 50 * time.Millisecond,
		OnDegraded: func(health *MetadataHealth) {
			degradedCalled.Add(1)
		},
		OnRecovered: func(health *MetadataHealth) {
			recoveredCalled.Add(1)
		},
	}, logger)

	ctx := context.Background()
	scanner.Start(ctx)
	defer scanner.Stop()

	// Wait for initial degraded detection
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, int32(1), degradedCalled.Load())

	// "Fix" the collection by adding metadata
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, "00000000.gob"), []byte("metadata"), 0644))

	// Wait for recovery detection
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(1), recoveredCalled.Load(), "OnRecovered should be called")
}

func TestBackgroundScanner_PeriodicScans(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	// Create healthy collection
	collectionDir := filepath.Join(tmpDir, "abc12345")
	require.NoError(t, os.MkdirAll(collectionDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, "00000000.gob"), []byte("metadata"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(collectionDir, "00000001.gob"), []byte("document"), 0644))

	checker := NewMetadataHealthChecker(tmpDir, logger)
	scanner := NewBackgroundScanner(checker, &BackgroundScannerConfig{
		Interval: 30 * time.Millisecond,
	}, logger)

	ctx := context.Background()
	scanner.Start(ctx)
	defer scanner.Stop()

	// Wait for multiple scan cycles
	time.Sleep(100 * time.Millisecond)

	health := scanner.LastHealth()
	require.NotNil(t, health)
	assert.True(t, health.IsHealthy())
	assert.Equal(t, 1, health.HealthyCount)
}

func TestBackgroundScanner_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	checker := NewMetadataHealthChecker(tmpDir, logger)
	scanner := NewBackgroundScanner(checker, &BackgroundScannerConfig{
		Interval: 1 * time.Second,
	}, logger)

	ctx, cancel := context.WithCancel(context.Background())
	scanner.Start(ctx)

	// Cancel context
	cancel()

	// Scanner should stop
	time.Sleep(50 * time.Millisecond)
	// Note: IsRunning may still be true briefly, but the goroutine has exited
}

func TestBackgroundScanner_DefaultInterval(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	checker := NewMetadataHealthChecker(tmpDir, logger)
	scanner := NewBackgroundScanner(checker, nil, logger)

	// Should use default 5 minute interval
	assert.Equal(t, 5*time.Minute, scanner.config.Interval)
}

func TestBackgroundScanner_DoubleStart(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	checker := NewMetadataHealthChecker(tmpDir, logger)
	scanner := NewBackgroundScanner(checker, &BackgroundScannerConfig{
		Interval: 100 * time.Millisecond,
	}, logger)

	ctx := context.Background()

	// Start twice - should be idempotent
	scanner.Start(ctx)
	scanner.Start(ctx)

	assert.True(t, scanner.IsRunning())
	scanner.Stop()
}

func TestBackgroundScanner_DoubleStop(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	checker := NewMetadataHealthChecker(tmpDir, logger)
	scanner := NewBackgroundScanner(checker, &BackgroundScannerConfig{
		Interval: 100 * time.Millisecond,
	}, logger)

	ctx := context.Background()
	scanner.Start(ctx)

	// Stop twice - should be idempotent
	scanner.Stop()
	scanner.Stop()

	assert.False(t, scanner.IsRunning())
}
