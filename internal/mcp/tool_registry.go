package mcp

import (
	"regexp"
	"strings"
	"sync"
)

// ToolCategory represents the functional category of a tool.
type ToolCategory string

const (
	// CategoryMemory is for memory/ReasoningBank tools.
	CategoryMemory ToolCategory = "memory"
	// CategoryCheckpoint is for checkpoint tools.
	CategoryCheckpoint ToolCategory = "checkpoint"
	// CategoryRemediation is for remediation tools.
	CategoryRemediation ToolCategory = "remediation"
	// CategoryRepository is for repository/search tools.
	CategoryRepository ToolCategory = "repository"
	// CategoryTroubleshoot is for troubleshooting/diagnostic tools.
	CategoryTroubleshoot ToolCategory = "troubleshoot"
	// CategoryFolding is for context-folding branch tools.
	CategoryFolding ToolCategory = "folding"
	// CategoryConversation is for conversation indexing/search tools.
	CategoryConversation ToolCategory = "conversation"
	// CategoryReflection is for reflection/analysis tools.
	CategoryReflection ToolCategory = "reflection"
	// CategorySearch is for tool discovery (tool_search itself).
	CategorySearch ToolCategory = "search"
)

// ToolMetadata contains metadata about a registered MCP tool.
type ToolMetadata struct {
	// Name is the unique tool name (e.g., "memory_search").
	Name string `json:"name"`

	// Description is a human-readable description of what the tool does.
	Description string `json:"description"`

	// Category is the functional category of the tool.
	Category ToolCategory `json:"category"`

	// DeferLoading indicates if this tool should be defer-loaded.
	// When true, the tool definition is not sent to the client initially,
	// and the client must discover it via tool_search.
	DeferLoading bool `json:"defer_loading"`

	// Keywords are additional searchable terms for this tool.
	Keywords []string `json:"keywords,omitempty"`
}

// ToolRegistry manages metadata about all registered MCP tools.
// It enables tool discovery via search rather than loading all tools upfront.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*ToolMetadata
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*ToolMetadata),
	}
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(tool *ToolMetadata) {
	if tool == nil || tool.Name == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name] = tool
}

// RegisterAll adds multiple tools to the registry.
func (r *ToolRegistry) RegisterAll(tools []*ToolMetadata) {
	for _, tool := range tools {
		r.Register(tool)
	}
}

// Get returns the metadata for a specific tool.
func (r *ToolRegistry) Get(name string) (*ToolMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tool metadata.
func (r *ToolRegistry) List() []*ToolMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ToolMetadata, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

// ListNames returns all registered tool names.
func (r *ToolRegistry) ListNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]string, 0, len(r.tools))
	for name := range r.tools {
		result = append(result, name)
	}
	return result
}

// ListByCategory returns all tools in a specific category.
func (r *ToolRegistry) ListByCategory(category ToolCategory) []*ToolMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ToolMetadata, 0)
	for _, tool := range r.tools {
		if tool.Category == category {
			result = append(result, tool)
		}
	}
	return result
}

// ListNonDeferred returns tools that should be loaded immediately (not deferred).
func (r *ToolRegistry) ListNonDeferred() []*ToolMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ToolMetadata, 0)
	for _, tool := range r.tools {
		if !tool.DeferLoading {
			result = append(result, tool)
		}
	}
	return result
}

// ListDeferred returns tools that should be defer-loaded.
func (r *ToolRegistry) ListDeferred() []*ToolMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ToolMetadata, 0)
	for _, tool := range r.tools {
		if tool.DeferLoading {
			result = append(result, tool)
		}
	}
	return result
}

// SearchResult contains a tool match from a search query.
type SearchResult struct {
	// Tool is the matched tool metadata.
	Tool *ToolMetadata `json:"tool"`

	// Score indicates match quality (higher is better).
	// 3 = exact name match
	// 2 = name contains query
	// 1 = description/keywords match
	Score int `json:"score"`

	// MatchReason describes why this tool matched.
	MatchReason string `json:"match_reason"`
}

