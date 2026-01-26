// Package main implements metadata recovery commands for the ctxd CLI.
package main

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	// vectorstorePath is the path to the vectorstore directory
	vectorstorePath string
)

func init() {
	// Add metadata command group to root
	rootCmd.AddCommand(metadataCmd)

	// Add subcommands
	metadataCmd.AddCommand(metadataHealthCmd)
	metadataCmd.AddCommand(metadataListCmd)
	metadataCmd.AddCommand(metadataRecoverCmd)
	metadataCmd.AddCommand(quarantineCmd)

	// Add quarantine subcommands
	quarantineCmd.AddCommand(quarantineListCmd)
	quarantineCmd.AddCommand(quarantineRestoreCmd)

	// Global flags for metadata commands
	metadataCmd.PersistentFlags().StringVar(&vectorstorePath, "path", "", "vectorstore path (default: ~/.config/contextd/vectorstore)")
}

// metadataCmd is the parent command for metadata operations
var metadataCmd = &cobra.Command{
	Use:   "metadata",
	Short: "Metadata recovery and health operations",
	Long: `Manage vectorstore metadata files for recovery and diagnostics.

Commands for checking health, listing collections, recovering metadata,
and managing quarantined collections.`,
}

// metadataHealthCmd checks metadata health via HTTP or local filesystem
var metadataHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check vectorstore metadata health",
	Long: `Check the health status of vectorstore metadata files.

Attempts to use the HTTP server first, falls back to local filesystem scan.

Examples:
  # Check health via HTTP server
  ctxd metadata health

  # Check health directly on filesystem
  ctxd metadata health --path ~/.config/contextd/vectorstore`,
	RunE: runMetadataHealth,
}

// metadataListCmd lists all collections
var metadataListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all collections with status",
	Long: `List all vectorstore collections with their health status.

Shows collection hash, name (if known), document count, and status.

Examples:
  ctxd metadata list
  ctxd metadata list --path /custom/vectorstore`,
	RunE: runMetadataList,
}

// metadataRecoverCmd recovers metadata for a collection
var metadataRecoverCmd = &cobra.Command{
	Use:   "recover <collection-name>",
	Short: "Recover metadata for a collection",
	Long: `Recover a missing metadata file for a collection.

Creates a new metadata file (00000000.gob) with the collection name.
Use this after identifying a corrupt collection that needs recovery.

Examples:
  # Recover the contextd_memories collection
  ctxd metadata recover contextd_memories

  # Recover with custom path
  ctxd metadata recover contextd_memories --path /custom/vectorstore`,
	Args: cobra.ExactArgs(1),
	RunE: runMetadataRecover,
}

// quarantineCmd is the parent command for quarantine operations
var quarantineCmd = &cobra.Command{
	Use:   "quarantine",
	Short: "Manage quarantined collections",
	Long:  `View and restore collections that were quarantined due to corruption.`,
}

// quarantineListCmd lists quarantined collections
var quarantineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List quarantined collections",
	Long: `List all collections in the quarantine directory.

Quarantined collections have documents but missing metadata.
They were moved by the resilient DB wrapper during startup.

Examples:
  ctxd metadata quarantine list`,
	RunE: runQuarantineList,
}

// quarantineRestoreCmd restores a quarantined collection
var quarantineRestoreCmd = &cobra.Command{
	Use:   "restore <collection-hash>",
	Short: "Restore a quarantined collection",
	Long: `Restore a quarantined collection after recovering its metadata.

1. First recover the metadata: ctxd metadata recover <name>
2. Then restore from quarantine: ctxd metadata quarantine restore <hash>

Examples:
  ctxd metadata quarantine restore e9f85bf6`,
	Args: cobra.ExactArgs(1),
	RunE: runQuarantineRestore,
}

// MetadataHealthResponse matches the HTTP endpoint response
type MetadataHealthResponse struct {
	Status        string            `json:"status"`
	Healthy       []string          `json:"healthy"`
	Corrupt       []string          `json:"corrupt"`
	Empty         []string          `json:"empty"`
	Total         int               `json:"total"`
	HealthyCount  int               `json:"healthy_count"`
	CorruptCount  int               `json:"corrupt_count"`
	LastCheckTime time.Time         `json:"last_check_time"`
	CheckDuration string            `json:"check_duration"`
	Details       map[string]string `json:"details"`
}

func getVectorstorePath() (string, error) {
	if vectorstorePath != "" {
		return vectorstorePath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".config", "contextd", "vectorstore"), nil
}

func runMetadataHealth(cmd *cobra.Command, args []string) error {
	// Try HTTP first if no path specified
	if vectorstorePath == "" {
		health, err := fetchHealthHTTP()
		if err == nil {
			printHealthReport(health)
			return nil
		}
		fmt.Fprintf(os.Stderr, "HTTP server unavailable, scanning filesystem...\n\n")
	}

	// Fall back to filesystem scan
	path, err := getVectorstorePath()
	if err != nil {
		return err
	}

	health, err := scanFilesystemHealth(path)
	if err != nil {
		return fmt.Errorf("failed to scan filesystem: %w", err)
	}

	printHealthReport(health)
	return nil
}

