// Package http provides HTTP server functionality for contextd.
package http

import (
	"context"
	"strings"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// CountFromCollections counts checkpoints and memories from vector store collections.
//
// Collection names follow tenant naming conventions:
//   - org_checkpoints, org_memories
//   - {team}_checkpoints, {team}_memories
//   - {team}_{project}_checkpoints, {team}_{project}_memories
//
// Returns (-1, -1) if:
//   - store is nil
//   - listing collections fails
//   - collections list is empty (chromem lazy loading case)
//
// Otherwise returns the sum of point counts for matching collections.
func CountFromCollections(ctx context.Context, store vectorstore.Store) (checkpoints int, memories int) {
	if store == nil {
		return -1, -1
	}

	collections, err := store.ListCollections(ctx)
	if err != nil {
		return -1, -1
	}

	// chromem loads collections lazily - on fresh open, collections map is empty
	// until they are accessed. Return -1 to indicate we can't determine counts.
	if len(collections) == 0 {
		return -1, -1
	}

	for _, coll := range collections {
		info, err := store.GetCollectionInfo(ctx, coll)
		if err != nil || info == nil {
			continue
		}
		// Check for checkpoint collections (plural form per tenant spec)
		if strings.Contains(coll, "checkpoint") {
			checkpoints += info.PointCount
		}
		// Check for memory collections (memories per tenant spec, or reasoning for legacy)
		if strings.Contains(coll, "memor") || strings.Contains(coll, "reasoning") {
			memories += info.PointCount
		}
	}

	return checkpoints, memories
}
