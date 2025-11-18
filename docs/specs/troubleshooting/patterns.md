# Common Error Patterns

**Parent**: [../SPEC.md](../SPEC.md)

## 5-Step Troubleshooting Process

The service implements a comprehensive 5-step process based on industry best practices:

```
┌─────────────────────────────────────────────────────────────────┐
│ Step 1: Symptom Collection                                      │
│ - Error message, stack trace, context                           │
│ - Environment metadata (file, line, language)                   │
│ - Mode selection (auto, interactive, guided)                    │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ Step 2: Pattern Recognition (Semantic Search)                   │
│ - Generate embedding for error message                          │
│ - Vector similarity search in knowledge base                    │
│ - Filter by category, tags (optional)                           │
│ - Rerank by hybrid score (semantic + success + usage)           │
│ - Return top N similar issues (default: 5)                      │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ Step 3: Hypothesis Formation                                    │
│ - Extract root causes from similar issues                       │
│ - Calculate probability: match_score * success_rate             │
│ - Aggregate evidence for recurring patterns                     │
│ - Generate verification steps for each hypothesis               │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ Step 4: Hypothesis Ranking                                      │
│ - Normalize probabilities (sum to 1.0)                          │
│ - Sort by probability (descending)                              │
│ - Select top hypothesis as most likely root cause               │
│ - Determine confidence level (high: ≥0.8, medium: ≥0.5)         │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ Step 5: Action Generation & Resolution                          │
│ - Extract verification steps from top hypothesis                │
│ - Extract solution steps from best matching issue               │
│ - Detect destructive operations (delete, remove, kill, etc.)    │
│ - Add safety warnings for destructive steps                     │
│ - Return recommended actions with expected outcomes             │
└─────────────────────────────────────────────────────────────────┘
```

## Confidence Levels

The diagnosis assigns confidence levels based on match quality:

| Confidence | Probability | Meaning | Behavior |
|-----------|-------------|---------|----------|
| **High** | ≥ 0.8 | Strong match, high success rate | Include detailed timeline, affected resources, solution directly actionable |
| **Medium** | 0.5 - 0.79 | Moderate match, some uncertainty | Include general recommendations, suggest verification steps |
| **Low** | < 0.5 | Weak match or novel issue | Recommend manual investigation, external documentation search |

## Progressive Disclosure

The service returns information based on confidence level to avoid overwhelming users with low-quality data:

### High Confidence (≥0.8)

- Root cause with evidence
- Detailed timeline of events
- Affected resources list
- Direct solution steps
- Expected outcomes for each step

### Medium Confidence (0.5-0.79)

- Root cause with caveats
- Verification steps to confirm hypothesis
- General diagnostic guidance
- Similar issues for reference

### Low Confidence (<0.5)

- Generic troubleshooting guidance
- Recommendation to search external docs
- Suggestion to create new knowledge entry after resolution

## Pattern Retrieval Process

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Generate Query Embedding                                 │
│    - Input: error_message                                   │
│    - Output: []float32 (1536 dimensions)                    │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│ 2. Build Search Query                                       │
│    - Vector: query embedding                                │
│    - TopK: limit * 2 (for reranking)                       │
│    - Filter: category AND tags (if provided)               │
│    - Example: `category == "network" and tags like "%dns%"`│
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│ 3. Execute Vector Search                                    │
│    - Database: "shared"                                     │
│    - Collection: "troubleshooting_knowledge"                │
│    - Search with IVF index (nprobe=128)                     │
│    - Returns: Top 2N results by vector similarity           │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│ 4. Calculate Hybrid Scores                                  │
│    - Semantic: 1.0 / (1.0 + distance) * 0.6                │
│    - Success Rate: success_rate * 0.3                       │
│    - Usage: (usage_count / 100) * 0.1                       │
│    - Total: semantic + success_rate + usage                 │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│ 5. Rerank and Filter                                        │
│    - Sort by hybrid score (descending)                      │
│    - Take top N results                                     │
│    - Enrich with safety information                         │
│    - Add confidence levels                                  │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
               [Ranked Results]
```

## Safety Detection

### Destructive Operation Keywords

Keywords that trigger destructive flag and safety warnings:
- `delete`, `remove`, `drop`, `destroy`
- `restart`, `kill`, `terminate`
- `wipe`, `format`, `reset`

### Safety Warning Template

```
CAUTION: This action may cause service disruption. Confirm before proceeding.
```

### Detection Logic

```go
func isDestructive(text string) bool {
    destructiveKeywords := []string{
        "delete", "remove", "drop", "destroy",
        "restart", "kill", "terminate",
        "wipe", "format", "reset",
    }

    lowerText := strings.ToLower(text)
    for _, keyword := range destructiveKeywords {
        if strings.Contains(lowerText, keyword) {
            return true
        }
    }
    return false
}
```

## Confidence Determination

```go
func determineConfidence(matchScore float64) string {
    if matchScore >= 0.8 {
        return ConfidenceHigh
    } else if matchScore >= 0.5 {
        return ConfidenceMedium
    }
    return ConfidenceLow
}
```
