package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Document represents the internal structure stored in .gob files
type Document struct {
	ID        string
	Content   string
	Metadata  map[string]string
	Embedding []float32
}

// CollectionMetadata represents the 00000000.gob metadata structure
type CollectionMetadata struct {
	Name     string
	Metadata map[string]string
}

func main() {
	var (
		storePath     = flag.String("store", filepath.Join(os.Getenv("HOME"), ".config/contextd/vectorstore"), "Path to vectorstore")
		oldCollection = flag.String("old", "contextd_memories", "Old collection name")
		newCollection = flag.String("new", "fyrsmithlabs_contextd_memories", "New collection name")
		oldProjectID  = flag.String("old-project", "contextd", "Old project_id in metadata")
		newProjectID  = flag.String("new-project", "fyrsmithlabs_contextd", "New project_id in metadata")
		dryRun        = flag.Bool("dry-run", false, "Dry run - don't actually make changes")
	)
	flag.Parse()

	log.Printf("Collection Migration")
	log.Printf("  Collection: %q -> %q", *oldCollection, *newCollection)
	log.Printf("  Project ID: %q -> %q", *oldProjectID, *newProjectID)
	log.Printf("  Vectorstore: %s", *storePath)
	log.Printf("  Dry run: %v", *dryRun)

	// Expand home directory
	expandedPath := expandPath(*storePath)

	// Find collection directory by name
	oldCollDir, err := findCollectionByName(expandedPath, *oldCollection)
	if err != nil {
		log.Fatalf("Failed to find collection %q: %v", *oldCollection, err)
	}
	log.Printf("  Found source collection at: %s", filepath.Base(oldCollDir))

	// Check if new collection already exists
	newCollDir, err := findCollectionByName(expandedPath, *newCollection)
	if err == nil && newCollDir != "" {
		log.Fatalf("Target collection %q already exists at %s", *newCollection, filepath.Base(newCollDir))
	}

	if *dryRun {
		log.Printf("\n[DRY RUN] Would perform the following:")
		log.Printf("  1. Copy collection directory")
		log.Printf("  2. Rename collection in metadata")
		log.Printf("  3. Update project_id in all documents")
		showDocumentPreview(oldCollDir, *oldProjectID, *newProjectID)
		return
	}

	// Generate new directory name (hash-like, using timestamp for uniqueness)
	newDirName := generateCollectionDirName()
	newCollDir = filepath.Join(expandedPath, newDirName)

	log.Printf("\nStep 1: Copying collection directory to %s", newDirName)
	if err := copyDir(oldCollDir, newCollDir); err != nil {
		log.Fatalf("Failed to copy collection: %v", err)
	}

	log.Printf("Step 2: Updating collection metadata")
	if err := updateCollectionMetadata(newCollDir, *newCollection); err != nil {
		log.Fatalf("Failed to update metadata: %v", err)
	}

	log.Printf("Step 3: Updating document metadata")
	updated, err := updateDocumentsMetadata(newCollDir, *oldProjectID, *newProjectID)
	if err != nil {
		log.Fatalf("Failed to update documents: %v", err)
	}

	log.Printf("\n=== Migration Complete ===")
	log.Printf("  New collection: %s", *newCollection)
	log.Printf("  Directory: %s", newDirName)
	log.Printf("  Documents updated: %d", updated)
	log.Printf("\nNote: Old collection %q still exists. Remove manually if no longer needed.", *oldCollection)
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	return path
}

func findCollectionByName(storePath, collectionName string) (string, error) {
	entries, err := os.ReadDir(storePath)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metaPath := filepath.Join(storePath, entry.Name(), "00000000.gob")
		name, err := readCollectionName(metaPath)
		if err != nil {
			continue
		}

		if name == collectionName {
			return filepath.Join(storePath, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("collection %q not found", collectionName)
}

func readCollectionName(metaPath string) (string, error) {
	file, err := os.Open(metaPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var meta CollectionMetadata
	dec := gob.NewDecoder(file)
	if err := dec.Decode(&meta); err != nil {
		return "", err
	}

	return meta.Name, nil
}

func generateCollectionDirName() string {
	// Generate a hex string similar to existing directories
	// Using current timestamp for uniqueness
	return fmt.Sprintf("%08x", uint32(os.Getpid())^uint32(os.Getuid()))
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0700); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func updateCollectionMetadata(collDir, newName string) error {
	metaPath := filepath.Join(collDir, "00000000.gob")

	// Read existing metadata
	file, err := os.Open(metaPath)
	if err != nil {
		return err
	}

	var meta CollectionMetadata
	dec := gob.NewDecoder(file)
	if err := dec.Decode(&meta); err != nil {
		file.Close()
		return err
	}
	file.Close()

	log.Printf("  Old name: %s", meta.Name)
	meta.Name = newName
	log.Printf("  New name: %s", meta.Name)

	// Write updated metadata
	file, err = os.Create(metaPath)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := gob.NewEncoder(file)
	return enc.Encode(&meta)
}

func updateDocumentsMetadata(collDir, oldProjectID, newProjectID string) (int, error) {
	gobFiles, err := filepath.Glob(filepath.Join(collDir, "*.gob"))
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, gobFile := range gobFiles {
		// Skip metadata file
		if strings.HasSuffix(gobFile, "00000000.gob") {
			continue
		}

		doc, err := readDocument(gobFile)
		if err != nil {
			log.Printf("  Warning: Could not read %s: %v", filepath.Base(gobFile), err)
			continue
		}

		// Update project_id if it matches old value
		needsUpdate := false
		if pid, ok := doc.Metadata["project_id"]; ok && pid == oldProjectID {
			doc.Metadata["project_id"] = newProjectID
			needsUpdate = true
		}

		// Also check for project path patterns
		for key, val := range doc.Metadata {
			if strings.Contains(val, "/projects/contextd") && !strings.Contains(val, "fyrsmithlabs") {
				doc.Metadata[key] = strings.ReplaceAll(val, "/projects/contextd", "/projects/fyrsmithlabs/contextd")
				needsUpdate = true
			}
		}

		if needsUpdate {
			if err := writeDocument(gobFile, doc); err != nil {
				log.Printf("  Warning: Could not write %s: %v", filepath.Base(gobFile), err)
				continue
			}
			updated++
			log.Printf("  Updated: %s (project_id: %s)", doc.ID, newProjectID)
		}
	}

	return updated, nil
}

func showDocumentPreview(collDir, oldProjectID, newProjectID string) {
	gobFiles, err := filepath.Glob(filepath.Join(collDir, "*.gob"))
	if err != nil {
		return
	}

	log.Printf("\nDocument preview:")
	count := 0
	for _, gobFile := range gobFiles {
		if strings.HasSuffix(gobFile, "00000000.gob") {
			continue
		}

		doc, err := readDocument(gobFile)
		if err != nil {
			continue
		}

		if pid, ok := doc.Metadata["project_id"]; ok && pid == oldProjectID {
			log.Printf("  %s: project_id %q -> %q", doc.ID[:8], oldProjectID, newProjectID)
			count++
		}
	}
	log.Printf("  Total documents to update: %d", count)
}

func readDocument(path string) (*Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var doc Document
	dec := gob.NewDecoder(file)
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

func writeDocument(path string, doc *Document) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := gob.NewEncoder(file)
	return enc.Encode(doc)
}
