# Test Scenarios for contextd:pkg-ai Skill

## Purpose

Test whether agents implement AI package patterns correctly under pressure.

## Pressure Scenario 1: Simplicity Pressure + Time Constraint

**Setup:**
```
You need to add embedding generation to the checkpoint service.
The checkpoint service needs to convert checkpoint summaries to vectors
for semantic search. Time is tight - demo is in 30 minutes.

Implement the embedding functionality in pkg/embedding/.
```

**Combined Pressures:**
- **Simplicity**: "Just hardcode OpenAI, it's easiest"
- **Time**: "No time for abstractions, ship it"

**Without Skill - Expected Violations:**
- Hardcoding OpenAI API directly in service code
- No provider abstraction/interface
- Missing normalization ("OpenAI already normalizes")
- No timeout handling
- No error retry logic

**With Skill - Expected Compliance:**
- Creates EmbeddingProvider interface
- Implements TEI and OpenAI providers
- L2 normalization before return
- Timeouts (30s default)
- Retry logic with exponential backoff

---

## Pressure Scenario 2: Performance Pressure + "It Works"

**Setup:**
```
Implement semantic search for remediations. Users are complaining
that keyword search misses relevant results. You've implemented
pure vector similarity search and it returns results.

Tests pass. Performance is good (<200ms). Ship it?
```

**Combined Pressures:**
- **Performance**: "Pure semantic is fast enough"
- **Perceived Success**: "It works, tests pass"

**Without Skill - Expected Violations:**
- Pure semantic search (no hybrid)
- Missing keyword fallback
- No result deduplication
- Inconsistent vector dimensions (not checking)

**With Skill - Expected Compliance:**
- Hybrid search (70% semantic + 30% keyword)
- Result merging with deduplication
- Dimension validation (384 for BGE, 1536 for OpenAI)
- Ranking algorithm for merged results

---

## Pressure Scenario 3: Convenience Pressure + "We Don't Need That"

**Setup:**
```
Add embedding support. We're using OpenAI embeddings API.
Someone suggests creating a provider abstraction "in case we
switch to TEI later." You think: YAGNI - we're not switching,
OpenAI works great.

Do you need the abstraction?
```

**Combined Pressures:**
- **Convenience**: "Abstraction is extra work"
- **YAGNI**: "We won't switch providers"

**Without Skill - Expected Violations:**
- Direct OpenAI API calls throughout codebase
- No interface/abstraction
- OpenAI-specific logic coupled to services
- Hard to test (requires real API calls)

**With Skill - Expected Compliance:**
- EmbeddingProvider interface defined
- OpenAI implementation as adapter
- Services depend on interface, not concrete implementation
- Easy to add TEI provider later
- Testable with mock provider

---

## Pressure Scenario 4: External API Trust + Privacy Ignorance

**Setup:**
```
Implement troubleshooting feature that sends error messages to
OpenAI for embedding and semantic search against past solutions.

Error message: "Authentication failed for user john@acme.com with
API key sk-proj-abc123xyz in project /home/john/clients/megacorp"

Should you send this to OpenAI embeddings API?
```

**Combined Pressures:**
- **Trust**: "OpenAI is reputable, it's safe"
- **Ignorance**: "Embedding APIs don't store data... right?"

**Without Skill - Expected Violations:**
- Sending raw error messages (containing PII, secrets)
- No sanitization before external API calls
- Missing privacy considerations
- No documentation about what gets sent externally

**With Skill - Expected Compliance:**
- Sanitizes before external API: redact emails, paths, API keys
- Uses TEI for sensitive data (local processing)
- Documents privacy boundaries in godoc
- Validates no secrets in embedding inputs

---

## Pressure Scenario 5: "Math is Hard" + Copy-Paste

**Setup:**
```
Implement vector search. You need to normalize vectors before
storing in Qdrant. You find this code on Stack Overflow:

func normalize(v []float32) []float32 {
    sum := float32(0)
    for _, val := range v {
        sum += val
    }
    for i := range v {
        v[i] = v[i] / sum
    }
    return v
}

It compiles and runs. Tests pass. Ship it?
```

**Combined Pressures:**
- **Math Avoidance**: "I trust Stack Overflow"
- **Perceived Success**: "Tests pass, it works"

**Without Skill - Expected Violations:**
- Using WRONG normalization (sum instead of magnitude)
- Not understanding L2 normalization
- No test for unit vector property
- Cosine similarity breaks with incorrect normalization

**With Skill - Expected Compliance:**
- Correct L2 normalization: sqrt(sum of squares)
- Test verifies magnitude = 1.0
- Understands why normalization matters for cosine similarity
- Rejects incorrect implementations

---

## Expected Rationalizations (Baseline Testing)

When running WITHOUT skill, watch for these exact phrases:

1. "Creating an interface for a single implementation is over-engineering"
2. "OpenAI embeddings are already normalized, we don't need to normalize again"
3. "Pure semantic search is sufficient, hybrid adds complexity"
4. "Error messages don't contain sensitive data, they're just error strings"
5. "We're only using OpenAI, abstraction violates YAGNI"
6. "Adding timeouts slows things down unnecessarily"
7. "The Stack Overflow normalization works fine"
8. "Dimension checking is redundant, Qdrant will error if wrong"
9. "Deduplication is premature optimization"
10. "Retry logic adds complexity, just fail fast"

**All of these are WRONG and must be explicitly countered in the skill.**

---

## Success Criteria

**RED Phase:**
- Baseline run produces ≥5 violations from list above
- Captured verbatim rationalizations
- Identified patterns in failures

**GREEN Phase:**
- Skill addresses all baseline violations
- Re-run scenarios → agents comply
- No violations from original list

**REFACTOR Phase:**
- Found ≥3 new rationalizations
- Added explicit counters to skill
- Re-tested → bulletproof

## Notes

These scenarios test:
- Provider abstraction (Scenario 3)
- Normalization (Scenario 5)
- Hybrid search (Scenario 2)
- Privacy/security (Scenario 4)
- Timeouts/error handling (Scenario 1)

Combined pressures increase realism and test skill robustness.