func fetchHealthHTTP() (*MetadataHealthResponse, error) {
	url := fmt.Sprintf("%s/api/v1/health/metadata", serverURL)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, body)
	}

	var health MetadataHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, err
	}

	return &health, nil
}

func scanFilesystemHealth(path string) (*MetadataHealthResponse, error) {
	health := &MetadataHealthResponse{
		Details:       make(map[string]string),
		LastCheckTime: time.Now(),
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		health.Total++
		hash := entry.Name()
		collectionPath := filepath.Join(path, hash)
		metadataPath := filepath.Join(collectionPath, "00000000.gob")

		// Check metadata existence
		_, metaErr := os.Stat(metadataPath)
		hasMetadata := metaErr == nil

		// Count documents
		files, _ := os.ReadDir(collectionPath)
		docCount := 0
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".gob") && f.Name() != "00000000.gob" {
				docCount++
			}
		}

		if hasMetadata {
			health.Healthy = append(health.Healthy, hash)
			health.HealthyCount++
			health.Details[hash] = fmt.Sprintf("healthy: %d documents", docCount)
		} else if docCount > 0 {
			health.Corrupt = append(health.Corrupt, hash)
			health.CorruptCount++
			health.Details[hash] = fmt.Sprintf("CORRUPT: %d documents, missing metadata", docCount)
		} else {
			health.Empty = append(health.Empty, hash)
			health.Details[hash] = "empty: no documents"
		}
	}

	if health.CorruptCount > 0 {
		health.Status = "degraded"
	} else {
		health.Status = "healthy"
	}

	return health, nil
}

func printHealthReport(health *MetadataHealthResponse) {
	fmt.Printf("Vectorstore Health Report\n")
	fmt.Printf("========================\n\n")

	statusIcon := "✅"
	if health.Status == "degraded" {
		statusIcon = "⚠️"
	}

	fmt.Printf("Status: %s %s\n", statusIcon, health.Status)
	fmt.Printf("Total Collections: %d\n", health.Total)
	fmt.Printf("  Healthy: %d\n", health.HealthyCount)
	fmt.Printf("  Corrupt: %d\n", health.CorruptCount)
	fmt.Printf("  Empty:   %d\n\n", len(health.Empty))

	if health.CorruptCount > 0 {
		fmt.Printf("⚠️  Corrupt Collections (require recovery):\n")
		for _, hash := range health.Corrupt {
			fmt.Printf("  - %s: %s\n", hash, health.Details[hash])
		}
		fmt.Printf("\nTo recover, run:\n")
		fmt.Printf("  ctxd metadata recover <collection-name>\n\n")
	}
}

func runMetadataList(cmd *cobra.Command, args []string) error {
	path, err := getVectorstorePath()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read vectorstore directory: %w", err)
	}

	fmt.Printf("Collections in %s\n", path)
	fmt.Printf("%-10s %-30s %-8s %s\n", "HASH", "NAME", "DOCS", "STATUS")
	fmt.Printf("%s\n", strings.Repeat("-", 70))

	// Known collection names for reverse lookup
	knownCollections := []string{
		"contextd_memories",
		"contextd_checkpoints",
		"contextd_remediations",
		"contextd_repository",
	}

	hashToName := make(map[string]string)
	for _, name := range knownCollections {
		h := sha256.Sum256([]byte(name))
		hashToName[fmt.Sprintf("%x", h)[:8]] = name
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		hash := entry.Name()
		collectionPath := filepath.Join(path, hash)
		metadataPath := filepath.Join(collectionPath, "00000000.gob")

		// Try to get name from metadata or known list
		name := hashToName[hash]
		if name == "" {
			name = tryReadCollectionName(metadataPath)
		}
		if name == "" {
			name = "(unknown)"
		}

		// Count documents
		files, _ := os.ReadDir(collectionPath)
		docCount := 0
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".gob") && f.Name() != "00000000.gob" {
				docCount++
			}
		}

		// Check status
		status := "✅ healthy"
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			if docCount > 0 {
				status = "❌ CORRUPT"
			} else {
				status = "⚪ empty"
			}
		}

		fmt.Printf("%-10s %-30s %-8d %s\n", hash, name, docCount, status)
	}

	return nil
}

