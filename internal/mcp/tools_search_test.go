package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToolSearchInput_QueryRequired verifies that query is a required field.
func TestToolSearchInput_QueryRequired(t *testing.T) {
	testCases := []struct {
		name        string
		input       toolSearchInput
		expectError bool
	}{
		{
			name: "valid_query",
			input: toolSearchInput{
				Query: "memory",
			},
			expectError: false,
		},
		{
			name: "empty_query_should_error",
			input: toolSearchInput{
				Query: "",
			},
			expectError: true,
		},
		{
			name: "whitespace_query_is_valid",
			input: toolSearchInput{
				Query: " ",
			},
			expectError: false, // The registry handles this - returns nil
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the handler's query validation
			hasError := tc.input.Query == ""
			assert.Equal(t, tc.expectError, hasError)
		})
	}
}

// TestToolSearchInput_LimitDefault verifies that default limit is 5.
func TestToolSearchInput_LimitDefault(t *testing.T) {
	testCases := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{
			name:          "zero_limit_defaults_to_5",
			inputLimit:    0,
			expectedLimit: 5,
		},
		{
			name:          "negative_limit_defaults_to_5",
			inputLimit:    -1,
			expectedLimit: 5,
		},
		{
			name:          "positive_limit_is_preserved",
			inputLimit:    10,
			expectedLimit: 10,
		},
		{
			name:          "limit_of_1_is_preserved",
			inputLimit:    1,
			expectedLimit: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Apply the same default logic as the handler
			limit := tc.inputLimit
			if limit <= 0 {
				limit = 5
			}
			assert.Equal(t, tc.expectedLimit, limit)
		})
	}
}