// Search finds tools matching the query string.
// Uses case-insensitive matching against tool names, descriptions, and keywords.
// Supports regex patterns (similar to Anthropic's regex variant for tool search).
func (r *ToolRegistry) Search(query string) []*SearchResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if query == "" {
		return nil
	}

	queryLower := strings.ToLower(query)
	var results []*SearchResult

	// Try to compile as regex, fall back to literal matching if invalid
	var regex *regexp.Regexp
	if re, err := regexp.Compile("(?i)" + query); err == nil {
		regex = re
	}

	for _, tool := range r.tools {
		nameLower := strings.ToLower(tool.Name)
		descLower := strings.ToLower(tool.Description)

		// Score 3: Exact name match
		if nameLower == queryLower {
			results = append(results, &SearchResult{
				Tool:        tool,
				Score:       3,
				MatchReason: "exact name match",
			})
			continue
		}

		// Score 2: Name contains query (or regex matches name)
		if strings.Contains(nameLower, queryLower) {
			results = append(results, &SearchResult{
				Tool:        tool,
				Score:       2,
				MatchReason: "name contains query",
			})
			continue
		}

		if regex != nil && regex.MatchString(tool.Name) {
			results = append(results, &SearchResult{
				Tool:        tool,
				Score:       2,
				MatchReason: "name matches pattern",
			})
			continue
		}

		// Score 1: Description contains query (or regex matches description)
		if strings.Contains(descLower, queryLower) {
			results = append(results, &SearchResult{
				Tool:        tool,
				Score:       1,
				MatchReason: "description contains query",
			})
			continue
		}

		if regex != nil && regex.MatchString(tool.Description) {
			results = append(results, &SearchResult{
				Tool:        tool,
				Score:       1,
				MatchReason: "description matches pattern",
			})
			continue
		}

		// Score 1: Keywords contain query
		for _, kw := range tool.Keywords {
			if strings.Contains(strings.ToLower(kw), queryLower) {
				results = append(results, &SearchResult{
					Tool:        tool,
					Score:       1,
					MatchReason: "keyword contains query",
				})
				break
			}
			if regex != nil && regex.MatchString(kw) {
				results = append(results, &SearchResult{
					Tool:        tool,
					Score:       1,
					MatchReason: "keyword matches pattern",
				})
				break
			}
		}
	}

	// Sort by score (highest first)
	sortSearchResults(results)

	return results
}

