# Memory Distiller Implementation

Replace the stub distiller with real memory consolidation that compresses, merges, and prioritizes memories based on usage patterns. Uses LLM to synthesize related memories into more valuable consolidated knowledge.

## Rationale
The distiller is currently a stub returning empty results. Memory consolidation prevents knowledge rot and reduces storage while improving retrieval relevance. Competitors like Letta struggle with memory retention (letta-1); a working distiller ensures ContextD memories remain valuable over time.

## User Stories
- As a developer, I want my memories to be automatically consolidated so that retrieval returns the most valuable insights
- As a long-term user, I want memory cleanup so that my knowledge base stays manageable
- As a power user, I want to trigger consolidation manually so that I can optimize before complex sessions

## Acceptance Criteria
- [ ] Distiller consolidates memories with >0.8 similarity into merged entries
- [ ] Original memories are preserved with link to consolidated version
- [ ] Confidence scores are updated based on consolidation
- [ ] Distiller can run automatically on schedule or manually via MCP tool
- [ ] Consolidated memories include source attribution
