// Package http provides HTTP API for contextd.
package http

// StatusResponse is the response body for GET /api/v1/status.
type StatusResponse struct {
	Status      string             `json:"status"`
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
