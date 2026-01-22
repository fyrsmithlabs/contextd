package vectorstore

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	chromem "github.com/philippgille/chromem-go"
	"go.uber.org/zap"
)

var collectionHashPattern = regexp.MustCompile(`^[a-f0-9]{8}$`)

// NewResilientChromemDB creates a chromem DB with graceful degradation for corrupt collections.
// If a collection is missing its metadata file, it will be quarantined and the DB will load successfully.
func NewResilientChromemDB(path string, compress bool, logger *zap.Logger) (*chromem.DB, error) {
	// Try normal load
	db, err := chromem.NewPersistentDB(path, compress)
	if err == nil {
		logger.Info("ChromemDB loaded successfully", zap.String("path", path))
		return db, nil
	}

	// Check if error is due to missing metadata
	if !strings.Contains(err.Error(), "collection metadata file not found") {
		return nil, err // Different error, fail normally
	}

	// Find and quarantine corrupt collections
	corruptCollections, findErr := findCorruptCollections(path, logger)
	if findErr != nil {
		logger.Error("Failed to find corrupt collections", zap.Error(findErr))
		return nil, err // Return original error
	}

	if len(corruptCollections) == 0 {
		return nil, err // No corrupt collections found, return original error
	}

	// Quarantine corrupt collections
	quarantinePath := filepath.Join(path, ".quarantine")
	if err := os.MkdirAll(quarantinePath, 0755); err != nil {
		logger.Error("Failed to create quarantine directory", zap.Error(err))
		return nil, err
	}

	for _, hash := range corruptCollections {
		// Validate hash format to prevent path traversal
		if !isValidCollectionHash(hash) {
			logger.Error("Invalid collection hash format, skipping",
				zap.String("hash", hash))
			continue
		}

		src := filepath.Join(path, hash)
		dst := filepath.Join(quarantinePath, hash)

		logger.Warn("Quarantining corrupt collection",
			zap.String("collection_hash", hash),
			zap.String("from", src),
			zap.String("to", dst))

		if err := os.Rename(src, dst); err != nil {
			logger.Error("Failed to quarantine collection",
				zap.String("collection", hash),
				zap.Error(err))
			continue
		}
	}

	// Retry DB load
	db, err = chromem.NewPersistentDB(path, compress)
	if err != nil {
		logger.Error("Failed to load DB even after quarantine", zap.Error(err))
		return nil, err
	}

	logger.Info("ChromemDB loaded successfully after quarantine",
		zap.Int("quarantined_count", len(corruptCollections)))

	return db, nil
}

// findCorruptCollections identifies collections with documents but no metadata file.
func findCorruptCollections(path string, logger *zap.Logger) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var corrupt []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue // Skip files and hidden directories
		}

		collectionPath := filepath.Join(path, entry.Name())
		metadataPath := filepath.Join(collectionPath, "00000000.gob")

		// Check if metadata exists
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			// Check if collection has any .gob files (documents)
			files, readErr := os.ReadDir(collectionPath)
			if readErr != nil {
				logger.Warn("Failed to read collection directory while checking for corruption",
					zap.String("collection_hash", entry.Name()),
					zap.Error(readErr))
				continue
			}
			hasDocuments := false
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".gob") {
					hasDocuments = true
					break
				}
			}

			if hasDocuments {
				logger.Warn("Found corrupt collection (missing metadata but has documents)",
					zap.String("collection_hash", entry.Name()),
					zap.String("path", collectionPath))
				corrupt = append(corrupt, entry.Name())
			}
		}
	}

	return corrupt, nil
}

// isValidCollectionHash validates that a collection hash is safe for filesystem operations.
// Collection hashes are 8-character lowercase hex strings (SHA256 prefix).
func isValidCollectionHash(hash string) bool {
	return collectionHashPattern.MatchString(hash)
}
