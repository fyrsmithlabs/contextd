package mcp

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToolRegistry_Register tests basic registration of tools
func TestToolRegistry_Register(t *testing.T) {
	registry := NewToolRegistry()

	tool := &ToolMetadata{
		Name:         "memory_search",
		Description:  "Search for relevant memories from past sessions",
		Category:     CategoryMemory,
		DeferLoading: false,
		Keywords:     []string{"search", "recall", "find"},
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	// Verify we can retrieve the tool
	retrieved, err := registry.Get("memory_search")
	require.NoError(t, err)
	assert.Equal(t, tool.Name, retrieved.Name)
	assert.Equal(t, tool.Description, retrieved.Description)
	assert.Equal(t, tool.Category, retrieved.Category)
	assert.Equal(t, tool.DeferLoading, retrieved.DeferLoading)
	assert.Equal(t, tool.Keywords, retrieved.Keywords)
}

// TestToolRegistry_RegisterDuplicate tests that registering duplicate tools fails
func TestToolRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewToolRegistry()

	tool := &ToolMetadata{
		Name:        "memory_search",
		Description: "Search for memories",
		Category:    CategoryMemory,
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	// Try to register again
	err = registry.Register(tool)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

// TestToolRegistry_RegisterInvalid tests validation of tool metadata
func TestToolRegistry_RegisterInvalid(t *testing.T) {
	registry := NewToolRegistry()

	tests := []struct {
		name     string
		tool     *ToolMetadata
		wantErr  string
	}{
		{
			name:    "nil tool",
			tool:    nil,
			wantErr: "tool metadata is required",
		},
		{
			name: "empty name",
			tool: &ToolMetadata{
				Description: "Test tool",
				Category:    CategoryMemory,
			},
			wantErr: "tool name is required",
		},
		{
			name: "empty description",
			tool: &ToolMetadata{
				Name:     "test_tool",
				Category: CategoryMemory,
			},
			wantErr: "tool description is required",
		},
		{
			name: "empty category",
			tool: &ToolMetadata{
				Name:        "test_tool",
				Description: "A test tool",
			},
			wantErr: "tool category is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(tt.tool)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestToolRegistry_RegisterAll tests batch registration
func TestToolRegistry_RegisterAll(t *testing.T) {
	t.Run("successful batch registration", func(t *testing.T) {
		registry := NewToolRegistry()

		tools := []*ToolMetadata{
			{
				Name:        "memory_search",
				Description: "Search memories",
				Category:    CategoryMemory,
			},
			{
				Name:        "memory_record",
				Description: "Record memory",
				Category:    CategoryMemory,
			},
			{
				Name:        "checkpoint_save",
				Description: "Save checkpoint",
				Category:    CategoryCheckpoint,
			},
		}

		err := registry.RegisterAll(tools)
		require.NoError(t, err)
		assert.Equal(t, 3, registry.Count())
	})

	t.Run("duplicate within batch", func(t *testing.T) {
		registry := NewToolRegistry()

		tools := []*ToolMetadata{
			{Name: "tool1", Description: "Tool 1", Category: CategoryMemory},
			{Name: "tool1", Description: "Tool 1 duplicate", Category: CategoryMemory},
		}

		err := registry.RegisterAll(tools)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
		// No tools should be registered on failure
		assert.Equal(t, 0, registry.Count())
	})

	t.Run("duplicate with existing", func(t *testing.T) {
		registry := NewToolRegistry()

		// Register one tool first
		err := registry.Register(&ToolMetadata{
			Name:        "existing_tool",
			Description: "Existing",
			Category:    CategoryMemory,
		})
		require.NoError(t, err)

		// Try to register batch with duplicate
		tools := []*ToolMetadata{
			{Name: "new_tool", Description: "New", Category: CategoryMemory},
			{Name: "existing_tool", Description: "Duplicate", Category: CategoryMemory},
		}

		err = registry.RegisterAll(tools)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
		// Only the first tool should exist
		assert.Equal(t, 1, registry.Count())
	})

	t.Run("invalid tool in batch", func(t *testing.T) {
		registry := NewToolRegistry()

		tools := []*ToolMetadata{
			{Name: "valid_tool", Description: "Valid", Category: CategoryMemory},
			{Name: "", Description: "Invalid", Category: CategoryMemory}, // Empty name
		}

		err := registry.RegisterAll(tools)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty name")
		// No tools should be registered
		assert.Equal(t, 0, registry.Count())
	})
}

// TestToolRegistry_Get tests retrieving tools
func TestToolRegistry_Get(t *testing.T) {
	registry := NewToolRegistry()

	tool := &ToolMetadata{
		Name:        "memory_search",
		Description: "Search memories",
		Category:    CategoryMemory,
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	// Test successful get
	retrieved, err := registry.Get("memory_search")
	require.NoError(t, err)
	assert.Equal(t, tool.Name, retrieved.Name)

	// Test not found
	_, err = registry.Get("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestToolRegistry_List tests listing all tools
func TestToolRegistry_List(t *testing.T) {
	registry := NewToolRegistry()

	tools := []*ToolMetadata{
		{Name: "tool1", Description: "Tool 1", Category: CategoryMemory},
		{Name: "tool2", Description: "Tool 2", Category: CategoryCheckpoint},
		{Name: "tool3", Description: "Tool 3", Category: CategoryRemediation},
	}

	err := registry.RegisterAll(tools)
	require.NoError(t, err)

	list := registry.List()
	assert.Equal(t, 3, len(list))
}

// TestToolRegistry_ListNames tests listing tool names
func TestToolRegistry_ListNames(t *testing.T) {
	registry := NewToolRegistry()

	tools := []*ToolMetadata{
		{Name: "tool1", Description: "Tool 1", Category: CategoryMemory},
		{Name: "tool2", Description: "Tool 2", Category: CategoryCheckpoint},
	}

	err := registry.RegisterAll(tools)
	require.NoError(t, err)

	names := registry.ListNames()
	assert.Equal(t, 2, len(names))
	assert.Contains(t, names, "tool1")
	assert.Contains(t, names, "tool2")
}

// TestToolRegistry_ListByCategory tests filtering by category
func TestToolRegistry_ListByCategory(t *testing.T) {
	registry := NewToolRegistry()

	tools := []*ToolMetadata{
		{Name: "memory_search", Description: "Search", Category: CategoryMemory},
		{Name: "memory_record", Description: "Record", Category: CategoryMemory},
		{Name: "checkpoint_save", Description: "Save", Category: CategoryCheckpoint},
	}

	err := registry.RegisterAll(tools)
	require.NoError(t, err)

	memoryTools := registry.ListByCategory(CategoryMemory)
	assert.Equal(t, 2, len(memoryTools))

	checkpointTools := registry.ListByCategory(CategoryCheckpoint)
	assert.Equal(t, 1, len(checkpointTools))
}

// TestToolRegistry_ListDeferred tests filtering deferred tools
func TestToolRegistry_ListDeferred(t *testing.T) {
	registry := NewToolRegistry()

	tools := []*ToolMetadata{
		{Name: "tool1", Description: "Tool 1", Category: CategoryMemory, DeferLoading: true},
		{Name: "tool2", Description: "Tool 2", Category: CategoryMemory, DeferLoading: false},
		{Name: "tool3", Description: "Tool 3", Category: CategoryCheckpoint, DeferLoading: true},
	}

	err := registry.RegisterAll(tools)
	require.NoError(t, err)

	deferred := registry.ListDeferred()
	assert.Equal(t, 2, len(deferred))

	nonDeferred := registry.ListNonDeferred()
	assert.Equal(t, 1, len(nonDeferred))
}

// TestToolRegistry_Count tests tool count
func TestToolRegistry_Count(t *testing.T) {
	registry := NewToolRegistry()

	assert.Equal(t, 0, registry.Count())

	tools := []*ToolMetadata{
		{Name: "tool1", Description: "Tool 1", Category: CategoryMemory},
		{Name: "tool2", Description: "Tool 2", Category: CategoryCheckpoint},
	}

	err := registry.RegisterAll(tools)
	require.NoError(t, err)
	assert.Equal(t, 2, registry.Count())
}

// TestToolRegistry_SearchExactMatch tests exact name matching (score 3)
func TestToolRegistry_SearchExactMatch(t *testing.T) {
	registry := NewToolRegistry()

	tool := &ToolMetadata{
		Name:        "memory_search",
		Description: "Search for memories",
		Category:    CategoryMemory,
		Keywords:    []string{"find", "recall"},
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	results, err := registry.Search("memory_search")
	require.NoError(t, err)
	require.Equal(t, 1, len(results))
	assert.Equal(t, "memory_search", results[0].Tool.Name)
	assert.Equal(t, 3, results[0].Score)
	assert.Contains(t, results[0].MatchReason, "exact")
}

// TestToolRegistry_SearchContains tests name contains matching (score 2)
func TestToolRegistry_SearchContains(t *testing.T) {
	registry := NewToolRegistry()

	tools := []*ToolMetadata{
		{Name: "memory_search", Description: "Search", Category: CategoryMemory},
		{Name: "memory_record", Description: "Record", Category: CategoryMemory},
		{Name: "checkpoint_save", Description: "Save", Category: CategoryCheckpoint},
	}

	err := registry.RegisterAll(tools)
	require.NoError(t, err)

	results, err := registry.Search("memory")
	require.NoError(t, err)
	assert.Equal(t, 2, len(results))
	
	// Both should have score 2 (contains)
	for _, r := range results {
		assert.Equal(t, 2, r.Score)
		assert.Contains(t, r.MatchReason, "name contains")
	}
}

// TestToolRegistry_SearchKeyword tests keyword matching (score 1)
func TestToolRegistry_SearchKeyword(t *testing.T) {
	registry := NewToolRegistry()

	tool := &ToolMetadata{
		Name:        "memory_search",
		Description: "Search for memories",
		Category:    CategoryMemory,
		Keywords:    []string{"find", "recall", "lookup"},
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	results, err := registry.Search("recall")
	require.NoError(t, err)
	require.Equal(t, 1, len(results))
	assert.Equal(t, 1, results[0].Score)
	assert.Contains(t, results[0].MatchReason, "keyword")
}

// TestToolRegistry_SearchDescription tests description matching (score 1)
func TestToolRegistry_SearchDescription(t *testing.T) {
	registry := NewToolRegistry()

	tool := &ToolMetadata{
		Name:        "memory_search",
		Description: "Search for relevant memories from past sessions",
		Category:    CategoryMemory,
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	results, err := registry.Search("relevant")
	require.NoError(t, err)
	require.Equal(t, 1, len(results))
	assert.Equal(t, 1, results[0].Score)
	assert.Contains(t, results[0].MatchReason, "description")
}

// TestToolRegistry_SearchSorting tests that results are sorted by score
func TestToolRegistry_SearchSorting(t *testing.T) {
	registry := NewToolRegistry()

	tools := []*ToolMetadata{
		{
			Name:        "memory_search",
			Description: "Search for memories",
			Category:    CategoryMemory,
			Keywords:    []string{"find"},
		},
		{
			Name:        "search_tool",
			Description: "A search utility",
			Category:    CategoryRepository,
		},
		{
			Name:        "semantic_search",
			Description: "Semantic code search",
			Category:    CategoryRepository,
		},
	}

	err := registry.RegisterAll(tools)
	require.NoError(t, err)

	results, err := registry.Search("search")
	require.NoError(t, err)
	require.True(t, len(results) >= 2)

	// First result should have highest score
	// "search_tool" has exact match (score 3) should be first
	// Others have "search" in name (score 2)
	assert.True(t, results[0].Score >= results[1].Score)
	if len(results) > 2 {
		assert.True(t, results[1].Score >= results[2].Score)
	}
}

// TestToolRegistry_SearchByCategory tests category-filtered search
func TestToolRegistry_SearchByCategory(t *testing.T) {
	registry := NewToolRegistry()

	tools := []*ToolMetadata{
		{Name: "memory_search", Description: "Search", Category: CategoryMemory},
		{Name: "semantic_search", Description: "Search", Category: CategoryRepository},
		{Name: "remediation_search", Description: "Search", Category: CategoryRemediation},
	}

	err := registry.RegisterAll(tools)
	require.NoError(t, err)

	results, err := registry.SearchByCategory("search", CategoryMemory)
	require.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "memory_search", results[0].Tool.Name)
}

// TestToolRegistry_SearchRegex tests regex pattern matching
func TestToolRegistry_SearchRegex(t *testing.T) {
	registry := NewToolRegistry()

	tools := []*ToolMetadata{
		{Name: "memory_search", Description: "Search", Category: CategoryMemory},
		{Name: "memory_record", Description: "Record", Category: CategoryMemory},
		{Name: "checkpoint_save", Description: "Save", Category: CategoryCheckpoint},
	}

	err := registry.RegisterAll(tools)
	require.NoError(t, err)

	// Test wildcard pattern
	results, err := registry.Search("memory_.*")
	require.NoError(t, err)
	assert.Equal(t, 2, len(results))

	// Test case-insensitive pattern
	results, err = registry.Search("(?i)MEMORY")
	require.NoError(t, err)
	assert.True(t, len(results) >= 2)
}

// TestToolRegistry_SearchCaseInsensitive tests case-insensitive search
func TestToolRegistry_SearchCaseInsensitive(t *testing.T) {
	registry := NewToolRegistry()

	tool := &ToolMetadata{
		Name:        "memory_search",
		Description: "Search for Memories",
		Category:    CategoryMemory,
		Keywords:    []string{"Recall"},
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	// Case-insensitive regex
	results, err := registry.Search("(?i)memory")
	require.NoError(t, err)
	assert.True(t, len(results) >= 1)

	results, err = registry.Search("(?i)recall")
	require.NoError(t, err)
	assert.True(t, len(results) >= 1)
}

// TestToolRegistry_ConcurrentAccess tests thread safety
func TestToolRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewToolRegistry()

	// Pre-populate with some tools
	tools := []*ToolMetadata{
		{Name: "tool1", Description: "Tool 1", Category: CategoryMemory},
		{Name: "tool2", Description: "Tool 2", Category: CategoryCheckpoint},
	}
	err := registry.RegisterAll(tools)
	require.NoError(t, err)

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = registry.Search("tool")
			_ = registry.List()
			_ = registry.Count()
		}()
	}

	// Concurrent writes (different tool names to avoid duplicate errors)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tool := &ToolMetadata{
				Name:        "concurrent_tool_" + string(rune('a'+idx)),
				Description: "Concurrent tool",
				Category:    CategoryMemory,
			}
			_ = registry.Register(tool)
		}(i)
	}

	wg.Wait()

	// Verify registry is still consistent
	assert.True(t, registry.Count() >= 2)
}

// TestToolRegistry_EmptySearch tests searching with empty query
func TestToolRegistry_EmptySearch(t *testing.T) {
	registry := NewToolRegistry()

	tools := []*ToolMetadata{
		{Name: "tool1", Description: "Tool 1", Category: CategoryMemory},
	}
	err := registry.RegisterAll(tools)
	require.NoError(t, err)

	results, err := registry.Search("")
	require.NoError(t, err)
	// Empty search should return all tools
	assert.Equal(t, 1, len(results))
}

// TestToolRegistry_NoMatches tests search with no matches
func TestToolRegistry_NoMatches(t *testing.T) {
	registry := NewToolRegistry()

	tools := []*ToolMetadata{
		{Name: "memory_search", Description: "Search", Category: CategoryMemory},
	}
	err := registry.RegisterAll(tools)
	require.NoError(t, err)

	results, err := registry.Search("nonexistent_pattern_xyz")
	require.NoError(t, err)
	assert.Equal(t, 0, len(results))
}
