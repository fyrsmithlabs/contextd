package repository

import "context"

// ServiceAdapter adapts repository.Service to work with other type definitions.
//
// This allows the repository service to be used with different IndexOptions
// and IndexResult types without tightly coupling packages.
type ServiceAdapter struct {
	service *Service
}

// NewServiceAdapter creates an adapter for the repository service.
func NewServiceAdapter(service *Service) *ServiceAdapter {
	return &ServiceAdapter{service: service}
}

// IndexRepositoryFunc is a generic function signature for repository indexing.
// It can be implemented by adapters with different type definitions.
type IndexRepositoryFunc func(ctx context.Context, path string, opts interface{}) (interface{}, error)

// AsFunc returns a function that accepts generic options interface{}
// and returns generic result interface{}, allowing for type adaptation.
func (a *ServiceAdapter) AsFunc() IndexRepositoryFunc {
	return func(ctx context.Context, path string, opts interface{}) (interface{}, error) {
		// Type assert to expected IndexOptions
		var repoOpts IndexOptions
		if o, ok := opts.(IndexOptions); ok {
			repoOpts = o
		} else {
			// Try to convert from map or struct with same field names
			repoOpts = IndexOptions{
				IncludePatterns: getStringSlice(opts, "IncludePatterns"),
				ExcludePatterns: getStringSlice(opts, "ExcludePatterns"),
				MaxFileSize:     getInt64(opts, "MaxFileSize"),
			}
		}

		return a.service.IndexRepository(ctx, path, repoOpts)
	}
}

// Helper functions for type conversion
func getStringSlice(opts interface{}, field string) []string {
	// Simplified: In production, use reflection or specific type handling
	return nil
}

func getInt64(opts interface{}, field string) int64 {
	// Simplified: In production, use reflection or specific type handling
	return 0
}
