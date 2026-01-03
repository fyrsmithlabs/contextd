package workflows

import (
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

const instrumentationName = "github.com/fyrsmithlabs/contextd/internal/workflows"

// Metrics for version validation workflow
var (
	versionValidationCounter   metric.Int64Counter
	versionValidationDuration  metric.Float64Histogram
	versionMismatchCounter     metric.Int64Counter
	versionMatchCounter        metric.Int64Counter
	activityDuration           metric.Float64Histogram
	activityErrorCounter       metric.Int64Counter
)

// initMetrics initializes OpenTelemetry metrics for workflows.
// This is called once during package initialization.
func initMetrics() {
	meter := otel.Meter(instrumentationName)

	var err error

	// Workflow execution counters
	versionValidationCounter, err = meter.Int64Counter(
		"contextd.workflows.version_validation.executions",
		metric.WithDescription("Total number of version validation workflow executions"),
		metric.WithUnit("{execution}"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create version validation counter: %v", err))
	}

	// Workflow duration histogram
	versionValidationDuration, err = meter.Float64Histogram(
		"contextd.workflows.version_validation.duration",
		metric.WithDescription("Duration of version validation workflow executions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create version validation duration: %v", err))
	}

	// Version mismatch counter
	versionMismatchCounter, err = meter.Int64Counter(
		"contextd.workflows.version_validation.mismatches",
		metric.WithDescription("Number of version mismatches detected"),
		metric.WithUnit("{mismatch}"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create version mismatch counter: %v", err))
	}

	// Version match counter
	versionMatchCounter, err = meter.Int64Counter(
		"contextd.workflows.version_validation.matches",
		metric.WithDescription("Number of version matches detected"),
		metric.WithUnit("{match}"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create version match counter: %v", err))
	}

	// Activity duration histogram
	activityDuration, err = meter.Float64Histogram(
		"contextd.workflows.activity.duration",
		metric.WithDescription("Duration of workflow activity executions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create activity duration: %v", err))
	}

	// Activity error counter
	activityErrorCounter, err = meter.Int64Counter(
		"contextd.workflows.activity.errors",
		metric.WithDescription("Number of activity execution errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create activity error counter: %v", err))
	}
}

func init() {
	initMetrics()
}