func tryReadCollectionName(metadataPath string) string {
	f, err := os.Open(metadataPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	// Decode the gob to get the name
	type persistedCollection struct {
		Name     string
		Metadata map[string]string
	}

	var pc persistedCollection
	if err := gob.NewDecoder(f).Decode(&pc); err != nil {
		return ""
	}

	return pc.Name
}

func runMetadataRecover(cmd *cobra.Command, args []string) error {
	collectionName := args[0]

	path, err := getVectorstorePath()
	if err != nil {
		return err
	}

	// Calculate hash
	h := sha256.Sum256([]byte(collectionName))
	hash := fmt.Sprintf("%x", h)[:8]

	collectionPath := filepath.Join(path, hash)
	metadataPath := filepath.Join(collectionPath, "00000000.gob")

	// Check if collection directory exists
	if _, err := os.Stat(collectionPath); os.IsNotExist(err) {
		// Also check quarantine
		quarantinePath := filepath.Join(path, ".quarantine", hash)
		if _, err := os.Stat(quarantinePath); err == nil {
			fmt.Printf("Collection %s (hash: %s) is in quarantine.\n", collectionName, hash)
			fmt.Printf("Run: ctxd metadata quarantine restore %s\n", hash)
			return nil
		}
		return fmt.Errorf("collection directory not found: %s (hash: %s)", collectionName, hash)
	}

	// Check if metadata already exists
	if _, err := os.Stat(metadataPath); err == nil {
		fmt.Printf("Metadata file already exists for %s\n", collectionName)
		fmt.Printf("Path: %s\n", metadataPath)
		return nil
	}

	// Create metadata structure
	type persistedCollection struct {
		Name     string
		Metadata map[string]string
	}

	pc := persistedCollection{
		Name:     collectionName,
		Metadata: map[string]string{},
	}

	// Create file
	f, err := os.Create(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer f.Close()

	// Encode
	if err := gob.NewEncoder(f).Encode(pc); err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	// Sync
	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to sync metadata file: %w", err)
	}

	fmt.Printf("✅ Successfully recovered metadata for %s\n", collectionName)
	fmt.Printf("   Hash: %s\n", hash)
	fmt.Printf("   Path: %s\n", metadataPath)
	fmt.Printf("\nRestart contextd to load the recovered collection.\n")

	return nil
}

func runQuarantineList(cmd *cobra.Command, args []string) error {
	path, err := getVectorstorePath()
	if err != nil {
		return err
	}

	quarantinePath := filepath.Join(path, ".quarantine")

	if _, err := os.Stat(quarantinePath); os.IsNotExist(err) {
		fmt.Println("No quarantine directory found. No collections have been quarantined.")
		return nil
	}

	entries, err := os.ReadDir(quarantinePath)
	if err != nil {
		return fmt.Errorf("failed to read quarantine directory: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("Quarantine is empty. No collections have been quarantined.")
		return nil
	}

	fmt.Printf("Quarantined Collections in %s\n", quarantinePath)
	fmt.Printf("%-10s %-8s %s\n", "HASH", "DOCS", "PATH")
	fmt.Printf("%s\n", strings.Repeat("-", 60))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		hash := entry.Name()
		collectionPath := filepath.Join(quarantinePath, hash)

		// Count documents
		files, _ := os.ReadDir(collectionPath)
		docCount := 0
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".gob") {
				docCount++
			}
		}

		fmt.Printf("%-10s %-8d %s\n", hash, docCount, collectionPath)
	}

	fmt.Printf("\nTo restore a collection:\n")
	fmt.Printf("  1. Identify the collection name for the hash\n")
	fmt.Printf("  2. Recover metadata: ctxd metadata recover <name>\n")
	fmt.Printf("  3. Restore: ctxd metadata quarantine restore <hash>\n")

	return nil
}

func runQuarantineRestore(cmd *cobra.Command, args []string) error {
	hash := args[0]

	// Validate hash format
	if len(hash) != 8 {
		return fmt.Errorf("invalid hash format: expected 8-character hex string")
	}

	path, err := getVectorstorePath()
	if err != nil {
		return err
	}

	quarantinePath := filepath.Join(path, ".quarantine", hash)
	targetPath := filepath.Join(path, hash)

	// Check quarantine exists
	if _, err := os.Stat(quarantinePath); os.IsNotExist(err) {
		return fmt.Errorf("collection %s not found in quarantine", hash)
	}

	// Check target doesn't already exist
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("collection %s already exists in vectorstore - cannot restore", hash)
	}

	// Check metadata exists in quarantine (should have been recovered)
	metadataPath := filepath.Join(quarantinePath, "00000000.gob")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return fmt.Errorf("metadata file not found in quarantined collection\nRecover metadata first: ctxd metadata recover <collection-name>")
	}

	// Move from quarantine to vectorstore
	if err := os.Rename(quarantinePath, targetPath); err != nil {
		return fmt.Errorf("failed to restore collection: %w", err)
	}

	fmt.Printf("✅ Successfully restored collection %s from quarantine\n", hash)
	fmt.Printf("   Path: %s\n", targetPath)
	fmt.Printf("\nRestart contextd to load the restored collection.\n")

	return nil
}
