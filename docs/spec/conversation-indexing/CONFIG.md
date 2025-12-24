# Conversation Indexing Configuration

**Related Documents:**
- [SPEC.md](SPEC.md) - Requirements and success criteria
- [DESIGN.md](DESIGN.md) - Architecture and components
- [SCHEMA.md](SCHEMA.md) - Collection and document schemas

## Configuration File

Location: `~/.config/contextd/config.yaml`

## Template Processing

Configuration files support Go templates for dynamic values. Templates process before YAML parsing.

### Template Functions

| Function | Description | Example |
|----------|-------------|---------|
| `env` | Read environment variable | `{{ env "API_KEY" }}` |
| `default` | Provide fallback value | `{{ env "URL" \| default "http://localhost" }}` |
| `file` | Read file contents | `{{ file "/run/secrets/key" }}` |
| `required` | Fail if empty | `{{ env "KEY" \| required "KEY" }}` |

## Full Configuration Reference

```yaml
# Conversation indexing configuration
conversation:
  # Enable conversation indexing feature
  enabled: true

  # Path pattern for Claude Code conversations
  # Default: ~/.claude/projects/{project}/*.jsonl
  conversations_path: "{{ env \"CLAUDE_CONVERSATIONS_PATH\" | default \"~/.claude/projects\" }}"

# Decision extraction configuration
extraction:
  # Enable LLM-based extraction
  enabled: false

  # Active provider: "anthropic", "openai", or "disabled"
  provider: disabled

  # Confidence threshold for LLM refinement
  # Candidates below this threshold refine through LLM when enabled
  llm_threshold: 0.8

  # Minimum confidence to index (skip below this)
  min_confidence: 0.5

  # Provider-specific configurations
  providers:
    anthropic:
      api_key: "{{ env \"ANTHROPIC_API_KEY\" }}"
      model: claude-3-haiku-20240307
      max_tokens: 256
      timeout: 30s

    openai:
      api_key: "{{ env \"OPENAI_API_KEY\" }}"
      model: gpt-4o-mini
      max_tokens: 256
      base_url: "{{ env \"OPENAI_BASE_URL\" | default \"\" }}"
      timeout: 30s

# Tag extraction configuration
tags:
  # Use default tag rules
  use_defaults: true

  # Additional custom tag rules (appended to defaults)
  # Format: tag_name: [patterns...]
  custom: {}
  #  custom:
  #    myframework: ["myframework", ".myf"]
  #    internal-tool: ["internal-tool", "itool"]

  # Override default rules entirely (replaces defaults)
  # Only used if use_defaults: false
  overrides: {}

# Decision pattern configuration
patterns:
  # Use default decision patterns
  use_defaults: true

  # Additional custom patterns (appended to defaults)
  custom: []
  #  custom:
  #    - name: team_decision
  #      regex: "(?i)team decided"
  #      weight: 0.85

  # Override default patterns entirely (replaces defaults)
  # Only used if use_defaults: false
  overrides: []
```

## Minimal Configuration

For basic usage without LLM extraction:

```yaml
conversation:
  enabled: true

extraction:
  enabled: false
```

## LLM Extraction with Anthropic

```yaml
conversation:
  enabled: true

extraction:
  enabled: true
  provider: anthropic
  providers:
    anthropic:
      api_key: "{{ env \"ANTHROPIC_API_KEY\" | required \"ANTHROPIC_API_KEY\" }}"
      model: claude-3-haiku-20240307
```

## LLM Extraction with Local Ollama

Use OpenAI provider pointing to Ollama's OpenAI-compatible endpoint:

```yaml
conversation:
  enabled: true

extraction:
  enabled: true
  provider: openai
  providers:
    openai:
      api_key: ""  # Not needed for local Ollama
      model: llama3.2:3b
      base_url: "{{ env \"OLLAMA_URL\" | default \"http://localhost:11434/v1\" }}"
```

## Custom Tags

Add project-specific tags:

```yaml
tags:
  use_defaults: true
  custom:
    our-service: ["our-service", "oursvc", "our_service"]
    legacy-api: ["legacy", "v1-api", "deprecated"]
```

Replace default tags entirely:

```yaml
tags:
  use_defaults: false
  overrides:
    golang: [".go"]
    kubernetes: ["kubectl", "k8s"]
    # Only these tags will be used
```

## Custom Decision Patterns

Add patterns for your team's conventions:

```yaml
patterns:
  use_defaults: true
  custom:
    - name: adr_reference
      regex: "(?i)see ADR-\\d+"
      weight: 0.95
    - name: team_consensus
      regex: "(?i)team agreed"
      weight: 0.9
```

## Environment Variables

| Variable | Description | Used By |
|----------|-------------|---------|
| `ANTHROPIC_API_KEY` | Anthropic API key | anthropic provider |
| `OPENAI_API_KEY` | OpenAI API key | openai provider |
| `OPENAI_BASE_URL` | OpenAI API base URL | openai provider (Ollama, Azure) |
| `OLLAMA_URL` | Ollama server URL | openai provider (local) |
| `CLAUDE_CONVERSATIONS_PATH` | Override conversations location | conversation.conversations_path |

## Config Struct Reference

```go
type ConversationConfig struct {
    Enabled           bool   `koanf:"enabled"`
    ConversationsPath string `koanf:"conversations_path"`
}

type ExtractionConfig struct {
    Enabled      bool                      `koanf:"enabled"`
    Provider     string                    `koanf:"provider"`
    LLMThreshold float64                   `koanf:"llm_threshold"`
    MinConfidence float64                  `koanf:"min_confidence"`
    Providers    map[string]ProviderConfig `koanf:"providers"`
}

type ProviderConfig struct {
    APIKey    string        `koanf:"api_key"`
    Model     string        `koanf:"model"`
    MaxTokens int           `koanf:"max_tokens"`
    BaseURL   string        `koanf:"base_url"`
    Timeout   time.Duration `koanf:"timeout"`
}

type TagsConfig struct {
    UseDefaults bool                `koanf:"use_defaults"`
    Custom      map[string][]string `koanf:"custom"`
    Overrides   map[string][]string `koanf:"overrides"`
}

type PatternsConfig struct {
    UseDefaults bool            `koanf:"use_defaults"`
    Custom      []PatternConfig `koanf:"custom"`
    Overrides   []PatternConfig `koanf:"overrides"`
}

type PatternConfig struct {
    Name   string  `koanf:"name"`
    Regex  string  `koanf:"regex"`
    Weight float64 `koanf:"weight"`
}
```
