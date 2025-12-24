# Conversation Indexing Specification

**Feature**: Conversation Indexing
**Status**: Draft
**Created**: 2025-12-09

## Overview

Conversation Indexing extracts and indexes past Claude Code sessions for semantic search. Agents search previous discussions, find decisions, and recover context from old sessions.

**Related Documents:**
- [DESIGN.md](DESIGN.md) - Architecture and components
- [SCHEMA.md](SCHEMA.md) - Collection and document schemas
- [CONFIG.md](CONFIG.md) - Configuration reference

## User Scenarios

### P1: Search Past Discussions

**Story**: As an agent resuming work on a feature, I search past conversations to find prior decisions and context.

**Acceptance Criteria**:
```gherkin
Given indexed conversations for the project
When the agent calls conversation_search with "authentication approach"
Then results include relevant conversation segments
And results show session timestamps for context
And results include files discussed in those conversations
```

### P1: Index Conversations on First Use

**Story**: As a user starting a new project session, I index my past Claude Code conversations to make them searchable.

**Acceptance Criteria**:
```gherkin
Given a project with 10+ unindexed conversation files
When the agent detects unindexed conversations at session start
Then the agent asks "Index past conversations for searchable history?"
And if confirmed, indexes all conversation files
And scrubs all content through secret scrubber before storage
```

### P2: Find Decisions About Specific Files

**Story**: As an agent modifying a file, I find past conversations that discussed this file to understand prior decisions.

**Acceptance Criteria**:
```gherkin
Given indexed conversations mentioning "internal/mcp/server.go"
When the agent calls conversation_search with file_path filter
Then results include only conversations that touched that file
And results indicate whether file was read, edited, or created
```

### P2: Extract and Store Decisions

**Story**: As a system indexing conversations, I extract architectural decisions and store them for retrieval.

**Acceptance Criteria**:
```gherkin
Given a conversation containing "decided to use chromem over Qdrant"
When the heuristic extractor processes the conversation
Then it identifies this as a decision candidate
And stores it with tags ["architecture", "vectorstore"]
And links it to affected files
```

### P3: LLM-Enhanced Decision Extraction

**Story**: As a user with LLM extraction enabled, I get cleaner decision summaries from my conversations.

**Acceptance Criteria**:
```gherkin
Given LLM extraction enabled with Anthropic provider
And a decision candidate with confidence < 0.8
When the LLM summarizer processes the candidate
Then it produces a structured decision with summary, alternatives, reasoning
And stores the refined decision in the conversations collection
```

## Functional Requirements

### FR-001: JSONL Parsing
The system parses Claude Code JSONL conversation files and extracts user messages, assistant responses, and tool call metadata.

### FR-002: Secret Scrubbing
The system scrubs all conversation content through gitleaks before storage. No secrets persist in the vector store.

### FR-003: Conversation Collection
The system stores conversations in a dedicated `{tenant}_{project}_conversations` collection, separate from codebase indexing.

### FR-004: Semantic Search
The system retrieves conversations by semantic similarity, filtered by type, tags, file path, or domain.

### FR-005: Tag Extraction
The system extracts context tags from conversation content:
- Language tags from file extensions discussed
- Domain tags from keywords (kubernetes, terraform, docker)
- Activity tags from patterns (debugging, testing, documentation)

### FR-006: File Cross-References
The system extracts file references from tool calls and stores them with conversations. Users query "conversations about this file."

### FR-007: Commit Cross-References
The system extracts commit SHAs from git operations and links them to conversations. Users query "what was discussed when this commit was made."

### FR-008: Heuristic Decision Detection
The system identifies decision candidates using keyword patterns with configurable weights. High-confidence matches index directly; lower-confidence candidates optionally refine through LLM.

### FR-009: LLM Summarization (Optional)
The system supports optional LLM-based decision refinement via langchain-go. Users configure Anthropic or OpenAI providers. OpenAI provider connects to local Ollama when configured.

### FR-010: Templated Configuration
The system processes configuration through Go templates, interpolating environment variables for API keys and URLs.

### FR-011: MCP Tools
The system exposes `conversation_index` and `conversation_search` as MCP tools.

### FR-012: CLI Commands (Phase 2)
The system provides `ctxd conversations index` and `ctxd conversations search` CLI commands.

## Success Criteria

### SC-001: Search Relevance
80% of conversation search results rated "relevant" when users provide feedback.

### SC-002: Decision Extraction Precision
70% of extracted decisions represent actual architectural or technical decisions (not routine actions).

### SC-003: Indexing Coverage
System indexes 95%+ of conversation messages (excluding empty files and parse errors).

### SC-004: Secret Scrubbing
Zero secrets detected in stored conversation content when audited.

### SC-005: Cross-Reference Accuracy
90%+ of file references correctly link to actual files discussed in conversations.

## Implementation Phases

### Phase 1: Core Indexing
- JSONL parser
- Secret scrubbing integration
- `_conversations` collection
- `conversation_index` MCP tool
- `conversation_search` MCP tool
- Heuristic decision detection
- Tag extraction
- File/commit cross-references

### Phase 2: LLM + CLI
- langchain-go integration
- Provider configs (Anthropic, OpenAI)
- Templated config
- CLI commands
- Configurable patterns and tags

### Phase 3: ReasoningBank Integration
- Decision to memory distillation
- Smart filtering
- Marketplace skill updates

### Phase 4: Polish
- Session piggyback prompts
- First-use detection
- Bidirectional search
- Performance optimization
