package mcp

import (
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()
	require.NotNil(t, registry)
	require.NotNil(t, registry.tools)
	require.Equal(t, 0, registry.Count())
}

func TestToolRegistry_Register(t *testing.T) {
	t.Run("registers valid tool", func(t *testing.T) {
		registry := NewToolRegistry()
		tool := &ToolMetadata{
			Name:        "memory_search",
			Description: "Search for memories",
			Category:    CategoryMemory,
		}

		registry.Register(tool)

		require.Equal(t, 1, registry.Count())
		retrieved, ok := registry.Get("memory_search")
		require.True(t, ok)
		require.Equal(t, tool, retrieved)
	})

	t.Run("ignores nil tool", func(t *testing.T) {
		registry := NewToolRegistry()
		registry.Register(nil)
		require.Equal(t, 0, registry.Count())
	})

	t.Run("ignores tool with empty name", func(t *testing.T) {
		registry := NewToolRegistry()
		tool := &ToolMetadata{
			Name:        "",
			Description: "Some description",
		}

		registry.Register(tool)
		require.Equal(t, 0, registry.Count())
	})

	t.Run("overwrites existing tool with same name", func(t *testing.T) {
		registry := NewToolRegistry()
		tool1 := &ToolMetadata{
			Name:        "memory_search",
			Description: "First description",
		}
		tool2 := &ToolMetadata{
			Name:        "memory_search",
			Description: "Second description",
		}

		registry.Register(tool1)
		registry.Register(tool2)

		require.Equal(t, 1, registry.Count())
		retrieved, ok := registry.Get("memory_search")
		require.True(t, ok)
		require.Equal(t, "Second description", retrieved.Description)
	})
}

func TestToolRegistry_RegisterAll(t *testing.T) {
	t.Run("registers multiple tools", func(t *testing.T) {
		registry := NewToolRegistry()
		tools := []*ToolMetadata{
			{Name: "tool1", Description: "First tool"},
			{Name: "tool2", Description: "Second tool"},
			{Name: "tool3", Description: "Third tool"},
		}

		registry.RegisterAll(tools)

		require.Equal(t, 3, registry.Count())
		for _, tool := range tools {
			retrieved, ok := registry.Get(tool.Name)
			require.True(t, ok)
			require.Equal(t, tool.Description, retrieved.Description)
		}
	})

	t.Run("handles empty slice", func(t *testing.T) {
		registry := NewToolRegistry()
		registry.RegisterAll([]*ToolMetadata{})
		require.Equal(t, 0, registry.Count())
	})

	t.Run("filters nil and empty name tools", func(t *testing.T) {
		registry := NewToolRegistry()
		tools := []*ToolMetadata{
			{Name: "valid_tool", Description: "Valid"},
			nil,
			{Name: "", Description: "Empty name"},
		}

		registry.RegisterAll(tools)

		require.Equal(t, 1, registry.Count())
		_, ok := registry.Get("valid_tool")
		require.True(t, ok)
	})
}

func TestToolRegistry_Get(t *testing.T) {
	t.Run("returns existing tool", func(t *testing.T) {
		registry := NewToolRegistry()
		tool := &ToolMetadata{
			Name:        "checkpoint_save",
			Description: "Save a checkpoint",
			Category:    CategoryCheckpoint,
		}
		registry.Register(tool)

		retrieved, ok := registry.Get("checkpoint_save")
		require.True(t, ok)
		require.Equal(t, tool, retrieved)
	})

	t.Run("returns false for non-existent tool", func(t *testing.T) {
		registry := NewToolRegistry()

		retrieved, ok := registry.Get("non_existent")
		require.False(t, ok)
		require.Nil(t, retrieved)
	})
}

func TestToolRegistry_List(t *testing.T) {
	t.Run("returns all tools", func(t *testing.T) {
		registry := NewToolRegistry()
		tools := []*ToolMetadata{
			{Name: "tool1", Description: "First tool"},
			{Name: "tool2", Description: "Second tool"},
		}
		registry.RegisterAll(tools)

		list := registry.List()
		require.Len(t, list, 2)

		names := make([]string, len(list))
		for i, tool := range list {
			names[i] = tool.Name
		}
		sort.Strings(names)
		require.Equal(t, []string{"tool1", "tool2"}, names)
	})

	t.Run("returns empty slice for empty registry", func(t *testing.T) {
		registry := NewToolRegistry()
		list := registry.List()
		require.NotNil(t, list)
		require.Len(t, list, 0)
	})
}

