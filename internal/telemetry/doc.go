// Package telemetry provides OpenTelemetry instrumentation for contextd.
//
// # Overview
//
// This package implements distributed tracing and metrics collection using the
// OpenTelemetry Go SDK. It exports telemetry data to an OTEL Collector, which
// forwards to VictoriaMetrics (metrics, logs, traces).
//
// # Usage
//
// Create telemetry instance:
//
//	cfg := telemetry.NewDefaultConfig()
//	tel, err := telemetry.New(ctx, cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer tel.Shutdown(ctx)
//
// Use tracer and meter:
//
//	tracer := tel.Tracer("contextd.grpc")
//	ctx, span := tracer.Start(ctx, "SafeExec.Bash")
//	defer span.End()
//
//	meter := tel.Meter("contextd.grpc")
//	counter, _ := meter.Int64Counter("grpc.requests")
//	counter.Add(ctx, 1)
//
// # Configuration
//
//	telemetry:
//	  enabled: true
//	  endpoint: "localhost:4317"
//	  service_name: "contextd"
//	  sampling:
//	    rate: 1.0  # 100% in dev, lower in prod
//	    always_on_errors: true
//	  metrics:
//	    enabled: true
//	    export_interval: "15s"
//
// # Error Handling
//
// Telemetry failures do not crash the application. If telemetry cannot be
// initialized, the instance degrades gracefully and returns no-op providers.
//
// # Testing
//
// Use TestTelemetry for tests:
//
//	tt := telemetry.NewTestTelemetry()
//	tracer := tt.Tracer("test")
//	_, span := tracer.Start(ctx, "test-span")
//	span.End()
//	tt.AssertSpanExists(t, "test-span")
//
// See CLAUDE.md for instrumentation layers and key metrics.
package telemetry