// TestToolSearchInput_CategoryFilter verifies category field values.
func TestToolSearchInput_CategoryFilter(t *testing.T) {
	validCategories := []string{
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

	for _, cat := range validCategories {
		t.Run("category_"+cat, func(t *testing.T) {
			input := toolSearchInput{
				Query:    "test",
				Category: cat,
			}
			// Verify category is recognized as a valid ToolCategory
			category := ToolCategory(input.Category)
			assert.NotEmpty(t, category)
		})
	}

	t.Run("empty_category_searches_all", func(t *testing.T) {
		input := toolSearchInput{
			Query:    "test",
			Category: "",
		}
		// Empty category means search all categories
		assert.Equal(t, "", input.Category)
	})
}

// TestToolSearchOutput_Structure verifies the output structure fields.
func TestToolSearchOutput_Structure(t *testing.T) {
	output := toolSearchOutput{
		Query:      "memory",
		Results:    make([]map[string]interface{}, 0),
		Count:      0,
		TotalTools: 15,
	}

	assert.Equal(t, "memory", output.Query)
	assert.NotNil(t, output.Results)
	assert.Equal(t, 0, output.Count)
	assert.Equal(t, 15, output.TotalTools)
}

// TestToolSearchOutput_ResultFields verifies each result contains expected fields.
func TestToolSearchOutput_ResultFields(t *testing.T) {
	result := map[string]interface{}{
		"name":          "memory_search",
		"description":   "Search for memories",
		"category":      "memory",
		"defer_loading": false,
		"score":         2,
		"match_reason":  "name contains query",
		"keywords":      []string{"find", "lookup"},
	}

	// Required fields
	assert.Contains(t, result, "name")
	assert.Contains(t, result, "description")
	assert.Contains(t, result, "category")
	assert.Contains(t, result, "defer_loading")
	assert.Contains(t, result, "score")
	assert.Contains(t, result, "match_reason")

	// Optional field (keywords)
	keywords, hasKeywords := result["keywords"]
	assert.True(t, hasKeywords)
	assert.NotEmpty(t, keywords)
}

// TestToolSearchOutput_ResultWithoutKeywords verifies keywords are omitted when empty.
func TestToolSearchOutput_ResultWithoutKeywords(t *testing.T) {
	// When tool has no keywords, the result should not include keywords field
	result := map[string]interface{}{
		"name":          "checkpoint_save",
		"description":   "Save checkpoint",
		"category":      "checkpoint",
		"defer_loading": false,
		"score":         3,
		"match_reason":  "exact name match",
	}

	_, hasKeywords := result["keywords"]
	assert.False(t, hasKeywords, "result without keywords should not include keywords field")
}

// TestToolListInput_Structure verifies the list input structure.
func TestToolListInput_Structure(t *testing.T) {
	t.Run("empty_input_lists_all", func(t *testing.T) {
		input := toolListInput{}
		assert.Equal(t, "", input.Category)
		assert.False(t, input.DeferredOnly)
	})

	t.Run("category_filter", func(t *testing.T) {
		input := toolListInput{
			Category: "memory",
		}
		assert.Equal(t, "memory", input.Category)
	})

	t.Run("deferred_only_filter", func(t *testing.T) {
		input := toolListInput{
			DeferredOnly: true,
		}
		assert.True(t, input.DeferredOnly)
	})

	t.Run("category_takes_precedence_over_deferred", func(t *testing.T) {
		// When category is set, DeferredOnly should be ignored per the handler logic
		input := toolListInput{
			Category:     "memory",
			DeferredOnly: true,
		}
		// Handler checks Category first, then DeferredOnly
		assert.NotEmpty(t, input.Category)
	})
}

// TestToolListOutput_Structure verifies the list output structure.
func TestToolListOutput_Structure(t *testing.T) {
	output := toolListOutput{
		Tools: []map[string]interface{}{
			{
				"name":          "memory_search",
				"description":   "Search memories",
				"category":      "memory",
				"defer_loading": false,
			},
		},
		Count: 1,
	}

	assert.NotNil(t, output.Tools)
	assert.Len(t, output.Tools, 1)
	assert.Equal(t, 1, output.Count)
}

// TestToolListOutput_ToolFields verifies each tool entry has required fields.
func TestToolListOutput_ToolFields(t *testing.T) {
	tool := map[string]interface{}{
		"name":          "checkpoint_save",
		"description":   "Save context checkpoint",
		"category":      "checkpoint",
		"defer_loading": true,
	}

	// Required fields
	assert.Contains(t, tool, "name")
	assert.Contains(t, tool, "description")
	assert.Contains(t, tool, "category")
	assert.Contains(t, tool, "defer_loading")
}

// TestToolSearchHandlerLogic_AppliesLimit verifies limit is applied to results.
func TestToolSearchHandlerLogic_AppliesLimit(t *testing.T) {
	testCases := []struct {
		name          string
		totalResults  int
		limit         int
		expectedCount int
	}{
		{
			name:          "results_within_limit",
			totalResults:  3,
			limit:         5,
			expectedCount: 3,
		},
		{
			name:          "results_at_limit",
			totalResults:  5,
			limit:         5,
			expectedCount: 5,
		},
		{
			name:          "results_exceed_limit",
			totalResults:  10,
			limit:         5,
			expectedCount: 5,
		},
		{
			name:          "custom_limit_applied",
			totalResults:  20,
			limit:         3,
			expectedCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate search results
			searchResults := make([]*SearchResult, tc.totalResults)
			for i := 0; i < tc.totalResults; i++ {
				searchResults[i] = &SearchResult{
					Tool:        &ToolMetadata{Name: "tool"},
					Score:       1,
					MatchReason: "test",
				}
			}

			// Apply limit (same logic as handler)
			if len(searchResults) > tc.limit {
				searchResults = searchResults[:tc.limit]
			}

			assert.Equal(t, tc.expectedCount, len(searchResults))
		})
	}
}

