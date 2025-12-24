package main

import (
	"encoding/gob"
	"flag"
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

func main() {
	var (
		storePath   = flag.String("store", filepath.Join(os.Getenv("HOME"), ".config/contextd/vectorstore"), "Path to vectorstore")
		oldTenantID = flag.String("old", "dahendel", "Old tenant ID to migrate from")
		newTenantID = flag.String("new", "fyrsmithlabs", "New tenant ID to migrate to")
		dryRun      = flag.Bool("dry-run", false, "Dry run - don't actually update")
	)
	flag.Parse()

	log.Printf("Migrating tenant_id: %q -> %q", *oldTenantID, *newTenantID)
	log.Printf("Vectorstore path: %s", *storePath)
	log.Printf("Dry run: %v", *dryRun)

	// Expand home directory
	if len(*storePath) > 0 && (*storePath)[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get home dir: %v", err)
		}
		*storePath = filepath.Join(home, (*storePath)[1:])
	}

	// List all collection directories
	entries, err := os.ReadDir(*storePath)
	if err != nil {
		log.Fatalf("Failed to read vectorstore directory: %v", err)
	}

	log.Printf("Found %d collections", len(entries))

	var totalDocs, updatedDocs, totalCollections int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		collectionPath := filepath.Join(*storePath, entry.Name())
		totalCollections++

		// Find all .gob files except 00000000.gob (metadata file)
		gobFiles, err := filepath.Glob(filepath.Join(collectionPath, "*.gob"))
		if err != nil {
			log.Printf("  Error globbing %s: %v", entry.Name(), err)
			continue
		}

		var docsInCollection, updatedInCollection int

		for _, gobFile := range gobFiles {
			// Skip metadata file
			if strings.HasSuffix(gobFile, "00000000.gob") {
				continue
			}

			// Read document
			doc, err := readDocument(gobFile)
			if err != nil {
				continue // Skip files that can't be read
			}

			docsInCollection++
			totalDocs++

			// Check if this document has the old tenant_id
			if tenantID, ok := doc.Metadata["tenant_id"]; ok && tenantID == *oldTenantID {
				updatedInCollection++
				updatedDocs++

				if *dryRun {
					if updatedInCollection <= 3 {
						log.Printf("  [DRY RUN] Would update %s in %s", doc.ID, entry.Name())
					}
					continue
				}

				// Update tenant_id
				doc.Metadata["tenant_id"] = *newTenantID

				// Write back
				if err := writeDocument(gobFile, doc); err != nil {
					log.Printf("  Error writing %s: %v", gobFile, err)
					continue
				}
			}
		}

		if updatedInCollection > 0 {
			log.Printf("Collection %s: %d docs, %d with old tenant_id", entry.Name(), docsInCollection, updatedInCollection)
		}
	}

	log.Printf("\n=== Migration Summary ===")
	log.Printf("Collections processed: %d", totalCollections)
	log.Printf("Total documents found: %d", totalDocs)
	log.Printf("Documents with old tenant_id: %d", updatedDocs)
	if *dryRun {
		log.Printf("\n⚠️  DRY RUN - No changes were made")
	} else {
		log.Printf("\n✓ Migration complete!")
	}
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
