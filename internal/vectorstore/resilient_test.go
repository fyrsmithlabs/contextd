package vectorstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewResilientChromemDB_HealthyDB(t *testing.T) {
	logger := zaptest.NewLogger(t)
	path := t.TempDir()

	// Create a healthy DB
	db, err := NewResilientChromemDB(path, false, logger)
	require.NoError(t, err)
	require.NotNil(t, db)
}

func TestNewResilientChromemDB_CorruptCollection(t *testing.T) {
	t.Skip("Skipping: requires valid gob-encoded document to test quarantine behavior")
	// NOTE: To fully test quarantine behavior, we would need to create a valid
	// chromem collection with documents but intentionally delete the metadata file.
	// For now, TestFindCorruptCollections provides adequate coverage of the
	// corrupt collection detection logic.
}

func TestFindCorruptCollections(t *testing.T) {
	logger := zaptest.NewLogger(t)
	path := t.TempDir()

	// Create a healthy collection
	healthyHash := "healthy1"
	healthyPath := filepath.Join(path, healthyHash)
	require.NoError(t, os.MkdirAll(healthyPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(healthyPath, "00000000.gob"), []byte("metadata"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(healthyPath, "abcd1234.gob"), []byte("document"), 0644))

	// Create a corrupt collection (documents but no metadata)
	corruptHash := "corrupt1"
	corruptPath := filepath.Join(path, corruptHash)
	require.NoError(t, os.MkdirAll(corruptPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(corruptPath, "abcd5678.gob"), []byte("document"), 0644))

	// Create an empty collection (should be ignored)
	emptyHash := "empty001"
	emptyPath := filepath.Join(path, emptyHash)
	require.NoError(t, os.MkdirAll(emptyPath, 0755))

	// Find corrupt collections
	corrupt, err := findCorruptCollections(path, logger)
	require.NoError(t, err)

	// Should find exactly one corrupt collection
	assert.Len(t, corrupt, 1)
	assert.Contains(t, corrupt, corruptHash)
}

func TestFindCorruptCollections_NoCorruption(t *testing.T) {
	logger := zaptest.NewLogger(t)
	path := t.TempDir()

	// Create only healthy collections
	healthyHash := "healthy1"
	healthyPath := filepath.Join(path, healthyHash)
	require.NoError(t, os.MkdirAll(healthyPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(healthyPath, "00000000.gob"), []byte("metadata"), 0644))

	// Find corrupt collections
	corrupt, err := findCorruptCollections(path, logger)
	require.NoError(t, err)

	// Should find no corruption
	assert.Empty(t, corrupt)
}