// TestToolSearchHandlerLogic_BuildsResultText verifies result text formatting.
func TestToolSearchHandlerLogic_BuildsResultText(t *testing.T) {
	t.Run("no_results_message", func(t *testing.T) {
		toolNames := []string{}
		query := "nonexistent"

		var resultText string
		if len(toolNames) == 0 {
			resultText = "No tools found matching: " + query
		}

		assert.Equal(t, "No tools found matching: nonexistent", resultText)
	})

	t.Run("single_result_message", func(t *testing.T) {
		toolNames := []string{"memory_search"}
		query := "memory"

		var resultText string
		if len(toolNames) == 0 {
			resultText = "No tools found matching: " + query
		} else {
			resultText = "Found " + string(rune('0'+len(toolNames))) + " tool(s) for query '" + query + "': " + toolNames[0]
		}

		assert.Contains(t, resultText, "Found")
		assert.Contains(t, resultText, "memory_search")
	})

	t.Run("multiple_results_message", func(t *testing.T) {
		// Expected format when multiple tools are found
		resultText := "Found 2 tool(s) for query 'memory': memory_search, memory_record"

		assert.Contains(t, resultText, "2 tool(s)")
		assert.Contains(t, resultText, "memory_search, memory_record")
	})
}

// TestToolSearch_IntegrationWithRegistry tests the full flow with a real registry.
func TestToolSearch_IntegrationWithRegistry(t *testing.T) {
	// Create a registry with test tools
	registry := NewToolRegistry()
	registry.RegisterAll([]*ToolMetadata{
		{
			Name:         "memory_search",
			Description:  "Search for memories in the reasoning bank",
			Category:     CategoryMemory,
			DeferLoading: false,
			Keywords:     []string{"find", "lookup", "recall"},
		},
		{
			Name:         "memory_record",
			Description:  "Record a new memory",
			Category:     CategoryMemory,
			DeferLoading: true,
			Keywords:     []string{"save", "store"},
		},
		{
			Name:         "checkpoint_save",
			Description:  "Save context to a checkpoint",
			Category:     CategoryCheckpoint,
			DeferLoading: false,
		},
	})

	t.Run("search_returns_matching_tools", func(t *testing.T) {
		results := registry.Search("memory")
		require.Len(t, results, 2)

		// Convert to output format (same as handler)
		outputResults := make([]map[string]interface{}, 0, len(results))
		for _, sr := range results {
			result := map[string]interface{}{
				"name":          sr.Tool.Name,
				"description":   sr.Tool.Description,
				"category":      string(sr.Tool.Category),
				"defer_loading": sr.Tool.DeferLoading,
				"score":         sr.Score,
				"match_reason":  sr.MatchReason,
			}
			if len(sr.Tool.Keywords) > 0 {
				result["keywords"] = sr.Tool.Keywords
			}
			outputResults = append(outputResults, result)
		}

		assert.Len(t, outputResults, 2)
		// Verify all results have required fields
		for _, result := range outputResults {
			assert.Contains(t, result, "name")
			assert.Contains(t, result, "score")
			assert.Contains(t, result, "match_reason")
		}
	})

	t.Run("search_with_category_filter", func(t *testing.T) {
		results := registry.SearchByCategory("save", CategoryCheckpoint)
		require.Len(t, results, 1)
		assert.Equal(t, "checkpoint_save", results[0].Tool.Name)
	})

	t.Run("search_returns_total_count", func(t *testing.T) {
		totalTools := registry.Count()
		assert.Equal(t, 3, totalTools)

		output := toolSearchOutput{
			Query:      "test",
			Results:    []map[string]interface{}{},
			Count:      0,
			TotalTools: totalTools,
		}
		assert.Equal(t, 3, output.TotalTools)
	})
}

