# Interface Migration to Shared Packages

Move interface definitions from internal/remediation/interfaces.go to shared packages (internal/vectorstore, internal/embeddings) as noted in the TODO comments. Improves code organization and enables reuse.

## Rationale
Current interface locations violate package boundaries (TODOs in code). This technical debt makes the codebase harder to maintain and understand. Clean architecture enables faster feature development. Addresses technical debt from code TODOs.

## User Stories
- As a contributor, I want interfaces in logical locations so that I can find and understand them
- As a maintainer, I want clean package boundaries so that changes don't cause unexpected side effects

## Acceptance Criteria
- [ ] VectorStore interface lives in internal/vectorstore package
- [ ] EmbeddingProvider interface lives in internal/embeddings package
- [ ] All imports are updated across the codebase
- [ ] No duplicate interface definitions remain
- [ ] Backward compatibility is maintained
