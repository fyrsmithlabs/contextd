package project

import (
	"fmt"
)

// CollectionType represents the type of data stored in a collection.
type CollectionType string

const (
	// CollectionMemories stores ReasoningBank memories.
	CollectionMemories CollectionType = "memories"

	// CollectionCheckpoints stores session checkpoints.
	CollectionCheckpoints CollectionType = "checkpoints"

	// CollectionRemediations stores error fix patterns.
	CollectionRemediations CollectionType = "remediations"

	// CollectionSessions stores session traces.
	CollectionSessions CollectionType = "sessions"

	// CollectionCodebase stores code embeddings.
	CollectionCodebase CollectionType = "codebase"
)

// GetCollectionName returns the collection name for a project and type.
// Format: {project_id}_{type}
//
// Examples:
//   - "550e8400-e29b-41d4-a716-446655440000_memories"
//   - "550e8400-e29b-41d4-a716-446655440000_checkpoints"
func GetCollectionName(projectID string, collectionType CollectionType) (string, error) {
	if projectID == "" {
		return "", ErrEmptyProjectID
	}
	if collectionType == "" {
		return "", fmt.Errorf("collection type cannot be empty")
	}

	return fmt.Sprintf("%s_%s", projectID, collectionType), nil
}

// GetAllCollectionNames returns all collection names for a project.
func GetAllCollectionNames(projectID string) ([]string, error) {
	if projectID == "" {
		return nil, ErrEmptyProjectID
	}

	types := []CollectionType{
		CollectionMemories,
		CollectionCheckpoints,
		CollectionRemediations,
		CollectionSessions,
		CollectionCodebase,
	}

	names := make([]string, 0, len(types))
	for _, t := range types {
		name, err := GetCollectionName(projectID, t)
		if err != nil {
			return nil, fmt.Errorf("failed to get collection name for %s: %w", t, err)
		}
		names = append(names, name)
	}

	return names, nil
}
