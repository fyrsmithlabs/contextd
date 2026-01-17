package mcp

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ToolCategory represents the functional category of a tool
type ToolCategory string

const (
	CategoryMemory       ToolCategory = "memory"
	CategoryCheckpoint   ToolCategory = "checkpoint"
	CategoryRemediation  ToolCategory = "remediation"
	CategoryRepository   ToolCategory = "repository"
	CategoryTroubleshoot ToolCategory = "troubleshoot"
	CategoryFolding      ToolCategory = "folding"
	CategoryConversation ToolCategory = "conversation"
	CategoryReflection   ToolCategory = "reflection"
	CategorySearch       ToolCategory = "search"
)

// ToolMetadata contains metadata about a registered MCP tool
type ToolMetadata struct {
	// Name is the unique tool identifier (e.g., "memory_search")
	Name string

	// Description is a human-readable description of what the tool does
	Description string

	// Category groups related tools together
	Category ToolCategory

	// DeferLoading indicates if this tool should be loaded on-demand via tool_search
	// false = always loaded in initial context
	// true = loaded only when discovered via tool_search
	DeferLoading bool

	// Keywords are additional search terms for discovery
	Keywords []string
}

// SearchResult represents a tool matched by a search query
type SearchResult struct {
	// Tool is the matched tool metadata
	Tool *ToolMetadata

	// Score indicates match quality:
	// 3 = exact name match
	// 2 = name contains query
	// 1 = description or keyword match
	Score int

	// MatchReason explains why this tool matched
	MatchReason string
}

// ToolRegistry stores and searches tool metadata
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*ToolMetadata
}

// NewToolRegistry creates a new thread-safe tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*ToolMetadata),
	}
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool *ToolMetadata) error {
	if tool == nil {
		return fmt.Errorf("tool metadata is required")
	}
	if tool.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if tool.Description == "" {
		return fmt.Errorf("tool description is required")
	}
	if tool.Category == "" {
		return fmt.Errorf("tool category is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %q already registered", tool.Name)
	}

	r.tools[tool.Name] = tool
	return nil
}

// RegisterAll registers multiple tools in a batch
// If any tool fails validation, no tools are registered and an error is returned
func (r *ToolRegistry) RegisterAll(tools []*ToolMetadata) error {
	// Validate all tools first before registering any
	for i, tool := range tools {
		if tool == nil {
			return fmt.Errorf("tool at index %d is nil", i)
		}
		if tool.Name == "" {
			return fmt.Errorf("tool at index %d has empty name", i)
		}
		if tool.Description == "" {
			return fmt.Errorf("tool at index %d (%s) has empty description", i, tool.Name)
		}
		if tool.Category == "" {
			return fmt.Errorf("tool at index %d (%s) has empty category", i, tool.Name)
		}
	}

	// Check for duplicates within the batch and with existing tools
	r.mu.RLock()
	seen := make(map[string]bool, len(tools))
	for i, tool := range tools {
		if seen[tool.Name] {
			r.mu.RUnlock()
			return fmt.Errorf("duplicate tool %q at index %d in batch", tool.Name, i)
		}
		seen[tool.Name] = true

		if _, exists := r.tools[tool.Name]; exists {
			r.mu.RUnlock()
			return fmt.Errorf("tool %q already registered", tool.Name)
		}
	}
	r.mu.RUnlock()

	// Now register all tools (all validations passed)
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, tool := range tools {
		r.tools[tool.Name] = tool
	}
	return nil
}

// Get retrieves a tool by name
func (r *ToolRegistry) Get(name string) (*ToolMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %q not found", name)
	}

	return tool, nil
}

// List returns all registered tools
func (r *ToolRegistry) List() []*ToolMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ToolMetadata, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}

	return result
}

// ListNames returns all registered tool names
func (r *ToolRegistry) ListNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	return names
}

// ListByCategory returns tools filtered by category
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

// ListDeferred returns tools with DeferLoading=true
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

// ListNonDeferred returns tools with DeferLoading=false
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

// Count returns the total number of registered tools
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

