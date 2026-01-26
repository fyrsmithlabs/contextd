package vectorstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// MetadataHealth represents the health status of collection metadata files.
type MetadataHealth struct {
	Healthy       []string          `json:"healthy"`         // Collections with valid metadata
	Corrupt       []string          `json:"corrupt"`         // Collections missing metadata but have documents
	Empty         []string          `json:"empty"`           // Collections with no documents
	Total         int               `json:"total"`           // Total collections found
	HealthyCount  int               `json:"healthy_count"`   // Count of healthy collections
	CorruptCount  int               `json:"corrupt_count"`   // Count of corrupt collections
	LastCheckTime time.Time         `json:"last_check_time"` // When the check was performed
	CheckDuration time.Duration     `json:"check_duration"`  // How long the check took
	Details       map[string]string `json:"details"`         // Per-collection status details
}

// MetadataHealthChecker provides metadata integrity verification.
type MetadataHealthChecker struct {
	path   string
	logger *zap.Logger
}

// NewMetadataHealthChecker creates a new metadata health checker.
func NewMetadataHealthChecker(path string, logger *zap.Logger) *MetadataHealthChecker {
	return &MetadataHealthChecker{
		path:   path,
		logger: logger,
	}
}

// Check performs a metadata integrity check on all collections.
func (c *MetadataHealthChecker) Check(ctx context.Context) (*MetadataHealth, error) {
	start := time.Now()

	health := &MetadataHealth{
		Healthy:       []string{},
		Corrupt:       []string{},
		Empty:         []string{},
		Details:       make(map[string]string),
		LastCheckTime: start,
	}

	// Read all collection directories
	entries, err := os.ReadDir(c.path)
	if err != nil {
		RecordHealthCheckResult(false)
		return nil, fmt.Errorf("reading vectorstore directory: %w", err)
	}

	for _, entry := range entries {
		// Skip files and hidden directories (like .quarantine)
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		health.Total++
		collectionHash := entry.Name()
		collectionPath := filepath.Join(c.path, collectionHash)
		metadataPath := filepath.Join(collectionPath, "00000000.gob")

		// Check if metadata file exists
		metadataInfo, metadataErr := os.Stat(metadataPath)

		// Count document files (.gob files excluding metadata)
		files, readErr := os.ReadDir(collectionPath)
		if readErr != nil {
			c.logger.Warn("failed to read collection directory",
				zap.String("collection", collectionHash),
				zap.Error(readErr))
			health.Details[collectionHash] = "error: unable to read collection"
			continue
		}

		documentCount := 0
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".gob") && f.Name() != "00000000.gob" {
				documentCount++
			}
		}

		// Classify collection health
		if os.IsNotExist(metadataErr) {
			// Metadata missing
			if documentCount > 0 {
				// Corrupt: has documents but no metadata
				health.Corrupt = append(health.Corrupt, collectionHash)
				health.CorruptCount++
				health.Details[collectionHash] = fmt.Sprintf("corrupt: %d documents, no metadata", documentCount)

				c.logger.Warn("corrupt collection detected",
					zap.String("collection", collectionHash),
					zap.Int("documents", documentCount))
			} else {
				// Empty: no metadata and no documents (newly created or cleared)
				health.Empty = append(health.Empty, collectionHash)
				health.Details[collectionHash] = "empty: no metadata or documents"
			}
		} else if metadataErr != nil {
			// Metadata file exists but stat failed (permission issue, etc.)
			health.Details[collectionHash] = "error: " + metadataErr.Error()
			c.logger.Warn("metadata stat failed",
				zap.String("collection", collectionHash),
				zap.Error(metadataErr))
		} else {
			// Metadata exists and is readable
			health.Healthy = append(health.Healthy, collectionHash)
			health.HealthyCount++
			health.Details[collectionHash] = fmt.Sprintf("healthy: %d documents, metadata size %d bytes",
				documentCount, metadataInfo.Size())
		}
	}

	health.CheckDuration = time.Since(start)

	// Update Prometheus metrics
	UpdateHealthMetrics(health)
	RecordHealthCheckResult(true)

	// Log summary
	c.logger.Info("metadata health check completed",
		zap.Int("total", health.Total),
		zap.Int("healthy", health.HealthyCount),
		zap.Int("corrupt", health.CorruptCount),
		zap.Int("empty", len(health.Empty)),
		zap.Duration("duration", health.CheckDuration))

	if health.CorruptCount > 0 {
		c.logger.Warn("corrupt collections detected",
			zap.Strings("corrupt_hashes", health.Corrupt),
			zap.Int("count", health.CorruptCount))
	}

	return health, nil
}

// IsHealthy returns true if all collections have valid metadata.
func (h *MetadataHealth) IsHealthy() bool {
	return h.CorruptCount == 0
}

// Status returns a simple status string.
func (h *MetadataHealth) Status() string {
	if h.IsHealthy() {
		return "healthy"
	}
	return "degraded"
}
