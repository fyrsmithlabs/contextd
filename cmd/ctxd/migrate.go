// Package main implements the ctxd CLI for manual operations against contextd.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	chromem "github.com/philippgille/chromem-go"
	"github.com/qdrant/go-client/qdrant"
	"github.com/spf13/cobra"
)

var (
	// Qdrant source flags
	qdrantHost       string
	qdrantPort       int
	qdrantCollection string

	// Chromem destination flags
	chromemPath       string
	chromemCollection string
	chromemCompress   bool

	// Migration options
	batchSize int
	dryRun    bool
)

func init() {
	migrateCmd.Flags().StringVar(&qdrantHost, "qdrant-host", "localhost", "Qdrant server host")
	migrateCmd.Flags().IntVar(&qdrantPort, "qdrant-port", 6334, "Qdrant gRPC port")
	migrateCmd.Flags().StringVar(&qdrantCollection, "qdrant-collection", "", "Qdrant collection to migrate (required, or 'all' for all collections)")
	migrateCmd.Flags().StringVar(&chromemPath, "chromem-path", "~/.config/contextd/vectorstore", "Chromem storage path")
	migrateCmd.Flags().StringVar(&chromemCollection, "chromem-collection", "", "Chromem collection name (defaults to source name)")
	migrateCmd.Flags().BoolVar(&chromemCompress, "chromem-compress", false, "Enable gzip compression for Chromem")
	migrateCmd.Flags().IntVar(&batchSize, "batch-size", 100, "Number of documents per batch")
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be migrated without actually migrating")

	_ = migrateCmd.MarkFlagRequired("qdrant-collection")

	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate data from Qdrant to Chromem",
	Long: `Migrate vector data from Qdrant to Chromem (embedded vector database).

This command exports all documents with their embeddings and metadata from
a Qdrant collection and imports them into a Chromem database.

Examples:
  # Migrate a single collection
  ctxd migrate --qdrant-collection=contextd_memories --chromem-path=/data/vectorstore

  # Migrate all collections
  ctxd migrate --qdrant-collection=all

  # Dry run to see what would be migrated
  ctxd migrate --qdrant-collection=contextd_memories --dry-run

  # Custom Qdrant server
  ctxd migrate --qdrant-host=qdrant.example.com --qdrant-port=6334 --qdrant-collection=my_collection`,
	RunE: runMigrate,
}

func runMigrate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Expand chromem path
	expandedPath := expandPath(chromemPath)

	fmt.Printf("Migration: Qdrant -> Chromem\n")
	fmt.Printf("  Source: %s:%d\n", qdrantHost, qdrantPort)
	fmt.Printf("  Destination: %s\n", expandedPath)
	fmt.Printf("  Batch size: %d\n", batchSize)
	if dryRun {
		fmt.Printf("  Mode: DRY RUN (no changes will be made)\n")
	}
	fmt.Println()

	// Connect to Qdrant
	qdrantAddr := fmt.Sprintf("%s:%d", qdrantHost, qdrantPort)
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: qdrantHost,
		Port: qdrantPort,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to Qdrant at %s: %w", qdrantAddr, err)
	}

	// Get collections to migrate
	var collections []string
	if qdrantCollection == "all" {
		result, err := client.ListCollections(ctx)
		if err != nil {
			return fmt.Errorf("failed to list Qdrant collections: %w", err)
		}
		for _, c := range result {
			collections = append(collections, c)
		}
		if len(collections) == 0 {
			fmt.Println("No collections found in Qdrant")
			return nil
		}
		fmt.Printf("Found %d collections to migrate\n\n", len(collections))
	} else {
		collections = []string{qdrantCollection}
	}

	// Create Chromem DB (unless dry run)
	var chromemDB *chromem.DB
	if !dryRun {
		if err := os.MkdirAll(expandedPath, 0755); err != nil {
			return fmt.Errorf("failed to create chromem directory: %w", err)
		}
		chromemDB, err = chromem.NewPersistentDB(expandedPath, chromemCompress)
		if err != nil {
			return fmt.Errorf("failed to create Chromem DB: %w", err)
		}
	}

	// Migrate each collection
	totalDocs := 0
	for _, collName := range collections {
		count, err := migrateCollection(ctx, client, chromemDB, collName)
		if err != nil {
			return fmt.Errorf("failed to migrate collection %s: %w", collName, err)
		}
		totalDocs += count
	}

	fmt.Printf("\n========================================\n")
	if dryRun {
		fmt.Printf("DRY RUN: Would migrate %d documents from %d collection(s)\n", totalDocs, len(collections))
	} else {
		fmt.Printf("Migration complete: %d documents from %d collection(s)\n", totalDocs, len(collections))
	}
	fmt.Printf("========================================\n")

	return nil
}

