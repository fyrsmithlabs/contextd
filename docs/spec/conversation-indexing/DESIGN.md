# Conversation Indexing Design

**Related Documents:**
- [SPEC.md](SPEC.md) - Requirements and success criteria
- [SCHEMA.md](SCHEMA.md) - Collection and document schemas
- [CONFIG.md](CONFIG.md) - Configuration reference

## Package Structure

```
internal/
├── conversation/
│   ├── parser.go          # Parse Claude Code JSONL files
│   ├── extractor.go       # Extract messages, decisions, metadata
│   ├── service.go         # Index, search, cross-reference operations
│   └── types.go           # Document types, options
│
├── extraction/
│   ├── heuristic.go       # Keyword-based decision detection
│   ├── llm.go             # LLM summarization via langchain-go
│   ├── provider.go        # Provider interface and factory
│   ├── tags.go            # Tag extraction logic
│   └── types.go           # Decision candidates, summaries
│
└── mcp/
    └── tools.go           # conversation_index, conversation_search
```

## Data Flow

```
Claude Code JSONL files (~/.claude/projects/{project}/*.jsonl)
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  Parser                                                      │
│  - Read JSONL lines                                         │
│  - Extract user/assistant messages                          │
│  - Extract tool calls and results                           │
│  - Filter noise (thinking blocks, raw tool output)          │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  Secret Scrubber (gitleaks)                                 │
│  - Scrub all text content                                   │
│  - Remove API keys, tokens, passwords                       │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  Extractor                                                   │
│  - Extract file references from tool calls                  │
│  - Extract commit SHAs from git operations                  │
│  - Extract tags from content and file types                 │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  Heuristic Decision Detector                                │
│  - Match decision patterns                                  │
│  - Score confidence by pattern weight                       │
│  - Mark candidates for LLM refinement if enabled            │
└─────────────────────────────────────────────────────────────┘
    │
    ▼ (if LLM enabled and confidence < threshold)
┌─────────────────────────────────────────────────────────────┐
│  LLM Summarizer (langchain-go)                              │
│  - Refine decision candidates                               │
│  - Extract summary, alternatives, reasoning                 │
│  - Add inferred tags                                        │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  Vectorstore (_conversations collection)                    │
│  - Embed and store documents                                │
│  - Index metadata for filtering                             │
└─────────────────────────────────────────────────────────────┘
```

## Interfaces

### ConversationParser

```go
type ConversationParser interface {
    // Parse reads a JSONL file and extracts messages
    Parse(path string) ([]RawMessage, error)

    // ParseAll reads all JSONL files in a directory
    ParseAll(dir string) (map[string][]RawMessage, error)
}

type RawMessage struct {
    SessionID  string
    UUID       string
    Timestamp  time.Time
    Role       string          // "user" or "assistant"
    Content    string
    ToolCalls  []ToolCall
    GitBranch  string
}

type ToolCall struct {
    Name   string
    Params map[string]string
    Result string
}
```

### DecisionExtractor

```go
type DecisionExtractor interface {
    // Extract finds decision candidates in messages
    Extract(messages []RawMessage) ([]DecisionCandidate, error)
}

type DecisionCandidate struct {
    SessionID      string
    MessageUUID    string
    Content        string
    Context        []string  // Surrounding messages
    PatternMatched string
    Confidence     float64
    NeedsLLMRefine bool
}
```

### Summarizer

```go
type Summarizer interface {
    // Summarize refines a decision candidate into a structured decision
    Summarize(ctx context.Context, candidate DecisionCandidate) (Decision, error)

    // Available returns true if the summarizer is configured and ready
    Available() bool
}

type Decision struct {
    Summary      string
    Alternatives []string
    Reasoning    string
    Tags         []string
    Confidence   float64
}
```

### ConversationService

```go
type ConversationService interface {
    // Index processes and stores conversations for a project
    Index(ctx context.Context, opts IndexOptions) (*IndexResult, error)

    // Search finds relevant conversations
    Search(ctx context.Context, opts SearchOptions) (*SearchResult, error)
}

type IndexOptions struct {
    ProjectPath string
    TenantID    string
    SessionIDs  []string // Empty = all sessions
    EnableLLM   bool
    Force       bool     // Reindex existing
}

type IndexResult struct {
    SessionsIndexed    int
    MessagesIndexed    int
    DecisionsExtracted int
    FilesReferenced    []string
    Errors             []error
}

type SearchOptions struct {
    Query       string
    ProjectPath string
    TenantID    string
    Types       []string // "message", "decision", "summary"
    Tags        []string
    FilePath    string
    Domain      string
    Limit       int
}
```

