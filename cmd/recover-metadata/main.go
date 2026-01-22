// Package main provides a utility to recover missing metadata files in chromem collections.
package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	// Collection name and path
	// NOTE: This tool is designed for emergency manual recovery with hardcoded values.
	// For production use, modify these values for your specific collection.
	collectionName := "contextd_memories"
	collectionHash := "e9f85bf6"

	// Validate collection hash format (must be 8-char hex)
	hashPattern := regexp.MustCompile(`^[a-f0-9]{8}$`)
	if !hashPattern.MatchString(collectionHash) {
		fmt.Fprintf(os.Stderr, "Invalid collection hash format: %s (expected 8-character hex)\n", collectionHash)
		os.Exit(1)
	}

	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	// Construct paths
	collectionPath := filepath.Join(home, ".config", "contextd", "vectorstore", collectionHash)
	metadataPath := filepath.Join(collectionPath, "00000000.gob")

	// Check if collection directory exists
	if _, err := os.Stat(collectionPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Collection directory does not exist: %s\n", collectionPath)
		os.Exit(1)
	}

	// Check if metadata file already exists
	if _, err := os.Stat(metadataPath); err == nil {
		fmt.Println("Metadata file already exists at:", metadataPath)
		fmt.Println("Skipping recovery.")
		return
	}

	// Create metadata structure matching chromem's format
	type persistedCollection struct {
		Name     string
		Metadata map[string]string
	}

	pc := persistedCollection{
		Name:     collectionName,
		Metadata: map[string]string{},
	}

	// Create metadata file
	f, err := os.Create(metadataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating metadata file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// Encode and write
	enc := gob.NewEncoder(f)
	if err := enc.Encode(pc); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding metadata: %v\n", err)
		os.Exit(1)
	}

	// Sync to ensure data is written
	if err := f.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "Error syncing metadata file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Successfully created metadata file for collection:", collectionName)
	fmt.Println("   Path:", metadataPath)
	fmt.Println("   Collection hash:", collectionHash)
	fmt.Println("\nYou can now restart contextd.")
}