func migrateCollection(ctx context.Context, client *qdrant.Client, chromemDB *chromem.DB, collName string) (int, error) {
	fmt.Printf("Migrating collection: %s\n", collName)

	// Get collection info
	collInfo, err := client.GetCollectionInfo(ctx, collName)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection info: %w", err)
	}

	pointCount := collInfo.GetPointsCount()
	vectorSize := collInfo.GetConfig().GetParams().GetVectorsConfig().GetParams().GetSize()

	fmt.Printf("  Points: %d\n", pointCount)
	fmt.Printf("  Vector size: %d\n", vectorSize)

	if pointCount == 0 {
		fmt.Printf("  Skipping empty collection\n\n")
		return 0, nil
	}

	// Determine target collection name
	targetCollName := chromemCollection
	if targetCollName == "" {
		targetCollName = collName
	}

	// Create Chromem collection (unless dry run)
	var chromemColl *chromem.Collection
	if !dryRun && chromemDB != nil {
		// Create embedding function that just returns pre-computed embeddings
		// We'll use a placeholder since we're providing embeddings directly
		embeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
			// This should never be called since we provide embeddings directly
			return nil, fmt.Errorf("embedding function should not be called during migration")
		}

		chromemColl, err = chromemDB.GetOrCreateCollection(targetCollName, nil, embeddingFunc)
		if err != nil {
			return 0, fmt.Errorf("failed to create Chromem collection: %w", err)
		}
	}

	// Scroll through all points
	var offset *qdrant.PointId
	migratedCount := 0
	batchNum := 0

	for {
		// Fetch batch of points with vectors and payload
		points, nextOffset, err := client.ScrollAndOffset(ctx, &qdrant.ScrollPoints{
			CollectionName: collName,
			Offset:         offset,
			Limit:          qdrant.PtrOf(uint32(batchSize)),
			WithPayload:    qdrant.NewWithPayload(true),
			WithVectors:    qdrant.NewWithVectors(true),
		})
		if err != nil {
			return migratedCount, fmt.Errorf("failed to scroll points: %w", err)
		}

		if len(points) == 0 {
			break
		}

		batchNum++
		fmt.Printf("  Batch %d: %d points", batchNum, len(points))

		if !dryRun && chromemColl != nil {
			// Convert and insert into Chromem
			docs := make([]chromem.Document, 0, len(points))
			for _, point := range points {
				doc := convertQdrantPoint(point)
				if doc != nil {
					docs = append(docs, *doc)
				}
			}

			if len(docs) > 0 {
				if err := chromemColl.AddDocuments(ctx, docs, 1); err != nil {
					return migratedCount, fmt.Errorf("failed to add documents to Chromem: %w", err)
				}
			}
			fmt.Printf(" -> migrated %d\n", len(docs))
		} else {
			fmt.Printf(" (dry run)\n")
		}

		migratedCount += len(points)

		// Check if we've reached the end
		if nextOffset == nil || len(points) < batchSize {
			break
		}
		offset = nextOffset
	}

	fmt.Printf("  Total migrated: %d\n\n", migratedCount)
	return migratedCount, nil
}

func convertQdrantPoint(point *qdrant.RetrievedPoint) *chromem.Document {
	if point == nil {
		return nil
	}

	// Get point ID
	var id string
	switch pid := point.Id.PointIdOptions.(type) {
	case *qdrant.PointId_Uuid:
		id = pid.Uuid
	case *qdrant.PointId_Num:
		id = fmt.Sprintf("%d", pid.Num)
	}

	// Get vector
	var embedding []float32
	if vectors := point.GetVectors(); vectors != nil {
		if v := vectors.GetVector(); v != nil {
			embedding = v.GetData()
		}
	}

	if embedding == nil {
		return nil // Skip points without vectors
	}

	// Get content from payload
	content := ""
	metadata := make(map[string]string)

	if point.Payload != nil {
		for k, v := range point.Payload {
			switch val := v.Kind.(type) {
			case *qdrant.Value_StringValue:
				if k == "content" || k == "text" {
					content = val.StringValue
				} else {
					metadata[k] = val.StringValue
				}
			case *qdrant.Value_IntegerValue:
				metadata[k] = fmt.Sprintf("%d", val.IntegerValue)
			case *qdrant.Value_DoubleValue:
				metadata[k] = fmt.Sprintf("%f", val.DoubleValue)
			case *qdrant.Value_BoolValue:
				metadata[k] = fmt.Sprintf("%t", val.BoolValue)
			}
		}
	}

	return &chromem.Document{
		ID:        id,
		Content:   content,
		Metadata:  metadata,
		Embedding: embedding,
	}
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