func TestToolRegistry_ListNames(t *testing.T) {
	t.Run("returns all tool names", func(t *testing.T) {
		registry := NewToolRegistry()
		tools := []*ToolMetadata{
			{Name: "alpha", Description: "Alpha tool"},
			{Name: "beta", Description: "Beta tool"},
			{Name: "gamma", Description: "Gamma tool"},
		}
		registry.RegisterAll(tools)

		names := registry.ListNames()
		require.Len(t, names, 3)
		sort.Strings(names)
		require.Equal(t, []string{"alpha", "beta", "gamma"}, names)
	})

	t.Run("returns empty slice for empty registry", func(t *testing.T) {
		registry := NewToolRegistry()
		names := registry.ListNames()
		require.NotNil(t, names)
		require.Len(t, names, 0)
	})
}

func TestToolRegistry_ListByCategory(t *testing.T) {
	registry := NewToolRegistry()
	tools := []*ToolMetadata{
		{Name: "memory_search", Description: "Search memories", Category: CategoryMemory},
		{Name: "memory_record", Description: "Record memory", Category: CategoryMemory},
		{Name: "checkpoint_save", Description: "Save checkpoint", Category: CategoryCheckpoint},
		{Name: "remediation_search", Description: "Search remediations", Category: CategoryRemediation},
	}
	registry.RegisterAll(tools)

	t.Run("returns tools in category", func(t *testing.T) {
		memoryTools := registry.ListByCategory(CategoryMemory)
		require.Len(t, memoryTools, 2)

		names := make([]string, len(memoryTools))
		for i, tool := range memoryTools {
			names[i] = tool.Name
		}
		sort.Strings(names)
		require.Equal(t, []string{"memory_record", "memory_search"}, names)
	})

	t.Run("returns single tool in category", func(t *testing.T) {
		checkpointTools := registry.ListByCategory(CategoryCheckpoint)
		require.Len(t, checkpointTools, 1)
		require.Equal(t, "checkpoint_save", checkpointTools[0].Name)
	})

	t.Run("returns empty slice for empty category", func(t *testing.T) {
		foldingTools := registry.ListByCategory(CategoryFolding)
		require.NotNil(t, foldingTools)
		require.Len(t, foldingTools, 0)
	})
}

func TestToolRegistry_ListNonDeferred(t *testing.T) {
	registry := NewToolRegistry()
	tools := []*ToolMetadata{
		{Name: "tool1", Description: "Non-deferred 1", DeferLoading: false},
		{Name: "tool2", Description: "Deferred 1", DeferLoading: true},
		{Name: "tool3", Description: "Non-deferred 2", DeferLoading: false},
		{Name: "tool4", Description: "Deferred 2", DeferLoading: true},
	}
	registry.RegisterAll(tools)

	t.Run("returns non-deferred tools", func(t *testing.T) {
		nonDeferred := registry.ListNonDeferred()
		require.Len(t, nonDeferred, 2)

		for _, tool := range nonDeferred {
			require.False(t, tool.DeferLoading)
		}
	})

	t.Run("returns empty slice when all deferred", func(t *testing.T) {
		deferredRegistry := NewToolRegistry()
		deferredRegistry.RegisterAll([]*ToolMetadata{
			{Name: "deferred1", DeferLoading: true},
			{Name: "deferred2", DeferLoading: true},
		})

		nonDeferred := deferredRegistry.ListNonDeferred()
		require.NotNil(t, nonDeferred)
		require.Len(t, nonDeferred, 0)
	})
}

func TestToolRegistry_ListDeferred(t *testing.T) {
	registry := NewToolRegistry()
	tools := []*ToolMetadata{
		{Name: "tool1", Description: "Non-deferred 1", DeferLoading: false},
		{Name: "tool2", Description: "Deferred 1", DeferLoading: true},
		{Name: "tool3", Description: "Non-deferred 2", DeferLoading: false},
		{Name: "tool4", Description: "Deferred 2", DeferLoading: true},
	}
	registry.RegisterAll(tools)

	t.Run("returns deferred tools", func(t *testing.T) {
		deferred := registry.ListDeferred()
		require.Len(t, deferred, 2)

		for _, tool := range deferred {
			require.True(t, tool.DeferLoading)
		}
	})

	t.Run("returns empty slice when none deferred", func(t *testing.T) {
		nonDeferredRegistry := NewToolRegistry()
		nonDeferredRegistry.RegisterAll([]*ToolMetadata{
			{Name: "non1", DeferLoading: false},
			{Name: "non2", DeferLoading: false},
		})

		deferred := nonDeferredRegistry.ListDeferred()
		require.NotNil(t, deferred)
		require.Len(t, deferred, 0)
	})
}