## Heuristic Decision Detection

### Default Patterns

```go
var DefaultDecisionPatterns = []Pattern{
    // Explicit decisions
    {Name: "lets_use",      Regex: `(?i)let's (go with|use|choose|pick)`, Weight: 0.9},
    {Name: "decided_to",    Regex: `(?i)decided to`,                      Weight: 0.9},
    {Name: "approach_is",   Regex: `(?i)the approach (is|will be)`,       Weight: 0.8},
    {Name: "choosing_over", Regex: `(?i)choosing .+ over`,                Weight: 0.9},

    // Architectural
    {Name: "architecture", Regex: `(?i)architecture.*(should|will)`, Weight: 0.7},
    {Name: "pattern_for",  Regex: `(?i)pattern for this`,            Weight: 0.7},

    // Anti-patterns
    {Name: "dont_because",    Regex: `(?i)don't (do|use).*because`, Weight: 0.8},
    {Name: "avoid_because",   Regex: `(?i)avoid.*because`,          Weight: 0.8},
    {Name: "failed_approach", Regex: `(?i)this (broke|failed)`,     Weight: 0.7},

    // Explicit capture
    {Name: "remember_this", Regex: `(?i)remember (this|that)`,     Weight: 1.0},
    {Name: "note_future",   Regex: `(?i)note for (future|later)`,  Weight: 1.0},
}
```

### Confidence Thresholds

| Confidence | Action |
|------------|--------|
| >= 0.8 | Index as decision directly |
| 0.5 - 0.8 | Index as candidate; refine with LLM if enabled |
| < 0.5 | Skip (too noisy) |

## Tag Extraction

### Default Tag Rules

```go
var DefaultTagRules = map[string][]string{
    // Languages
    "golang":     {".go", "go mod", "go build", "go test"},
    "python":     {".py", "pip", "pytest", "python"},
    "typescript": {".ts", ".tsx", "npm", "yarn", "node"},
    "rust":       {".rs", "cargo", "rustc"},

    // Infrastructure
    "kubernetes": {"kubectl", "k8s", "helm", "deployment.yaml", "service.yaml"},
    "terraform":  {".tf", "terraform", "tfstate", "tfvars"},
    "docker":     {"Dockerfile", "docker-compose", "container", "image"},
    "aws":        {"aws", "s3", "ec2", "lambda", "cloudformation"},

    // Activities
    "debugging":     {"fix", "bug", "error", "issue", "broken", "failing"},
    "documentation": {"docs", "readme", "comment", "explain", "document"},
    "testing":       {"test", "spec", "coverage", "mock", "assert"},
    "refactoring":   {"refactor", "cleanup", "rename", "extract", "simplify"},
    "security":      {"auth", "secret", "credential", "permission", "encrypt"},
    "performance":   {"optimize", "slow", "fast", "cache", "latency"},
}
```

## LLM Integration

### Provider Factory

```go
func NewSummarizer(cfg ExtractionConfig) (Summarizer, error) {
    if !cfg.Enabled || cfg.Provider == "disabled" {
        return &NoOpSummarizer{}, nil
    }

    providerCfg, ok := cfg.Providers[cfg.Provider]
    if !ok {
        return nil, fmt.Errorf("provider %q not configured", cfg.Provider)
    }

    switch cfg.Provider {
    case "anthropic":
        return newAnthropicSummarizer(providerCfg)
    case "openai":
        return newOpenAISummarizer(providerCfg)
    default:
        return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
    }
}
```

### Summarization Prompt

```
Extract the decision from this conversation snippet.

Context (preceding messages):
{{range .Context}}
{{.Role}}: {{.Content}}
{{end}}

Decision message:
{{.Content}}

Respond in JSON:
{
  "summary": "one-line decision summary",
  "alternatives_considered": ["alt1", "alt2"],
  "reasoning": "why this option was chosen",
  "tags": ["relevant", "context", "tags"]
}

Extract only what the text explicitly states. Do not infer or add information.
```

## Cross-Reference Extraction

### File References

Extract from tool calls:
- `Read` tool → file was read
- `Edit` tool → file was edited
- `Write` tool → file was created
- `Glob`/`Grep` results → files were searched

### Commit References

Extract from Bash tool results:
- Parse `git commit` output for SHA
- Parse `git log` for recent commits in session
- Link commits to conversation by timestamp proximity
