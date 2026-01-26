// Package vectorstore provides Prometheus metrics for health monitoring.
package vectorstore

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CollectionsTotal tracks the number of collections by status.
	// Labels: status (healthy, corrupt, empty)
	CollectionsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "contextd",
			Subsystem: "vectorstore",
			Name:      "collections_total",
			Help:      "Total number of collections by health status",
		},
		[]string{"status"},
	)

	// HealthCheckDuration tracks how long health checks take.
	HealthCheckDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "contextd",
			Subsystem: "vectorstore",
			Name:      "health_check_duration_seconds",
			Help:      "Duration of health check operations in seconds",
			Buckets:   prometheus.DefBuckets,
		},
	)

	// HealthCheckTotal counts health check operations.
	// Labels: result (success, error)
	HealthCheckTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "contextd",
			Subsystem: "vectorstore",
			Name:      "health_checks_total",
			Help:      "Total number of health check operations",
		},
		[]string{"result"},
	)

	// HealthStatus indicates current health status (1=healthy, 0=degraded).
	HealthStatus = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "contextd",
			Subsystem: "vectorstore",
			Name:      "health_status",
			Help:      "Current health status (1=healthy, 0=degraded)",
		},
	)

	// CorruptCollectionsDetected counts corruption detections.
	CorruptCollectionsDetected = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "contextd",
			Subsystem: "vectorstore",
			Name:      "corrupt_collections_detected_total",
			Help:      "Total number of corrupt collections detected across all health checks",
		},
	)

	// QuarantineOperations counts quarantine operations.
	// Labels: result (success, error)
	QuarantineOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "contextd",
			Subsystem: "vectorstore",
			Name:      "quarantine_operations_total",
			Help:      "Total number of quarantine operations",
		},
		[]string{"result"},
	)
)

// UpdateHealthMetrics updates Prometheus metrics from a MetadataHealth result.
func UpdateHealthMetrics(health *MetadataHealth) {
	if health == nil {
		return
	}

	// Update collection counts by status
	CollectionsTotal.WithLabelValues("healthy").Set(float64(health.HealthyCount))
	CollectionsTotal.WithLabelValues("corrupt").Set(float64(health.CorruptCount))
	CollectionsTotal.WithLabelValues("empty").Set(float64(len(health.Empty)))

	// Update health status gauge
	if health.IsHealthy() {
		HealthStatus.Set(1)
	} else {
		HealthStatus.Set(0)
	}

	// Update check duration
	HealthCheckDuration.Observe(health.CheckDuration.Seconds())

	// Count corrupt collections detected
	if health.CorruptCount > 0 {
		CorruptCollectionsDetected.Add(float64(health.CorruptCount))
	}
}

// RecordHealthCheckResult records the outcome of a health check.
func RecordHealthCheckResult(success bool) {
	if success {
		HealthCheckTotal.WithLabelValues("success").Inc()
	} else {
		HealthCheckTotal.WithLabelValues("error").Inc()
	}
}

// RecordQuarantineResult records the outcome of a quarantine operation.
func RecordQuarantineResult(success bool) {
	if success {
		QuarantineOperations.WithLabelValues("success").Inc()
	} else {
		QuarantineOperations.WithLabelValues("error").Inc()
	}
}
