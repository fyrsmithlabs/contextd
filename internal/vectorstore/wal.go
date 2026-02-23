// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"go.uber.org/zap"
)

// validOperations whitelist for deserialization safety.
var validOperations = map[string]bool{
	"add":    true,
	"delete": true,
}

// WALEntry represents a single write-ahead log entry.
type WALEntry struct {
	ID           string
	Operation    string     // "add", "delete" - validated against whitelist
	Docs         []Document // For add operations (scrubbed before write)
	IDs          []string   // For delete operations
	Timestamp    time.Time
	Synced       bool
	Checksum     []byte    // HMAC-SHA256 of entry content
	RemoteState  string    // "unknown", "exists", "deleted"
	SyncAttempts int       // Number of sync attempts
	LastAttempt  time.Time // When last sync attempted
	SyncError    string    // Last error message (for debugging)
}

// WAL manages a write-ahead log for pending sync operations.
type WAL struct {
	path     string           // .claude/contextd/wal/
	mu       sync.Mutex       // Protects entries and file operations
	entries  []WALEntry       // In-memory copy of WAL
	hmacKey  []byte           // Cryptographically random key (NOT derived from path)
	scrubber secrets.Scrubber // gitleaks integration
	keyPath  string           // .claude/contextd/wal/.hmac_key (0600 permissions)
	logger   *zap.Logger
}

const (
	maxEntrySize     = 10 * 1024 * 1024 // 10MB per entry
	maxDocsPerEntry  = 10000            // Max documents per entry
	maxOrphansToScan = 10000            // Prevent infinite loops
	hmacKeySize      = 32               // 256-bit HMAC key
)

// NewWAL creates a new Write-Ahead Log.
func NewWAL(path string, scrubber secrets.Scrubber, logger *zap.Logger) (*WAL, error) {
	if scrubber == nil {
		return nil, fmt.Errorf("WAL: scrubber is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("WAL: logger is required")
	}

	// Validate path for directory traversal
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("WAL: path contains directory traversal: %s", path)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(cleanPath, 0700); err != nil {
		return nil, fmt.Errorf("WAL: failed to create directory: %w", err)
	}

	w := &WAL{
		path:     cleanPath,
		entries:  make([]WALEntry, 0),
		scrubber: scrubber,
		logger:   logger,
	}

	// Initialize HMAC key (loads or creates)
	if err := w.initHMACKey(); err != nil {
		return nil, fmt.Errorf("WAL: failed to initialize HMAC key: %w", err)
	}

	// Load existing entries
	if err := w.load(); err != nil {
		return nil, fmt.Errorf("WAL: failed to load entries: %w", err)
	}

	logger.Info("WAL initialized",
		zap.String("path", path),
		zap.Int("entries_loaded", len(w.entries)))

	return w, nil
}

// initHMACKey generates or loads HMAC key with secure file creation.
//
// SECURITY WARNING: This implementation stores HMAC keys on the filesystem
// with file permissions (0600) as the only protection. This is acceptable
// for development and single-user deployments, but production multi-tenant
// systems should consider:
//   - OS keyring integration (macOS Keychain, Windows Credential Manager, Linux Secret Service)
//   - Hardware Security Module (HSM) for enterprise deployments
//   - Cloud KMS (AWS KMS, GCP KMS, Azure Key Vault) for cloud deployments
//
// The key file location (.claude/contextd/wal/.hmac_key) is intentionally
// hidden but predictable - rely on directory permissions and access controls.
func (w *WAL) initHMACKey() error {
	w.keyPath = filepath.Join(w.path, ".hmac_key")

	// Try to load existing key
	if key, err := w.loadKeySecure(); err == nil {
		w.hmacKey = key

		// Validate key file permissions
		if err := w.validateKeyPermissions(); err != nil {
			w.logger.Warn("WAL: HMAC key file has insecure permissions",
				zap.Error(err),
				zap.String("key_path", w.keyPath))
			// Don't fail - just warn (user may have valid reasons)
		}

		return nil
	}

	// Generate new 32-byte (256-bit) random key
	key := make([]byte, hmacKeySize)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate HMAC key: %w", err)
	}

	// Write key with secure permissions from creation (atomic)
	if err := w.writeKeySecure(key); err != nil {
		return err
	}

	w.hmacKey = key
	w.logger.Info("WAL: generated new HMAC key",
		zap.String("key_path", w.keyPath),
		zap.String("permissions", "0600"))

	return nil
}

// loadKeySecure loads the HMAC key from disk.
func (w *WAL) loadKeySecure() ([]byte, error) {
	data, err := os.ReadFile(w.keyPath)
	if err != nil {
		return nil, err
	}

	if len(data) != hmacKeySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", hmacKeySize, len(data))
	}

	return data, nil
}