// SearchByCategory searches within a specific category.
func (r *ToolRegistry) SearchByCategory(query string, category ToolCategory) []*SearchResult {
	allResults := r.Search(query)
	filtered := make([]*SearchResult, 0)
	for _, result := range allResults {
		if result.Tool.Category == category {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// Count returns the total number of registered tools.
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// sortSearchResults sorts results by score descending.
func sortSearchResults(results []*SearchResult) {
	// Simple insertion sort (registry is small, no need for complex sorting)
	for i := 1; i < len(results); i++ {
		j := i
		for j > 0 && results[j].Score > results[j-1].Score {
			results[j], results[j-1] = results[j-1], results[j]
			j--
		}
	}
}

// GetDefaultToolMetadata returns the default metadata for all contextd tools.
// This includes proper defer_loading configuration following Anthropic's recommendation
// to keep 3-5 most frequently used tools non-deferred.
//
// Non-deferred tools (always loaded):
//   - tool_search: Required for tool discovery
//   - semantic_search: Primary search tool
//   - memory_search: Primary memory lookup
//
// All other tools are defer-loaded to minimize context usage.
func GetDefaultToolMetadata() []*ToolMetadata {
	return []*ToolMetadata{
		// ===== SEARCH TOOLS (CategorySearch) =====
		{
			Name:         "tool_search",
			Description:  "Search for available tools by name, description, or keyword. Returns tool_reference blocks for discovered tools. Use this to find relevant tools without loading all tool definitions into context.",
			Category:     CategorySearch,
			DeferLoading: false, // Core tool - always loaded
			Keywords:     []string{"find", "discover", "lookup", "tools", "search"},
		},
		{
			Name:         "tool_list",
			Description:  "List all available tools in the registry with their metadata. Use this to see what tools are available without searching.",
			Category:     CategorySearch,
			DeferLoading: true,
			Keywords:     []string{"list", "all", "tools", "catalog"},
		},

		// ===== MEMORY TOOLS (CategoryMemory) =====
		{
			Name:         "memory_search",
			Description:  "Search for relevant memories/strategies from past sessions",
			Category:     CategoryMemory,
			DeferLoading: false, // Core tool - always loaded
			Keywords:     []string{"find", "lookup", "recall", "strategy", "past", "reasoning"},
		},
		{
			Name:         "memory_record",
			Description:  "Record a new memory/learning from the current session",
			Category:     CategoryMemory,
			DeferLoading: true,
			Keywords:     []string{"save", "store", "remember", "learning"},
		},
		{
			Name:         "memory_feedback",
			Description:  "Provide feedback on a memory to adjust its confidence",
			Category:     CategoryMemory,
			DeferLoading: true,
			Keywords:     []string{"rate", "feedback", "helpful", "confidence"},
		},
		{
			Name:         "memory_outcome",
			Description:  "Report whether a task succeeded after using a memory. Call this after completing a task that used a retrieved memory to help the system learn which memories are actually useful.",
			Category:     CategoryMemory,
			DeferLoading: true,
			Keywords:     []string{"outcome", "success", "failure", "result", "report"},
		},
		{
			Name:         "memory_consolidate",
			Description:  "Consolidate similar memories to reduce redundancy and improve knowledge quality. Merges memories with similarity above threshold into synthesized consolidated memories.",
			Category:     CategoryMemory,
			DeferLoading: true,
			Keywords:     []string{"merge", "consolidate", "deduplicate", "optimize", "distill"},
		},

		// ===== CHECKPOINT TOOLS (CategoryCheckpoint) =====
		{
			Name:         "checkpoint_save",
			Description:  "Save a session checkpoint for later resumption",
			Category:     CategoryCheckpoint,
			DeferLoading: true,
			Keywords:     []string{"save", "persist", "snapshot", "context"},
		},
		{
			Name:         "checkpoint_list",
			Description:  "List checkpoints for a session or project",
			Category:     CategoryCheckpoint,
			DeferLoading: true,
			Keywords:     []string{"list", "show", "available", "snapshots"},
		},
		{
			Name:         "checkpoint_resume",
			Description:  "Resume from a checkpoint at specified level (summary, context, or full)",
			Category:     CategoryCheckpoint,
			DeferLoading: true,
			Keywords:     []string{"resume", "restore", "load", "recover"},
		},

		// ===== REMEDIATION TOOLS (CategoryRemediation) =====
		{
			Name:         "remediation_search",
			Description:  "Search for remediations by error message or pattern",
			Category:     CategoryRemediation,
			DeferLoading: true,
			Keywords:     []string{"error", "fix", "solution", "debug", "troubleshoot"},
		},
		{
			Name:         "remediation_record",
			Description:  "Record a new remediation for an error that was successfully fixed",
			Category:     CategoryRemediation,
			DeferLoading: true,
			Keywords:     []string{"record", "save", "document", "error", "fix"},
		},

		// ===== REPOSITORY TOOLS (CategoryRepository) =====
		{
			Name:         "semantic_search",
			Description:  "Smart search that uses semantic understanding, falling back to grep if needed. Use this when the agent would normally use the Search tool.",
			Category:     CategoryRepository,
			DeferLoading: false, // Core tool - always loaded
			Keywords:     []string{"search", "find", "code", "semantic", "grep", "pattern"},
		},
		{
			Name:         "repository_search",
			Description:  "Semantic search over indexed repository code in _codebase collection. Prefer using collection_name from repository_index output.",
			Category:     CategoryRepository,
			DeferLoading: true,
			Keywords:     []string{"search", "code", "indexed", "vector", "similar"},
		},
		{
			Name:         "repository_index",
			Description:  "Index a repository for semantic code search",
			Category:     CategoryRepository,
			DeferLoading: true,
			Keywords:     []string{"index", "embed", "vectorize", "codebase"},
		},

		// ===== TROUBLESHOOT TOOLS (CategoryTroubleshoot) =====
		{
			Name:         "troubleshoot_diagnose",
			Description:  "Diagnose an error using AI and known patterns",
			Category:     CategoryTroubleshoot,
			DeferLoading: true,
			Keywords:     []string{"diagnose", "analyze", "error", "debug", "root cause"},
		},

		// ===== FOLDING TOOLS (CategoryFolding) =====
		{
			Name:         "branch_create",
			Description:  "Create a new context-folding branch. Branches allow isolated sub-tasks with their own token budget, automatically cleaned up on return. Use for complex multi-step operations that need context isolation.",
			Category:     CategoryFolding,
			DeferLoading: true,
			Keywords:     []string{"branch", "isolate", "context", "budget", "subtask"},
		},
		{
			Name:         "branch_return",
			Description:  "Return from a context-folding branch with results. The message will be scrubbed for secrets before being returned to the parent context. Any child branches will be force-returned first.",
			Category:     CategoryFolding,
			DeferLoading: true,
			Keywords:     []string{"return", "complete", "branch", "result", "summary"},
		},
		{
			Name:         "branch_status",
			Description:  "Get the status of a specific branch or the active branch for a session. Returns branch state, budget usage, and depth information.",
			Category:     CategoryFolding,
			DeferLoading: true,
			Keywords:     []string{"status", "budget", "depth", "branch", "info"},
		},

		// ===== CONVERSATION TOOLS (CategoryConversation) =====
		{
			Name:         "conversation_index",
			Description:  "Index Claude Code conversation files for a project. Parses JSONL files, extracts messages and decisions, and stores them for semantic search.",
			Category:     CategoryConversation,
			DeferLoading: true,
			Keywords:     []string{"index", "conversation", "chat", "history", "parse"},
		},
		{
			Name:         "conversation_search",
			Description:  "Search indexed Claude Code conversations for relevant past context, decisions, and patterns.",
			Category:     CategoryConversation,
			DeferLoading: true,
			Keywords:     []string{"search", "conversation", "history", "context", "decision"},
		},

		// ===== REFLECTION TOOLS (CategoryReflection) =====
		{
			Name:         "reflect_report",
			Description:  "Generate a self-reflection report analyzing memories and patterns for a project. Returns insights about behavior patterns, success/failure trends, and recommendations.",
			Category:     CategoryReflection,
			DeferLoading: true,
			Keywords:     []string{"report", "reflect", "analyze", "patterns", "insights"},
		},
		{
			Name:         "reflect_analyze",
			Description:  "Analyze memories for behavioral patterns. Returns patterns grouped by category (success, failure, recurring, improving, declining) with confidence scores.",
			Category:     CategoryReflection,
			DeferLoading: true,
			Keywords:     []string{"analyze", "patterns", "behavior", "trends", "categories"},
		},
	}
}

// PopulateDefaults registers all default contextd tools with their metadata.
// This should be called during server initialization before tools are registered.
func (r *ToolRegistry) PopulateDefaults() {
	r.RegisterAll(GetDefaultToolMetadata())
}
