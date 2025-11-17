# Tool Composition Framework Specification

## Overview

The Tool Composition Framework enables complex workflows by allowing MCP tools to be chained together in sequences with dependency resolution, error recovery, and rollback capabilities. This framework supports the creation of composite patterns that can solve multi-step problems efficiently.

## Goals

- **Enable Complex Workflows**: Support 3+ tool chains (e.g., search → fold → store)
- **Performance**: <200ms overhead for composition execution
- **Reliability**: 95% success rate for valid compositions
- **Developer Experience**: Simple JSON-based DSL for composition definition

## Architecture

### Core Components

1. **Composition DSL**: JSON-based language for defining tool sequences
2. **Execution Engine**: Runtime that orchestrates tool execution with dependency resolution
3. **Error Recovery**: Rollback and retry mechanisms for failed compositions
4. **Template Library**: Pre-built composition templates for common workflows

### Composition Structure

```json
{
  "name": "search-fold-store-workflow",
  "description": "Search for context, fold it, then store the result",
  "version": "1.0.0",
  "steps": [
    {
      "id": "search",
      "tool": "checkpoint_search",
      "parameters": {
        "query": "{{input.query}}",
        "top_k": 10
      },
      "output_key": "search_results"
    },
    {
      "id": "fold",
      "tool": "context_fold",
      "parameters": {
        "content": "{{search_results.content}}",
        "strategy": "abstractive",
        "target_ratio": 0.6
      },
      "depends_on": ["search"],
      "output_key": "folded_content"
    },
    {
      "id": "store",
      "tool": "checkpoint_save",
      "parameters": {
        "summary": "{{folded_content.summary}}",
        "content": "{{folded_content.folded}}",
        "tags": ["folded", "auto-generated"]
      },
      "depends_on": ["fold"],
      "output_key": "saved_checkpoint"
    }
  ],
  "error_handling": {
    "max_retries": 2,
    "rollback_on_failure": true,
    "failure_strategy": "partial_success"
  },
  "metadata": {
    "author": "contextd",
    "tags": ["search", "folding", "storage"],
    "estimated_duration_ms": 500
  }
}
```

## DSL Specification

### Step Definition

Each step in a composition represents a single tool execution:

```json
{
  "id": "unique_step_identifier",
  "tool": "tool_name",
  "parameters": {
    "param1": "value1",
    "param2": "{{variable_reference}}"
  },
  "depends_on": ["step_id_1", "step_id_2"],
  "output_key": "result_variable_name",
  "timeout_ms": 5000,
  "retry_count": 1
}
```

### Parameter Interpolation

Parameters support variable interpolation using `{{variable}}` syntax:

- `{{input.param}}`: Reference input parameters
- `{{step_id.output_key}}`: Reference outputs from other steps
- `{{context.user_id}}`: Reference context variables

### Dependency Resolution

The `depends_on` array specifies execution order:

```json
{
  "steps": [
    {
      "id": "step1",
      "tool": "tool_a"
    },
    {
      "id": "step2",
      "tool": "tool_b",
      "depends_on": ["step1"]
    },
    {
      "id": "step3",
      "tool": "tool_c",
      "depends_on": ["step1", "step2"]
    }
  ]
}
```

This creates a DAG (Directed Acyclic Graph) where:
- `step1` executes first
- `step2` executes after `step1` completes
- `step3` executes after both `step1` and `step2` complete

## Execution Engine

### Execution Flow

1. **Validation**: Validate composition structure and dependencies
2. **Planning**: Build execution plan with topological sort
3. **Execution**: Execute steps in dependency order
4. **Error Handling**: Apply error recovery strategies on failure
5. **Result Aggregation**: Collect and return final results

### Error Handling Strategies

- **fail_fast**: Stop execution on first error
- **partial_success**: Continue with remaining steps, return partial results
- **rollback**: Undo completed steps on failure
- **retry**: Retry failed steps with exponential backoff

### Rollback Mechanism

For compositions with `rollback_on_failure: true`:

1. Track completed steps in reverse order
2. Execute rollback actions for each completed step
3. Rollback actions are tool-specific (e.g., delete created checkpoint)

## Template Library

### Template Structure

Templates are stored compositions with metadata:

```json
{
  "id": "search-fold-store-v1",
  "name": "Search, Fold, and Store",
  "description": "Complete workflow for context optimization",
  "composition": { /* full composition object */ },
  "metadata": {
    "category": "context-management",
    "tags": ["search", "folding", "storage"],
    "estimated_duration_ms": 500,
    "success_rate": 0.95,
    "usage_count": 42
  },
  "validation": {
    "required_tools": ["checkpoint_search", "context_fold", "checkpoint_save"],
    "min_version": "1.0.0"
  }
}
```

### Template Categories

- **Context Management**: Search, fold, store workflows
- **Analysis**: Multi-step analysis and reporting
- **Automation**: Repetitive task automation
- **Integration**: Cross-system data synchronization

## API Design

### Composition Execution

```go
type CompositionRequest struct {
    TemplateID string                 `json:"template_id,omitempty"`
    Composition *Composition         `json:"composition,omitempty"`
    Input       map[string]interface{} `json:"input"`
    Options     ExecutionOptions      `json:"options,omitempty"`
}

type ExecutionOptions struct {
    Timeout         time.Duration     `json:"timeout,omitempty"`
    MaxConcurrency  int              `json:"max_concurrency,omitempty"`
    ErrorStrategy   ErrorStrategy    `json:"error_strategy,omitempty"`
    DryRun         bool             `json:"dry_run,omitempty"`
}

type CompositionResult struct {
    Success     bool                   `json:"success"`
    Results     map[string]interface{} `json:"results"`
    Errors      []ExecutionError       `json:"errors,omitempty"`
    Duration    time.Duration          `json:"duration"`
    StepsExecuted int                 `json:"steps_executed"`
}
```

### Template Management

```go
// List available templates
GET /api/v1/compositions/templates

// Get specific template
GET /api/v1/compositions/templates/{id}

// Execute composition
POST /api/v1/compositions/execute

// Validate composition
POST /api/v1/compositions/validate
```

## Implementation Plan

### Phase 1: Core DSL and Engine (Week 1)

1. Define Composition and Step structs
2. Implement dependency resolution algorithm
3. Create basic execution engine
4. Add error handling framework

### Phase 2: Advanced Features (Week 1.5)

1. Implement rollback mechanisms
2. Add parameter interpolation
3. Create template storage system
4. Add performance monitoring

### Phase 3: Templates and Testing (Week 2)

1. Create initial template library (10+ templates)
2. Add comprehensive validation
3. Performance optimization
4. Integration testing

## Success Criteria

- [ ] Support 3+ tool chains in single composition
- [ ] <200ms overhead for composition execution
- [ ] 95% success rate for valid compositions
- [ ] 10+ templates in library
- [ ] Full error recovery and rollback support
- [ ] Comprehensive test coverage (>80%)

## Dependencies

- MCP tool system (for tool execution)
- Configuration system (for template storage)
- Metrics system (for performance monitoring)

## Future Enhancements

- **Visual Composition Builder**: Web UI for creating compositions
- **Dynamic Dependencies**: Runtime dependency resolution
- **Composition Optimization**: Automatic parallelization
- **Learning System**: Success rate tracking and optimization</content>
</xai:function_call: write_file>
<parameter name="filePath">docs/specs/tool-composition/SPEC.md
