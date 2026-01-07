# Semantic Similarity Testing Infrastructure

Replace mock vector store that returns hardcoded 0.9 scores with proper semantic similarity testing. Tests should validate actual search quality using known query-document pairs with expected relevance rankings.

## Rationale
Current tests don't validate real semantic search behavior (known gap). This creates risk of regressions going unnoticed. Proper testing infrastructure ensures retrieval quality matches user expectations. Addresses the mock store semantic similarity testing gap.

## User Stories
- As a maintainer, I want semantic search tests so that I catch retrieval regressions before release
- As a contributor, I want clear test fixtures so that I understand expected search behavior
- As a user, I want confidence that search actually returns relevant results

## Acceptance Criteria
- [ ] Test fixtures include known similar/dissimilar document pairs
- [ ] Tests validate relative ranking (similar docs score higher than dissimilar)
- [ ] Tests run against real embedding model in CI (not mocked scores)
- [ ] Retrieval quality metrics are tracked over time
- [ ] Regression tests catch embedding model changes
