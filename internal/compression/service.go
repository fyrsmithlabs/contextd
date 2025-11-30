package compression

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/fyrsmithlabs/contextd/internal/compression"
const meterName = "compression"

// Service orchestrates content compression operations
type Service struct {
	extractive  *ExtractiveCompressor
	abstractive *AbstractiveCompressor
	hybrid      *HybridCompressor
	config      Config

	tracer trace.Tracer
	meter  metric.Meter

	// Metrics
	compressionCounter metric.Int64Counter
	compressionTime    metric.Float64Histogram
	compressionRatio   metric.Float64Histogram
	compressionQuality metric.Float64Histogram
	compressionErrors  metric.Int64Counter
}

// NewService creates a new compression service
func NewService(config Config) (*Service, error) {
	s := &Service{
		extractive:  NewExtractiveCompressor(config),
		abstractive: NewAbstractiveCompressor(config),
		hybrid:      NewHybridCompressor(config),
		config:      config,
		tracer:      otel.Tracer(tracerName),
		meter:       otel.Meter(meterName),
	}

	if err := s.initMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	return s, nil
}

// Compress compresses content using the specified algorithm
func (s *Service) Compress(ctx context.Context, content string, algorithm Algorithm, targetRatio float64) (*Result, error) {
	ctx, span := s.tracer.Start(ctx, "compression.compress",
		trace.WithAttributes(
			attribute.String("algorithm", string(algorithm)),
			attribute.Float64("target_ratio", targetRatio),
			attribute.Int("content_length", len(content)),
		),
	)
	defer span.End()

	start := time.Now()

	// Validate inputs
	if len(content) == 0 {
		return nil, fmt.Errorf("content cannot be empty")
	}

	if targetRatio <= 1.0 {
		return nil, fmt.Errorf("target ratio must be greater than 1.0")
	}

	// Select compressor based on algorithm
	var compressor Compressor
	switch algorithm {
	case AlgorithmExtractive:
		compressor = s.extractive
	case AlgorithmAbstractive:
		compressor = s.abstractive
	case AlgorithmHybrid:
		compressor = s.hybrid
	default:
		compressor = s.extractive // Default to extractive
	}

	// Check capabilities
	caps := compressor.GetCapabilities(ctx)
	if len(content) > caps.MaxContentLength {
		return nil, fmt.Errorf("content length %d exceeds maximum %d for algorithm %s",
			len(content), caps.MaxContentLength, algorithm)
	}

	// Perform compression
	result, err := compressor.Compress(ctx, content, algorithm, targetRatio)
	if err != nil {
		span.RecordError(err)
		s.compressionErrors.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("algorithm", string(algorithm)),
				attribute.String("error_type", "compression_failed"),
			),
		)
		return nil, fmt.Errorf("compression failed: %w", err)
	}

	// Record metrics
	processingTime := float64(time.Since(start).Milliseconds()) / 1000.0 // Convert to seconds
	compressionRatio := result.Metadata.CompressionRatio

	s.compressionCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("algorithm", string(algorithm)),
			attribute.String("compression_level", string(result.Metadata.Level)),
		),
	)

	s.compressionTime.Record(ctx, processingTime,
		metric.WithAttributes(attribute.String("algorithm", string(algorithm))))

	s.compressionRatio.Record(ctx, compressionRatio,
		metric.WithAttributes(attribute.String("algorithm", string(algorithm))))

	s.compressionQuality.Record(ctx, result.QualityScore,
		metric.WithAttributes(attribute.String("algorithm", string(algorithm))))

	// Add span attributes
	span.SetAttributes(
		attribute.Float64("processing_time_s", processingTime),
		attribute.Float64("compression_ratio", compressionRatio),
		attribute.Float64("quality_score", result.QualityScore),
		attribute.Int("original_size", result.Metadata.OriginalSize),
		attribute.Int("compressed_size", result.Metadata.CompressedSize),
	)

	return result, nil
}

// GetCapabilities returns the capabilities of all supported algorithms
func (s *Service) GetCapabilities(ctx context.Context) map[Algorithm]Capabilities {
	return map[Algorithm]Capabilities{
		AlgorithmExtractive:  s.extractive.GetCapabilities(ctx),
		AlgorithmAbstractive: s.abstractive.GetCapabilities(ctx),
		AlgorithmHybrid:      s.hybrid.GetCapabilities(ctx),
	}
}

// initMetrics initializes OpenTelemetry metrics
func (s *Service) initMetrics() error {
	var err error

	s.compressionCounter, err = s.meter.Int64Counter(
		"compression.operations_total",
		metric.WithDescription("Total number of compression operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create compression counter: %w", err)
	}

	s.compressionTime, err = s.meter.Float64Histogram(
		"compression.duration_seconds",
		metric.WithDescription("Time spent on compression operations"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0),
	)
	if err != nil {
		return fmt.Errorf("failed to create compression time histogram: %w", err)
	}

	s.compressionRatio, err = s.meter.Float64Histogram(
		"compression.ratio",
		metric.WithDescription("Compression ratios achieved"),
		metric.WithUnit("1"),
		metric.WithExplicitBucketBoundaries(1.0, 1.5, 2.0, 3.0, 5.0, 10.0, 20.0),
	)
	if err != nil {
		return fmt.Errorf("failed to create compression ratio histogram: %w", err)
	}

	s.compressionQuality, err = s.meter.Float64Histogram(
		"compression.quality_score",
		metric.WithDescription("Quality scores of compression results"),
		metric.WithUnit("1"),
		metric.WithExplicitBucketBoundaries(0.0, 0.2, 0.4, 0.6, 0.8, 1.0),
	)
	if err != nil {
		return fmt.Errorf("failed to create compression quality histogram: %w", err)
	}

	s.compressionErrors, err = s.meter.Int64Counter(
		"compression.errors_total",
		metric.WithDescription("Total number of compression errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create compression errors counter: %w", err)
	}

	return nil
}