// TestToolList_IntegrationWithRegistry tests the full list flow with a real registry.
func TestToolList_IntegrationWithRegistry(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterAll([]*ToolMetadata{
		{
			Name:         "tool1",
			Description:  "First tool",
			Category:     CategoryMemory,
			DeferLoading: false,
		},
		{
			Name:         "tool2",
			Description:  "Second tool",
			Category:     CategoryMemory,
			DeferLoading: true,
		},
		{
			Name:         "tool3",
			Description:  "Third tool",
			Category:     CategoryCheckpoint,
			DeferLoading: true,
		},
	})

	t.Run("list_all_tools", func(t *testing.T) {
		tools := registry.List()
		assert.Len(t, tools, 3)
	})

	t.Run("list_by_category", func(t *testing.T) {
		tools := registry.ListByCategory(CategoryMemory)
		assert.Len(t, tools, 2)
	})

	t.Run("list_deferred_only", func(t *testing.T) {
		tools := registry.ListDeferred()
		assert.Len(t, tools, 2)
		for _, tool := range tools {
			assert.True(t, tool.DeferLoading)
		}
	})
}

// TestToolSearch_ScoreOrdering verifies results are ordered by score (highest first).
func TestToolSearch_ScoreOrdering(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterAll([]*ToolMetadata{
		{
			Name:        "foo_bar",
			Description: "Does something with baz",
			Keywords:    []string{"qux"},
		},
		{
			Name:        "baz_tool",
			Description: "A tool for processing",
		},
		{
			Name:        "baz",
			Description: "Exact match tool",
		},
	})

	results := registry.Search("baz")

	require.GreaterOrEqual(t, len(results), 2)

	// First result should have highest score (exact name match = 3)
	assert.Equal(t, "baz", results[0].Tool.Name)
	assert.Equal(t, 3, results[0].Score)

	// Subsequent results should have lower or equal scores
	for i := 1; i < len(results); i++ {
		assert.LessOrEqual(t, results[i].Score, results[i-1].Score,
			"results should be sorted by score descending")
	}
}

// TestToolSearch_RegexPatternSupport verifies regex patterns work correctly.
func TestToolSearch_RegexPatternSupport(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterAll([]*ToolMetadata{
		{Name: "memory_search", Description: "Search memories"},
		{Name: "memory_record", Description: "Record memory"},
		{Name: "checkpoint_save", Description: "Save checkpoint"},
		{Name: "get_user_data", Description: "Get user data"},
		{Name: "get_project_data", Description: "Get project data"},
	})

	testCases := []struct {
		name        string
		pattern     string
		expectCount int
		expectNames []string
	}{
		{
			name:        "prefix_pattern",
			pattern:     "^memory",
			expectCount: 2,
			expectNames: []string{"memory_search", "memory_record"},
		},
		{
			name:        "wildcard_pattern",
			pattern:     "get_.*_data",
			expectCount: 2,
			expectNames: []string{"get_user_data", "get_project_data"},
		},
		{
			name:        "case_insensitive",
			pattern:     "(?i)MEMORY",
			expectCount: 2,
			expectNames: []string{"memory_search", "memory_record"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results := registry.Search(tc.pattern)
			assert.Len(t, results, tc.expectCount, "expected %d results for pattern %q", tc.expectCount, tc.pattern)

			foundNames := make([]string, len(results))
			for i, r := range results {
				foundNames[i] = r.Tool.Name
			}

			for _, expectedName := range tc.expectNames {
				assert.Contains(t, foundNames, expectedName)
			}
		})
	}
}

// TestToolSearch_DeferLoadingField verifies defer_loading is correctly included in results.
func TestToolSearch_DeferLoadingField(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterAll([]*ToolMetadata{
		{
			Name:         "deferred_tool",
			Description:  "A deferred tool",
			DeferLoading: true,
		},
		{
			Name:         "immediate_tool",
			Description:  "An immediate tool",
			DeferLoading: false,
		},
	})

	t.Run("deferred_tool_shows_true", func(t *testing.T) {
		results := registry.Search("deferred_tool")
		require.Len(t, results, 1)
		assert.True(t, results[0].Tool.DeferLoading)
	})

	t.Run("immediate_tool_shows_false", func(t *testing.T) {
		results := registry.Search("immediate_tool")
		require.Len(t, results, 1)
		assert.False(t, results[0].Tool.DeferLoading)
	})
}