func TestToolRegistry_Count(t *testing.T) {
	t.Run("returns zero for empty registry", func(t *testing.T) {
		registry := NewToolRegistry()
		require.Equal(t, 0, registry.Count())
	})

	t.Run("returns correct count", func(t *testing.T) {
		registry := NewToolRegistry()
		registry.RegisterAll([]*ToolMetadata{
			{Name: "tool1"},
			{Name: "tool2"},
			{Name: "tool3"},
		})
		require.Equal(t, 3, registry.Count())
	})
}

func TestToolRegistry_Search(t *testing.T) {
	registry := NewToolRegistry()
	tools := []*ToolMetadata{
		{Name: "memory_search", Description: "Search for memories in the reasoning bank", Category: CategoryMemory, Keywords: []string{"find", "lookup", "recall"}},
		{Name: "memory_record", Description: "Record a new memory", Category: CategoryMemory, Keywords: []string{"save", "store"}},
		{Name: "checkpoint_save", Description: "Save context to a checkpoint", Category: CategoryCheckpoint, Keywords: []string{"persist", "snapshot"}},
		{Name: "checkpoint_list", Description: "List available checkpoints", Category: CategoryCheckpoint},
		{Name: "remediation_search", Description: "Search for error remediation patterns", Category: CategoryRemediation, Keywords: []string{"fix", "error", "debug"}},
	}
	registry.RegisterAll(tools)

	t.Run("empty query returns nil", func(t *testing.T) {
		results := registry.Search("")
		require.Nil(t, results)
	})

	t.Run("exact name match returns score 3", func(t *testing.T) {
		results := registry.Search("memory_search")
		require.Len(t, results, 1)
		require.Equal(t, "memory_search", results[0].Tool.Name)
		require.Equal(t, 3, results[0].Score)
		require.Equal(t, "exact name match", results[0].MatchReason)
	})

	t.Run("case-insensitive exact name match", func(t *testing.T) {
		results := registry.Search("MEMORY_SEARCH")
		require.Len(t, results, 1)
		require.Equal(t, "memory_search", results[0].Tool.Name)
		require.Equal(t, 3, results[0].Score)
	})

	t.Run("name contains query returns score 2", func(t *testing.T) {
		results := registry.Search("checkpoint")
		require.Len(t, results, 2)

		// All should have score 2 (name contains query)
		for _, result := range results {
			require.Equal(t, 2, result.Score)
			require.Equal(t, "name contains query", result.MatchReason)
		}
	})

	t.Run("description contains query returns score 1", func(t *testing.T) {
		results := registry.Search("context")
		require.Len(t, results, 1)
		require.Equal(t, "checkpoint_save", results[0].Tool.Name)
		require.Equal(t, 1, results[0].Score)
		require.Equal(t, "description contains query", results[0].MatchReason)
	})

	t.Run("keyword contains query returns score 1", func(t *testing.T) {
		results := registry.Search("debug")
		require.Len(t, results, 1)
		require.Equal(t, "remediation_search", results[0].Tool.Name)
		require.Equal(t, 1, results[0].Score)
		require.Equal(t, "keyword contains query", results[0].MatchReason)
	})

	t.Run("results sorted by score descending", func(t *testing.T) {
		// "memory" should match:
		// - memory_search (score 2 - name contains)
		// - memory_record (score 2 - name contains)
		results := registry.Search("memory")
		require.Len(t, results, 2)

		// Both should have score 2
		for _, result := range results {
			require.Equal(t, 2, result.Score)
		}
	})

	t.Run("regex pattern matches name", func(t *testing.T) {
		results := registry.Search("^memory_.*")
		require.Len(t, results, 2)

		for _, result := range results {
			require.Contains(t, result.Tool.Name, "memory_")
		}
	})

	t.Run("regex pattern matches description", func(t *testing.T) {
		results := registry.Search("error.*patterns")
		require.Len(t, results, 1)
		require.Equal(t, "remediation_search", results[0].Tool.Name)
		require.Equal(t, 1, results[0].Score)
		require.Equal(t, "description matches pattern", results[0].MatchReason)
	})

	t.Run("no matches returns nil", func(t *testing.T) {
		results := registry.Search("nonexistent_query_xyz")
		// Search returns nil when no matches found (results slice starts as nil)
		require.Nil(t, results)
	})

	t.Run("invalid regex falls back to literal match", func(t *testing.T) {
		// "[" is an invalid regex
		results := registry.Search("[")
		// Should not panic and should return nil (no tools contain literal "[")
		require.Nil(t, results)
	})
}

