package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Document struct {
	ID        string
	Content   string
	Metadata  map[string]string
	Embedding []float32
}

func main() {
	// Check a document from collection 48ca0c90 which should have been migrated
	gobFile := filepath.Join(os.Getenv("HOME"), ".config/contextd/vectorstore/48ca0c90")

	// List files in the collection
	files, err := filepath.Glob(filepath.Join(gobFile, "*.gob"))
	if err != nil {
		log.Fatal(err)
	}

	// Read first non-metadata file
	for _, file := range files {
		if filepath.Base(file) == "00000000.gob" {
			continue
		}

		f, err := os.Open(file)
		if err != nil {
			log.Printf("Error opening %s: %v", file, err)
			continue
		}
		defer f.Close()

		var doc Document
		dec := gob.NewDecoder(f)
		if err := dec.Decode(&doc); err != nil {
			log.Printf("Error decoding %s: %v", file, err)
			continue
		}

		fmt.Printf("Document ID: %s\n", doc.ID)
		fmt.Printf("Content length: %d\n", len(doc.Content))
		fmt.Printf("Has embedding: %v (len=%d)\n", len(doc.Embedding) > 0, len(doc.Embedding))
		fmt.Printf("Metadata:\n")
		for k, v := range doc.Metadata {
			fmt.Printf("  %s: %s\n", k, v)
		}
		break
	}
}