// Search finds tools matching the query using regex-compatible patterns
// Supports Python re.search() compatible patterns
func (r *ToolRegistry) Search(query string) ([]SearchResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Empty query returns all tools
	if query == "" {
		results := make([]SearchResult, 0, len(r.tools))
		for _, tool := range r.tools {
			results = append(results, SearchResult{
				Tool:        tool,
				Score:       1,
				MatchReason: "empty query matches all",
			})
		}
		return results, nil
	}

	// Check if query contains regex special characters
	// If it does, treat as regex; otherwise use literal matching
	isRegex := containsRegexMetaChars(query)

	if isRegex {
		// Try to compile as regex
		re, err := regexp.Compile(query)
		if err != nil {
			// Invalid regex, fall back to literal
			return r.searchLiteral(query), nil
		}
		return r.searchRegex(query, re), nil
	}

	// Simple literal search for plain strings
	return r.searchLiteral(query), nil
}

// containsRegexMetaChars checks if a string contains regex special characters
func containsRegexMetaChars(s string) bool {
	// Common regex metacharacters and constructs
	metaChars := []string{
		".*", ".+", "\\", "^", "$", "[", "]", "{", "}", "(", ")", "|", "?", "+", "*",
		"(?i)", "(?m)", "(?s)",
	}
	for _, meta := range metaChars {
		if strings.Contains(s, meta) {
			return true
		}
	}
	return false
}

// searchLiteral performs literal string matching (case-insensitive)
func (r *ToolRegistry) searchLiteral(query string) []SearchResult {
	queryLower := strings.ToLower(query)
	results := make([]SearchResult, 0)

	for _, tool := range r.tools {
		nameLower := strings.ToLower(tool.Name)
		descLower := strings.ToLower(tool.Description)

		// Exact match (score 3)
		if nameLower == queryLower {
			results = append(results, SearchResult{
				Tool:        tool,
				Score:       3,
				MatchReason: "exact name match",
			})
			continue
		}

		// Name contains (score 2)
		if strings.Contains(nameLower, queryLower) {
			results = append(results, SearchResult{
				Tool:        tool,
				Score:       2,
				MatchReason: "name contains query",
			})
			continue
		}

		// Keyword match (score 1)
		keywordMatch := false
		for _, keyword := range tool.Keywords {
			if strings.Contains(strings.ToLower(keyword), queryLower) {
				results = append(results, SearchResult{
					Tool:        tool,
					Score:       1,
					MatchReason: "keyword match",
				})
				keywordMatch = true
				break
			}
		}
		if keywordMatch {
			continue
		}

		// Description match (score 1)
		if strings.Contains(descLower, queryLower) {
			results = append(results, SearchResult{
				Tool:        tool,
				Score:       1,
				MatchReason: "description match",
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// searchRegex performs regex pattern matching
// For regex patterns, we only use score 2 (name) and score 1 (keyword/description)
// Score 3 (exact match) is reserved for literal string searches
func (r *ToolRegistry) searchRegex(query string, re *regexp.Regexp) []SearchResult {
	results := make([]SearchResult, 0)

	for _, tool := range r.tools {
		// Regex match on name (score 2)
		if re.MatchString(tool.Name) {
			results = append(results, SearchResult{
				Tool:        tool,
				Score:       2,
				MatchReason: "name matches pattern",
			})
			continue
		}

		// Regex match on keywords (score 1)
		keywordMatch := false
		for _, keyword := range tool.Keywords {
			if re.MatchString(keyword) {
				results = append(results, SearchResult{
					Tool:        tool,
					Score:       1,
					MatchReason: "keyword matches pattern",
				})
				keywordMatch = true
				break
			}
		}
		if keywordMatch {
			continue
		}

		// Regex match on description (score 1)
		if re.MatchString(tool.Description) {
			results = append(results, SearchResult{
				Tool:        tool,
				Score:       1,
				MatchReason: "description matches pattern",
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// SearchByCategory finds tools matching the query within a specific category
func (r *ToolRegistry) SearchByCategory(query string, category ToolCategory) ([]SearchResult, error) {
	// Get all search results
	results, err := r.Search(query)
	if err != nil {
		return nil, err
	}

	// Filter by category
	filtered := make([]SearchResult, 0)
	for _, result := range results {
		if result.Tool.Category == category {
			filtered = append(filtered, result)
		}
	}

	return filtered, nil
}