func TestToolRegistry_SearchByCategory(t *testing.T) {
	registry := NewToolRegistry()
	tools := []*ToolMetadata{
		{Name: "memory_search", Description: "Search memories", Category: CategoryMemory, Keywords: []string{"find"}},
		{Name: "memory_record", Description: "Record memory", Category: CategoryMemory},
		{Name: "checkpoint_search", Description: "Search checkpoints", Category: CategoryCheckpoint},
	}
	registry.RegisterAll(tools)

	t.Run("filters results by category", func(t *testing.T) {
		results := registry.SearchByCategory("search", CategoryMemory)
		require.Len(t, results, 1)
		require.Equal(t, "memory_search", results[0].Tool.Name)
	})

	t.Run("returns empty slice when no matches in category", func(t *testing.T) {
		results := registry.SearchByCategory("record", CategoryCheckpoint)
		require.NotNil(t, results)
		require.Len(t, results, 0)
	})

	t.Run("empty query returns empty slice", func(t *testing.T) {
		results := registry.SearchByCategory("", CategoryMemory)
		// SearchByCategory explicitly creates an empty slice, even when Search returns nil
		require.NotNil(t, results)
		require.Len(t, results, 0)
	})
}

func TestSortSearchResults(t *testing.T) {
	t.Run("sorts by score descending", func(t *testing.T) {
		results := []*SearchResult{
			{Tool: &ToolMetadata{Name: "tool1"}, Score: 1},
			{Tool: &ToolMetadata{Name: "tool2"}, Score: 3},
			{Tool: &ToolMetadata{Name: "tool3"}, Score: 2},
		}

		sortSearchResults(results)

		require.Equal(t, 3, results[0].Score)
		require.Equal(t, 2, results[1].Score)
		require.Equal(t, 1, results[2].Score)
	})

	t.Run("handles empty slice", func(t *testing.T) {
		results := []*SearchResult{}
		sortSearchResults(results)
		require.Len(t, results, 0)
	})

	t.Run("handles single element", func(t *testing.T) {
		results := []*SearchResult{
			{Tool: &ToolMetadata{Name: "only"}, Score: 1},
		}
		sortSearchResults(results)
		require.Len(t, results, 1)
		require.Equal(t, "only", results[0].Tool.Name)
	})

	t.Run("handles already sorted", func(t *testing.T) {
		results := []*SearchResult{
			{Tool: &ToolMetadata{Name: "high"}, Score: 3},
			{Tool: &ToolMetadata{Name: "mid"}, Score: 2},
			{Tool: &ToolMetadata{Name: "low"}, Score: 1},
		}
		sortSearchResults(results)

		require.Equal(t, "high", results[0].Tool.Name)
		require.Equal(t, "mid", results[1].Tool.Name)
		require.Equal(t, "low", results[2].Tool.Name)
	})
}

func TestToolRegistry_Concurrency(t *testing.T) {
	registry := NewToolRegistry()

	// Pre-register some tools
	for i := 0; i < 10; i++ {
		registry.Register(&ToolMetadata{
			Name:        "initial_tool_" + string(rune('a'+i)),
			Description: "Initial tool",
			Category:    CategoryMemory,
		})
	}

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent reads
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = registry.Count()
			_ = registry.List()
			_ = registry.ListNames()
			_ = registry.ListByCategory(CategoryMemory)
			_ = registry.ListDeferred()
			_ = registry.ListNonDeferred()
			_, _ = registry.Get("initial_tool_a")
			_ = registry.Search("memory")
		}
	}()

	// Concurrent writes
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			registry.Register(&ToolMetadata{
				Name:        "concurrent_tool_" + string(rune(i%26+'A')),
				Description: "Concurrent tool",
			})
		}
	}()

	// More concurrent reads
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = registry.Search("tool")
			_ = registry.SearchByCategory("tool", CategoryMemory)
		}
	}()

	wg.Wait()

	// Verify registry is still functional
	require.Greater(t, registry.Count(), 0)
}

