package prefetch

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// Metrics holds Prometheus metrics for the pre-fetch engine.
type Metrics struct {
	// Git event detection
	GitEventsTotal *prometheus.CounterVec

	// Rule execution
	RulesExecutedTotal *prometheus.CounterVec
	RuleTimeoutsTotal  *prometheus.CounterVec
	RuleDuration       *prometheus.HistogramVec

	// Cache performance
	CacheHitsTotal   prometheus.Counter
	CacheMissesTotal prometheus.Counter
	CacheSize        prometheus.Gauge

	// Token savings estimation
	TokensSavedTotal prometheus.Counter
}

// NewMetrics creates and registers Prometheus metrics for pre-fetch engine.
//
// This function uses sync.Once to ensure metrics are only registered once
// globally, preventing "duplicate metrics collector registration" panics.
//
// All metrics are prefixed with "prefetch_" for namespacing.
//
// Metrics:
//   - prefetch_git_events_total{type} - Count of git events detected
//   - prefetch_rules_executed_total{rule} - Count of rules executed
//   - prefetch_rule_timeouts_total{rule} - Count of rule timeouts
//   - prefetch_rule_duration_seconds{rule} - Histogram of rule execution times
//   - prefetch_cache_hits_total - Count of cache hits
//   - prefetch_cache_misses_total - Count of cache misses
//   - prefetch_cache_size - Current number of cached projects
//   - prefetch_tokens_saved_total - Estimated tokens saved
func NewMetrics() *Metrics {
	metricsOnce.Do(func() {
		globalMetrics = &Metrics{
			GitEventsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "prefetch_git_events_total",
					Help: "Total number of git events detected",
				},
				[]string{"type"}, // "branch_switch" or "new_commit"
			),

			RulesExecutedTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "prefetch_rules_executed_total",
					Help: "Total number of pre-fetch rules executed",
				},
				[]string{"rule"}, // "branch_diff", "recent_commit", "common_files"
			),

			RuleTimeoutsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "prefetch_rule_timeouts_total",
					Help: "Total number of rule execution timeouts",
				},
				[]string{"rule"},
			),

			RuleDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "prefetch_rule_duration_seconds",
					Help:    "Duration of rule execution in seconds",
					Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
				},
				[]string{"rule"},
			),

			CacheHitsTotal: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "prefetch_cache_hits_total",
					Help: "Total number of pre-fetch cache hits",
				},
			),

			CacheMissesTotal: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "prefetch_cache_misses_total",
					Help: "Total number of pre-fetch cache misses",
				},
			),

			CacheSize: promauto.NewGauge(
				prometheus.GaugeOpts{
					Name: "prefetch_cache_size",
					Help: "Current number of projects in pre-fetch cache",
				},
			),

			TokensSavedTotal: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "prefetch_tokens_saved_total",
					Help: "Estimated total tokens saved by pre-fetch cache hits",
				},
			),
		}
	})

	return globalMetrics
}

// RecordGitEvent records a git event detection.
func (m *Metrics) RecordGitEvent(eventType string) {
	m.GitEventsTotal.WithLabelValues(eventType).Inc()
}

// RecordRuleExecution records successful rule execution with duration.
func (m *Metrics) RecordRuleExecution(rule string, durationSeconds float64) {
	m.RulesExecutedTotal.WithLabelValues(rule).Inc()
	m.RuleDuration.WithLabelValues(rule).Observe(durationSeconds)
}

// RecordRuleTimeout records a rule timeout.
func (m *Metrics) RecordRuleTimeout(rule string) {
	m.RuleTimeoutsTotal.WithLabelValues(rule).Inc()
}

// RecordCacheHit records a cache hit and estimated tokens saved.
//
// estimatedTokens is a rough estimate of tokens that would have been used
// without the cache hit (e.g., based on result size).
func (m *Metrics) RecordCacheHit(estimatedTokens int) {
	m.CacheHitsTotal.Inc()
	m.TokensSavedTotal.Add(float64(estimatedTokens))
}

// RecordCacheMiss records a cache miss.
func (m *Metrics) RecordCacheMiss() {
	m.CacheMissesTotal.Inc()
}

// SetCacheSize updates the current cache size gauge.
func (m *Metrics) SetCacheSize(size int) {
	m.CacheSize.Set(float64(size))
}
