// Package compression provides context compression algorithms for token optimization.
//
// The package implements three compression strategies: extractive (sentence selection),
// abstractive (AI-powered summarization), and hybrid (content-aware routing). These
// algorithms reduce token usage while preserving semantic meaning, enabling efficient
// context management for AI agents.
//
// # Security
//
// The package implements defense-in-depth security:
//   - API key protection for Anthropic Claude integration
//   - Content length validation (prevents DoS via oversized input)
//   - Input sanitization (empty content detection)
//   - Target ratio validation (must be > 1.0)
//   - Algorithm-specific size limits via GetCapabilities()
//
// # Algorithms
//
// Extractive Compression:
//   - Selects most important sentences from original content
//   - Fast and deterministic (no external API calls)
//   - Best for: Technical documentation, structured content, logs
//   - Max content length: 1MB (configurable)
//
// Abstractive Compression:
//   - Uses Claude Haiku to generate concise summaries
//   - Requires Anthropic API key (configure via Config.AnthropicAPIKey)
//   - Best for: Conversational text, narratives, mixed content
//   - Max content length: ~100K tokens (~400K characters)
//
// Hybrid Compression:
//   - Detects content type (code, prose, logs, structured data)
//   - Routes to extractive or abstractive based on analysis
//   - Best for: Unknown or mixed content types
//   - Combines strengths of both approaches
//
// # Usage
//
// Basic compression example:
//
//	cfg := compression.Config{
//	    DefaultAlgorithm: compression.AlgorithmExtractive,
//	    TargetRatio:      2.0,  // Compress to 50% of original size
//	    QualityThreshold: 0.7,  // Minimum acceptable quality
//	    AnthropicAPIKey:  os.Getenv("ANTHROPIC_API_KEY"),
//	}
//	svc, err := compression.NewService(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	result, err := svc.Compress(ctx, content, compression.AlgorithmExtractive, 2.0)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Compressed: %d → %d bytes (%.1fx)\n",
//	    result.Metadata.OriginalSize,
//	    result.Metadata.CompressedSize,
//	    result.Metadata.CompressionRatio)
//
// Check algorithm capabilities:
//
//	caps := svc.GetCapabilities(ctx)
//	for algo, cap := range caps {
//	    fmt.Printf("%s: max %d bytes, ratio support: %v\n",
//	        algo, cap.MaxContentLength, cap.SupportsTargetRatio)
//	}
//
// # Quality Scores
//
// All compression operations return a quality score (0.0 to 1.0):
//   - 1.0: Perfect preservation (no information loss)
//   - 0.8-0.9: High quality (minor details omitted)
//   - 0.6-0.7: Acceptable quality (key information preserved)
//   - < 0.6: Low quality (significant information loss)
//
// Quality scores are algorithm-specific:
//   - Extractive: Based on sentence coverage and semantic similarity
//   - Abstractive: Estimated from Claude's confidence and length ratio
//   - Hybrid: Weighted average of constituent algorithms
//
// # Observability
//
// The service exports OpenTelemetry metrics and traces:
//   - compression.operations_total (counter): Total operations by algorithm
//   - compression.duration_seconds (histogram): Processing time
//   - compression.ratio (histogram): Achieved compression ratios
//   - compression.quality_score (histogram): Quality score distribution
//   - compression.errors_total (counter): Error counts by type
//
// Traces include:
//   - Algorithm selection
//   - Content length and target ratio
//   - Processing time breakdown
//   - Compression metadata
//
// # Performance
//
// Algorithm benchmarks (approximate, hardware-dependent):
//   - Extractive: ~1MB/s (single-threaded)
//   - Abstractive: ~500ms per request (network latency + Claude API)
//   - Hybrid: Varies by routing decision
//
// Optimization strategies:
//   - Use extractive for latency-sensitive operations
//   - Batch abstractive requests when possible (future enhancement)
//   - Cache compression results for repeated content (caller responsibility)
//   - Monitor quality scores to tune target ratios
//
// # Content Type Detection
//
// The hybrid algorithm detects content types for intelligent routing:
//   - Code: High comment density, syntax patterns, file extensions
//   - Prose: Natural language, paragraph structure, narrative flow
//   - Logs: Timestamps, repeated patterns, structured format
//   - Structured: JSON, XML, tables, configuration files
//
// Detection heuristics are extensible via the ContentTypeDetector interface.
package compression