func TestToolCategories(t *testing.T) {
	// Verify all expected categories are defined
	categories := []ToolCategory{
		CategoryMemory,
		CategoryCheckpoint,
		CategoryRemediation,
		CategoryRepository,
		CategoryTroubleshoot,
		CategoryFolding,
		CategoryConversation,
		CategoryReflection,
		CategorySearch,
	}

	expected := []string{
		"memory",
		"checkpoint",
		"remediation",
		"repository",
		"troubleshoot",
		"folding",
		"conversation",
		"reflection",
		"search",
	}

	require.Len(t, categories, len(expected))

	for i, cat := range categories {
		require.Equal(t, ToolCategory(expected[i]), cat)
	}
}

func TestGetDefaultToolMetadata(t *testing.T) {
	tools := GetDefaultToolMetadata()

	t.Run("returns all 23 tools", func(t *testing.T) {
		require.Len(t, tools, 23, "Expected 23 tools, got %d", len(tools))
	})

	t.Run("has correct non-deferred tools", func(t *testing.T) {
		// According to implementation plan, these 3 tools should NOT be deferred
		nonDeferredExpected := []string{"tool_search", "semantic_search", "memory_search"}

		var nonDeferred []string
		for _, tool := range tools {
			if !tool.DeferLoading {
				nonDeferred = append(nonDeferred, tool.Name)
			}
		}

		sort.Strings(nonDeferred)
		sort.Strings(nonDeferredExpected)
		require.Equal(t, nonDeferredExpected, nonDeferred,
			"Expected exactly 3 non-deferred tools: tool_search, semantic_search, memory_search")
	})

	t.Run("has correct deferred tool count", func(t *testing.T) {
		var deferredCount int
		for _, tool := range tools {
			if tool.DeferLoading {
				deferredCount++
			}
		}
		require.Equal(t, 20, deferredCount, "Expected 20 deferred tools (23 - 3 non-deferred)")
	})

	t.Run("all tools have required fields", func(t *testing.T) {
		for _, tool := range tools {
			require.NotEmpty(t, tool.Name, "Tool name should not be empty")
			require.NotEmpty(t, tool.Description, "Tool %s should have description", tool.Name)
			require.NotEmpty(t, tool.Category, "Tool %s should have category", tool.Name)
		}
	})

	t.Run("all tools have keywords", func(t *testing.T) {
		for _, tool := range tools {
			require.NotEmpty(t, tool.Keywords, "Tool %s should have keywords", tool.Name)
		}
	})

	t.Run("contains expected tool names", func(t *testing.T) {
		expectedTools := []string{
			// Search tools
			"tool_search", "tool_list",
			// Memory tools
			"memory_search", "memory_record", "memory_feedback", "memory_outcome", "memory_consolidate",
			// Checkpoint tools
			"checkpoint_save", "checkpoint_list", "checkpoint_resume",
			// Remediation tools
			"remediation_search", "remediation_record",
			// Repository tools
			"semantic_search", "repository_search", "repository_index",
			// Troubleshoot tools
			"troubleshoot_diagnose",
			// Folding tools
			"branch_create", "branch_return", "branch_status",
			// Conversation tools
			"conversation_index", "conversation_search",
			// Reflection tools
			"reflect_report", "reflect_analyze",
		}

		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Name] = true
		}

		for _, expected := range expectedTools {
			require.True(t, toolNames[expected], "Expected tool %s not found in defaults", expected)
		}
	})

	t.Run("tools have correct categories", func(t *testing.T) {
		categoryMap := make(map[string]ToolCategory)
		for _, tool := range tools {
			categoryMap[tool.Name] = tool.Category
		}

		// Verify category assignments
		require.Equal(t, CategorySearch, categoryMap["tool_search"])
		require.Equal(t, CategorySearch, categoryMap["tool_list"])
		require.Equal(t, CategoryMemory, categoryMap["memory_search"])
		require.Equal(t, CategoryMemory, categoryMap["memory_record"])
		require.Equal(t, CategoryCheckpoint, categoryMap["checkpoint_save"])
		require.Equal(t, CategoryRemediation, categoryMap["remediation_search"])
		require.Equal(t, CategoryRepository, categoryMap["semantic_search"])
		require.Equal(t, CategoryRepository, categoryMap["repository_index"])
		require.Equal(t, CategoryTroubleshoot, categoryMap["troubleshoot_diagnose"])
		require.Equal(t, CategoryFolding, categoryMap["branch_create"])
		require.Equal(t, CategoryConversation, categoryMap["conversation_index"])
		require.Equal(t, CategoryReflection, categoryMap["reflect_report"])
	})
}

