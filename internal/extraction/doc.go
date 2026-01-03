// Package extraction provides decision extraction capabilities from conversation
// messages using heuristic pattern matching.
//
// The package supports:
//   - Heuristic-based decision detection using configurable regex patterns
//   - Confidence scoring for extracted decisions
//   - Context window building from surrounding messages
//   - Extensible pattern configuration
//
// # Architecture
//
// The main components are:
//   - HeuristicExtractor: Pattern-based decision extraction
//   - Pattern: Configurable regex patterns with weights and tags
//   - DecisionCandidate: Represents a potential decision with confidence score
//
// # Usage
//
// Create a heuristic extractor with default patterns:
//
//	extractor, err := extraction.NewHeuristicExtractor(extraction.ExtractionConfig{
//	    ConfidenceThreshold: 0.5,
//	    LLMRefineThreshold:  0.8,
//	})
//
// Extract decisions from messages:
//
//	candidates, err := extractor.Extract(messages)
//	for _, c := range candidates {
//	    fmt.Printf("Decision: %s (confidence: %.2f)\n", c.PatternMatched, c.Confidence)
//	}
//
// # Pattern Configuration
//
// Patterns can be customized via ExtractionConfig.Patterns. Each pattern has:
//   - Regex: The pattern to match
//   - Weight: Confidence score (0.0-1.0) when matched
//   - Name: Human-readable pattern name
//   - Tags: Categories for grouping (e.g., "architecture", "refactoring")
//
// Default patterns detect common decision indicators like "I decided to",
// "The approach I'll take", "After considering", etc.
//
// # LLM Refinement
//
// The NeedsLLMRefine field on DecisionCandidate indicates whether the decision
// should be refined by an LLM for higher accuracy. This is set when the
// confidence score is above ConfidenceThreshold but below LLMRefineThreshold.
//
// Note: LLM-based refinement is not yet implemented. The field is reserved
// for future enhancement.
package extraction
