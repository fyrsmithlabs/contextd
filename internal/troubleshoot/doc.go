// Package troubleshoot provides AI-powered error diagnosis and pattern recognition.
//
// The package analyzes error messages using AI and semantic pattern matching to provide
// root cause analysis, hypotheses, and remediation suggestions. Error patterns are stored
// in a vector database for team-wide knowledge sharing and continuous improvement.
//
// # Security
//
// The package implements defense-in-depth security:
//   - Input validation for all error messages and patterns
//   - Confidence score validation (0.0-1.0 range)
//   - Multi-tenant isolation via vector store filtering
//   - AI response sanitization and JSON parsing validation
//   - Pattern metadata validation before storage
//
// # Usage
//
// Basic diagnosis example:
//
//	svc, err := troubleshoot.NewService(vectorStore, logger, aiClient)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	diagnosis, err := svc.Diagnose(ctx,
//	    "panic: runtime error: invalid memory address",
//	    "occurred during user authentication",
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Root cause: %s\n", diagnosis.RootCause)
//	for _, rec := range diagnosis.Recommendations {
//	    fmt.Printf("- %s\n", rec)
//	}
//
// Storing error patterns:
//
//	pattern := &troubleshoot.Pattern{
//	    ErrorType:   "NullPointerException",
//	    Description: "Null pointer access in authentication flow",
//	    Solution:    "Add nil check before accessing user object",
//	    Confidence:  0.9,
//	    Frequency:   5,
//	}
//	err := svc.SavePattern(ctx, pattern)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Diagnosis Process
//
// The service follows a multi-stage diagnosis process:
//
// 1. Pattern Matching - Search vector database for similar error patterns
// 2. High-Confidence Check - If pattern match >0.8 confidence, return immediately
// 3. AI Hypothesis Generation - Query AI client for root cause analysis (if configured)
// 4. Result Synthesis - Combine pattern matches with AI hypotheses
// 5. Confidence Scoring - Calculate overall diagnosis confidence
//
// Pattern-based diagnosis is always attempted first for speed and cost efficiency.
// AI diagnosis is only used when patterns don't provide high-confidence matches.
//
// # AI Client
//
// The AI client is optional. If not provided during service creation, the service
// will operate in pattern-only mode. This is useful for:
//   - Testing environments
//   - Cost-sensitive deployments
//   - Offline operation
//
// When AI is available, it enhances diagnosis with:
//   - Root cause identification
//   - Multiple hypotheses with likelihood scores
//   - Step-by-step remediation recommendations
//
// # Performance
//
// Current implementation uses semantic search for pattern matching (100-200ms typical).
// AI diagnosis adds 1-3 seconds depending on the AI provider and model.
//
// Optimization strategies:
//   - High-confidence pattern matches bypass AI (10x faster)
//   - Pattern search limited to top 5 results
//   - AI prompt includes only top 3 patterns
//   - Vector search with filters for efficient pattern retrieval
//
// # Pattern Storage
//
// Patterns are stored with embeddings for semantic search:
//   - Content: "{error_type}: {description}" for embedding
//   - Metadata: error_type, description, solution, confidence, frequency, created_at
//   - Unique ID: Generated if not provided (pattern_{uuid})
//   - Timestamps: Auto-set if not provided
//   - Default confidence: 0.5 if not specified
//
// Patterns support incremental learning via frequency tracking.
package troubleshoot