// validateKeyPermissions checks that the HMAC key file has secure permissions.
func (w *WAL) validateKeyPermissions() error {
	info, err := os.Stat(w.keyPath)
	if err != nil {
		return fmt.Errorf("failed to stat key file: %w", err)
	}

	// Check permissions are 0600 (user read/write only)
	mode := info.Mode().Perm()
	if mode != 0600 {
		return fmt.Errorf("insecure permissions %04o (expected 0600)", mode)
	}

	return nil
}

// writeKeySecure uses atomic write with secure permissions from creation.
func (w *WAL) writeKeySecure(key []byte) error {
	// Create temp file with restricted permissions immediately
	tmpPath := w.keyPath + ".tmp." + randomSuffix()
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}

	if _, err := f.Write(key); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write key: %w", err)
	}

	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync key file: %w", err)
	}
	f.Close()

	// Atomic rename (prevents partial read)
	if err := os.Rename(tmpPath, w.keyPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize key file: %w", err)
	}

	return nil
}

// randomSuffix generates a random suffix for temp files.
func randomSuffix() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// WriteEntry writes a new entry to the WAL with security controls.
func (w *WAL) WriteEntry(ctx context.Context, entry WALEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 1. Validate operation at entry point (defense in depth)
	if !validOperations[entry.Operation] {
		return fmt.Errorf("invalid WAL operation: %s", entry.Operation)
	}

	// 2. Scrub sensitive content with validation
	for i := range entry.Docs {
		result := w.scrubber.Scrub(entry.Docs[i].Content)
		if result == nil {
			return fmt.Errorf("WAL: scrubbing failed for doc %s", entry.Docs[i].ID)
		}
		entry.Docs[i].Content = result.Scrubbed

		// Log if secrets were found and redacted (Debug level to reduce verbosity)
		if result.TotalFindings > 0 {
			w.logger.Debug("WAL: secrets redacted from document",
				zap.String("doc_id", entry.Docs[i].ID),
				zap.Int("secrets_found", result.TotalFindings))
		}

		// Scrub metadata too (only string values)
		for k, v := range entry.Docs[i].Metadata {
			if strVal, ok := v.(string); ok {
				metaResult := w.scrubber.Scrub(strVal)
				if metaResult != nil {
					entry.Docs[i].Metadata[k] = metaResult.Scrubbed
				}
			}
		}
	}

	// 3. Compute checksum using HMAC-SHA256
	entry.Checksum = w.computeHMAC(entry)

	// 4. Validate entry size limits
	if err := w.validateEntrySize(entry); err != nil {
		return err
	}

	// 5. Write with secure atomic pattern (no TOCTOU)
	if err := w.writeEntrySecure(entry); err != nil {
		return err
	}

	// 6. Add to in-memory entries
	w.entries = append(w.entries, entry)

	return nil
}

// computeHMAC computes HMAC-SHA256 for an entry.
func (w *WAL) computeHMAC(entry WALEntry) []byte {
	h := hmac.New(sha256.New, w.hmacKey)

	// Include all relevant fields in HMAC
	h.Write([]byte(entry.ID))
	h.Write([]byte(entry.Operation))
	h.Write([]byte(entry.Timestamp.Format(time.RFC3339Nano)))

	// Include document content for add operations
	if entry.Operation == "add" {
		for _, doc := range entry.Docs {
			h.Write([]byte(doc.ID))
			h.Write([]byte(doc.Content))
		}
	}

	// Include IDs for delete operations
	if entry.Operation == "delete" {
		for _, id := range entry.IDs {
			h.Write([]byte(id))
		}
	}

	return h.Sum(nil)
}

// validateChecksum uses constant-time comparison to prevent timing attacks.
func (w *WAL) validateChecksum(entry WALEntry) bool {
	expected := w.computeHMAC(entry)
	// subtle.ConstantTimeCompare returns 1 if equal, 0 otherwise
	return subtle.ConstantTimeCompare(entry.Checksum, expected) == 1
}

// validateEntrySize prevents DoS via oversized entries.
func (w *WAL) validateEntrySize(entry WALEntry) error {
	if len(entry.Docs) > maxDocsPerEntry {
		return fmt.Errorf("WAL: entry exceeds max documents (%d > %d)", len(entry.Docs), maxDocsPerEntry)
	}

	// Estimate size (actual gob encoding may vary)
	estimatedSize := 0
	for _, doc := range entry.Docs {
		estimatedSize += len(doc.Content) + len(doc.ID)
		for k, v := range doc.Metadata {
			estimatedSize += len(k)
			if strVal, ok := v.(string); ok {
				estimatedSize += len(strVal)
			}
		}
	}
	if estimatedSize > maxEntrySize {
		return fmt.Errorf("WAL: entry exceeds max size (%d > %d bytes)", estimatedSize, maxEntrySize)
	}

	return nil
}

