// Package http provides HTTP API for contextd.
package http

// StatusResponse is the response body for GET /api/v1/status.
type StatusResponse struct {
	Status      string             `json:"status"`
	Version     string             `json:"version,omitempty"`
	Services    map[string]string  `json:"services"`
	Counts      StatusCounts       `json:"counts"`
	Context     *ContextStatus     `json:"context,omitempty"`
	Compression *CompressionStatus `json:"compression,omitempty"`
	Memory      *MemoryStatus      `json:"memory,omitempty"`
}

// StatusCounts contains count information for various resources.
type StatusCounts struct {
	Checkpoints int `json:"checkpoints"`
	Memories    int `json:"memories"`
}

// ContextStatus contains context usage information.
type ContextStatus struct {
	UsagePercent     int  `json:"usage_percent"`
	ThresholdWarning bool `json:"threshold_warning"`
}

// CompressionStatus contains compression metrics.
type CompressionStatus struct {
	LastRatio       float64 `json:"last_ratio"`
	LastQuality     float64 `json:"last_quality"`
	OperationsTotal int64   `json:"operations_total"`
}

// MemoryStatus contains memory/reasoning bank metrics.
type MemoryStatus struct {
	LastConfidence float64 `json:"last_confidence"`
}

// MetadataHealthStatus contains metadata integrity health information.
type MetadataHealthStatus struct {
	Status        string   `json:"status"`         // "healthy" or "degraded"
	HealthyCount  int      `json:"healthy_count"`  // Number of healthy collections
	CorruptCount  int      `json:"corrupt_count"`  // Number of corrupt collections
	EmptyCount    int      `json:"empty_count"`    // Number of empty collections
	Total         int      `json:"total"`          // Total collections
	CorruptHashes []string `json:"corrupt_hashes"` // List of corrupt collection hashes
}