func TestToolRegistry_PopulateDefaults(t *testing.T) {
	t.Run("populates all default tools", func(t *testing.T) {
		registry := NewToolRegistry()
		require.Equal(t, 0, registry.Count(), "Registry should start empty")

		registry.PopulateDefaults()

		require.Equal(t, 23, registry.Count(), "Registry should have 23 tools after PopulateDefaults")
	})

	t.Run("all tools retrievable by name", func(t *testing.T) {
		registry := NewToolRegistry()
		registry.PopulateDefaults()

		expectedTools := []string{
			"tool_search", "tool_list",
			"memory_search", "memory_record", "memory_feedback", "memory_outcome", "memory_consolidate",
			"checkpoint_save", "checkpoint_list", "checkpoint_resume",
			"remediation_search", "remediation_record",
			"semantic_search", "repository_search", "repository_index",
			"troubleshoot_diagnose",
			"branch_create", "branch_return", "branch_status",
			"conversation_index", "conversation_search",
			"reflect_report", "reflect_analyze",
		}

		for _, name := range expectedTools {
			tool, ok := registry.Get(name)
			require.True(t, ok, "Tool %s should be retrievable", name)
			require.NotNil(t, tool, "Tool %s should not be nil", name)
			require.Equal(t, name, tool.Name)
		}
	})

	t.Run("non-deferred tools retrievable via ListNonDeferred", func(t *testing.T) {
		registry := NewToolRegistry()
		registry.PopulateDefaults()

		nonDeferred := registry.ListNonDeferred()
		require.Len(t, nonDeferred, 3, "Should have exactly 3 non-deferred tools")

		names := make([]string, len(nonDeferred))
		for i, tool := range nonDeferred {
			names[i] = tool.Name
		}
		sort.Strings(names)

		require.Equal(t, []string{"memory_search", "semantic_search", "tool_search"}, names)
	})

	t.Run("deferred tools retrievable via ListDeferred", func(t *testing.T) {
		registry := NewToolRegistry()
		registry.PopulateDefaults()

		deferred := registry.ListDeferred()
		require.Len(t, deferred, 20, "Should have exactly 20 deferred tools")

		// Verify all deferred tools have DeferLoading = true
		for _, tool := range deferred {
			require.True(t, tool.DeferLoading, "Tool %s should be deferred", tool.Name)
		}
	})

	t.Run("tools searchable by category", func(t *testing.T) {
		registry := NewToolRegistry()
		registry.PopulateDefaults()

		// Test each category has expected count
		categoryExpected := map[ToolCategory]int{
			CategorySearch:       2,
			CategoryMemory:       5,
			CategoryCheckpoint:   3,
			CategoryRemediation:  2,
			CategoryRepository:   3,
			CategoryTroubleshoot: 1,
			CategoryFolding:      3,
			CategoryConversation: 2,
			CategoryReflection:   2,
		}

		for cat, expectedCount := range categoryExpected {
			tools := registry.ListByCategory(cat)
			require.Len(t, tools, expectedCount, "Category %s should have %d tools", cat, expectedCount)
		}
	})

	t.Run("tools searchable by query", func(t *testing.T) {
		registry := NewToolRegistry()
		registry.PopulateDefaults()

		// Search for "memory" should find all memory tools
		results := registry.Search("memory")
		require.GreaterOrEqual(t, len(results), 5, "Should find at least 5 memory-related tools")

		// Search for "search" should find multiple tools
		results = registry.Search("search")
		require.GreaterOrEqual(t, len(results), 3, "Should find at least 3 search-related tools")

		// Search for "branch" should find folding tools
		results = registry.Search("branch")
		require.GreaterOrEqual(t, len(results), 3, "Should find at least 3 branch tools")
	})

	t.Run("idempotent - calling twice doesn't duplicate", func(t *testing.T) {
		registry := NewToolRegistry()
		registry.PopulateDefaults()
		registry.PopulateDefaults() // Call again

		// Should still have exactly 23 tools (not 46)
		require.Equal(t, 23, registry.Count(), "PopulateDefaults should be idempotent")
	})
}