// writeEntrySecure ensures no TOCTOU vulnerability.
func (w *WAL) writeEntrySecure(entry WALEntry) error {
	entryPath := filepath.Join(w.path, entry.ID+".wal")
	tmpPath := entryPath + ".tmp." + randomSuffix()

	// Create with secure permissions from the start (no TOCTOU window)
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("WAL: failed to create entry file: %w", err)
	}

	encoder := gob.NewEncoder(f)
	if err := encoder.Encode(entry); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("WAL: failed to encode entry: %w", err)
	}

	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("WAL: failed to sync entry: %w", err)
	}
	f.Close()

	// Atomic rename - no window where file exists with wrong permissions
	if err := os.Rename(tmpPath, entryPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("WAL: failed to finalize entry: %w", err)
	}

	return nil
}

// validateWALPath validates the WAL path to prevent directory traversal attacks.
func (w *WAL) validateWALPath() error {
	// Clean the path to resolve any ../ or ./ components
	cleanPath := filepath.Clean(w.path)

	// Get absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Ensure path is absolute after cleaning
	if !filepath.IsAbs(absPath) {
		return fmt.Errorf("WAL path must be absolute: %s", absPath)
	}

	// Ensure path doesn't contain suspicious patterns
	if strings.Contains(absPath, "..") {
		return fmt.Errorf("WAL path contains directory traversal: %s", absPath)
	}

	// Update to use cleaned absolute path
	w.path = absPath
	return nil
}

// load reads all WAL entries from disk.
func (w *WAL) load() error {
	// Validate path first to prevent injection
	if err := w.validateWALPath(); err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(w.path, "*.wal"))
	if err != nil {
		return fmt.Errorf("failed to list WAL files: %w", err)
	}

	for _, file := range files {
		entry, err := w.readEntry(file)
		if err != nil {
			w.logger.Warn("WAL: skipping corrupted entry",
				zap.String("file", file),
				zap.Error(err))
			continue
		}

		// Validate checksum
		if !w.validateChecksum(entry) {
			w.logger.Warn("WAL: skipping entry with invalid checksum",
				zap.String("file", file))
			continue
		}

		// Validate operation (defense in depth)
		if !validOperations[entry.Operation] {
			w.logger.Warn("WAL: skipping entry with invalid operation",
				zap.String("file", file),
				zap.String("operation", entry.Operation))
			continue
		}

		w.entries = append(w.entries, entry)
	}

	return nil
}

// readEntry reads a single WAL entry from file.
func (w *WAL) readEntry(path string) (WALEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return WALEntry{}, err
	}
	defer f.Close()

	var entry WALEntry
	decoder := gob.NewDecoder(f)
	if err := decoder.Decode(&entry); err != nil {
		return WALEntry{}, err
	}

	return entry, nil
}

// PendingEntries returns all unsynced entries.
func (w *WAL) PendingEntries() []WALEntry {
	w.mu.Lock()
	defer w.mu.Unlock()

	pending := make([]WALEntry, 0)
	for _, entry := range w.entries {
		if !entry.Synced {
			pending = append(pending, entry)
		}
	}
	return pending
}

// MarkSynced marks an entry as synced.
func (w *WAL) MarkSynced(id string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i := range w.entries {
		if w.entries[i].ID == id {
			w.entries[i].Synced = true
			w.entries[i].SyncError = ""

			// Update on disk
			if err := w.writeEntrySecure(w.entries[i]); err != nil {
				return fmt.Errorf("WAL: failed to update synced entry: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("WAL: entry not found: %s", id)
}

// RecordSyncAttempt records a failed sync attempt.
func (w *WAL) RecordSyncAttempt(id string, err error) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i := range w.entries {
		if w.entries[i].ID == id {
			w.entries[i].SyncAttempts++
			w.entries[i].LastAttempt = time.Now()
			if err != nil {
				w.entries[i].SyncError = err.Error()
			}

			// Update on disk
			if err := w.writeEntrySecure(w.entries[i]); err != nil {
				return fmt.Errorf("WAL: failed to update entry: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("WAL: entry not found: %s", id)
}

// Compact removes synced entries older than retention period.
func (w *WAL) Compact(retentionDays int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	kept := make([]WALEntry, 0)

	for _, entry := range w.entries {
		if !entry.Synced || entry.Timestamp.After(cutoff) {
			kept = append(kept, entry)
		} else {
			// Remove file
			entryPath := filepath.Join(w.path, entry.ID+".wal")
			if err := os.Remove(entryPath); err != nil && !os.IsNotExist(err) {
				w.logger.Warn("WAL: failed to remove compacted entry",
					zap.String("id", entry.ID),
					zap.Error(err))
			}
		}
	}

	originalLen := len(w.entries)
	w.entries = kept
	w.logger.Info("WAL: compaction complete",
		zap.Int("entries_kept", len(kept)),
		zap.Int("entries_removed", originalLen-len(kept)))

	return nil
}

// Close closes the WAL (currently a no-op, but future-proof).
func (w *WAL) Close() error {
	return nil
}
