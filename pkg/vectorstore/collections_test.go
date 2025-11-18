package vectorstore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateCollection tests creating a new collection.
func TestCreateCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		collectionName string
		vectorSize     int
		wantErr        bool
	}{
		{
			name:           "creates new collection",
			collectionName: "test_create_collection",
			vectorSize:     384,
			wantErr:        false,
		},
		{
			name:           "returns error for empty collection name",
			collectionName: "",
			vectorSize:     384,
			wantErr:        true,
		},
		{
			name:           "returns error for invalid vector size",
			collectionName: "test_invalid_size",
			vectorSize:     0,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create service
			config := ConfigFromEnv(tt.collectionName)
			service, err := NewService(config)
			if tt.collectionName == "" {
				require.Error(t, err) // Validation should fail in NewService
				return
			}
			require.NoError(t, err)

			// Execute
			err = service.CreateCollection(ctx, tt.collectionName, tt.vectorSize)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Cleanup
				defer func() {
					_ = service.DeleteCollection(ctx, tt.collectionName)
				}()

				// Verify collection exists by listing
				collections, err := service.ListCollections(ctx)
				require.NoError(t, err)
				assert.Contains(t, collections, tt.collectionName)
			}
		})
	}
}

// TestDeleteCollection tests deleting a collection.
func TestDeleteCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		collectionName string
		createFirst    bool
		wantErr        bool
	}{
		{
			name:           "deletes existing collection",
			collectionName: "test_delete_collection",
			createFirst:    true,
			wantErr:        false,
		},
		{
			name:           "returns error for non-existent collection",
			collectionName: "test_non_existent",
			createFirst:    false,
			wantErr:        true,
		},
		{
			name:           "returns error for empty collection name",
			collectionName: "",
			createFirst:    false,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create service
			config := ConfigFromEnv("test_service")
			service, err := NewService(config)
			require.NoError(t, err)

			// Create collection if needed
			if tt.createFirst {
				err = service.CreateCollection(ctx, tt.collectionName, 384)
				require.NoError(t, err)
			}

			// Execute
			err = service.DeleteCollection(ctx, tt.collectionName)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify collection doesn't exist
				collections, err := service.ListCollections(ctx)
				require.NoError(t, err)
				assert.NotContains(t, collections, tt.collectionName)
			}
		})
	}
}

// TestListCollections tests listing all collections.
func TestListCollections(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name              string
		createCollections []string
		wantMinCount      int
		checkContains     []string
	}{
		{
			name:              "lists all collections",
			createCollections: []string{"test_list_1", "test_list_2"},
			wantMinCount:      2,
			checkContains:     []string{"test_list_1", "test_list_2"},
		},
		{
			name:              "returns empty list when no collections",
			createCollections: []string{},
			wantMinCount:      0,
			checkContains:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create service
			config := ConfigFromEnv("test_service")
			service, err := NewService(config)
			require.NoError(t, err)

			// Create collections
			for _, collectionName := range tt.createCollections {
				err = service.CreateCollection(ctx, collectionName, 384)
				require.NoError(t, err)
				defer func(name string) {
					_ = service.DeleteCollection(ctx, name)
				}(collectionName)
			}

			// Execute
			collections, err := service.ListCollections(ctx)

			// Assert
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(collections), tt.wantMinCount)

			for _, expected := range tt.checkContains {
				assert.Contains(t, collections, expected)
			}
		})
	}
}

// TestCollectionExists tests checking if a collection exists.
func TestCollectionExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		collectionName string
		createFirst    bool
		wantExists     bool
	}{
		{
			name:           "returns true for existing collection",
			collectionName: "test_exists_true",
			createFirst:    true,
			wantExists:     true,
		},
		{
			name:           "returns false for non-existent collection",
			collectionName: "test_exists_false",
			createFirst:    false,
			wantExists:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create service
			config := ConfigFromEnv("test_service")
			service, err := NewService(config)
			require.NoError(t, err)

			// Create collection if needed
			if tt.createFirst {
				err = service.CreateCollection(ctx, tt.collectionName, 384)
				require.NoError(t, err)
				defer func() {
					_ = service.DeleteCollection(ctx, tt.collectionName)
				}()
			}

			// Execute
			exists, err := service.CollectionExists(ctx, tt.collectionName)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.wantExists, exists)
		})
	}
}
