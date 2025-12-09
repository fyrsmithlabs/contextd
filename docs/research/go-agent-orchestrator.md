# Research: Go Agent Orchestrator for Contextd

**Issue**: [#20](https://github.com/fyrsmithlabs/contextd/issues/20)
**Date**: 2025-12-09
**Status**: Research Complete

---

## Problem Statement

Sub-agents exhibit unreliable workflow adherence:
- Test and implementation bundling (violating TDD principles)
- Using `--help` as a substitute for actual testing
- Skipping workflow phases without documentation
- Minimal progress reporting

---

## Available Go Libraries

### 1. Official Anthropic Go SDK
**Repository**: [anthropics/anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go)

| Aspect | Details |
|--------|---------|
| Version | v1.19.0 |
| Go Version | 1.22+ |
| Status | Official, Production |
| Agent Support | None (Messages API only) |

**Features**:
- Messages API with streaming
- Tool calling with JSON schemas
- Multi-turn conversations
- System prompts

**Limitations**:
- No agent loop implementation
- No MCP integration
- Requires building orchestration from scratch

### 2. Ingenimax/agent-sdk-go
**Repository**: [Ingenimax/agent-sdk-go](https://github.com/Ingenimax/agent-sdk-go)

| Aspect | Details |
|--------|---------|
| Status | Third-party, Active |
| Multi-model | OpenAI, Anthropic, Google |
| MCP Support | Yes (HTTP + stdio) |
| Agent Loop | Yes |

**Features**:
- Multi-model support (OpenAI, Anthropic, Gemini)
- MCP server integration (HTTP and stdio)
- Sub-agent hierarchies
- Task-based orchestration with YAML
- Token usage tracking
- Enterprise multi-tenancy
- Built-in guardrails

**Trade-offs**:
- Third-party with ongoing maintenance question
- Multi-model abstraction adds complexity
- May be overkill for Claude-only use case

### 3. schlunsen/claude-agent-sdk-go
**Repository**: [schlunsen/claude-agent-sdk-go](https://github.com/schlunsen/claude-agent-sdk-go)

| Aspect | Details |
|--------|---------|
| Version | v0.1.0 |
| Go Version | 1.24+ |
| Status | Unofficial port, Production Ready |
| MCP Support | Yes |

**Features**:
- One-shot queries and interactive sessions
- Permission callbacks for tool approval
- Hook system (PreToolUse, PostToolUse)
- MCP server support
- Full streaming
- Zero stdlib dependencies
- Idiomatic Go (goroutines, channels)

**Key APIs**:
```go
// One-shot query
Query(ctx, prompt, options) <-chan Message

// Interactive session
client.Connect(ctx)
client.Query(ctx, prompt)
client.ReceiveResponse(ctx)
```

**Trade-offs**:
- Wraps Claude Code CLI (subprocess management)
- Unofficial - no Anthropic support
- Recent release (October 2025)

### 4. mark3labs/mcp-go
**Repository**: [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)

| Aspect | Details |
|--------|---------|
| Status | Third-party, Active |
| MCP Spec | Full implementation |
| Transports | stdio, SSE, HTTP |

**Features**:
- Complete MCP server/client implementation
- Tools, Resources, and Prompts support
- Session management
- Request hooks/middleware
- Type-safe argument handling

**Role**: Foundation for MCP integration, not an agent SDK itself.

---

## Orchestration Approach Analysis

### Option 1: Direct API (Official SDK)

```
┌─────────────────────────────────────────┐
│         Go Orchestrator                 │
│  ┌─────────────────────────────────┐    │
│  │    anthropic-sdk-go (official)  │    │
│  │    - Messages API               │    │
│  │    - Tool calling               │    │
│  │    - Streaming                  │    │
│  └─────────────────────────────────┘    │
│                  │                       │
│  ┌─────────────────────────────────┐    │
│  │    Custom Agent Loop            │    │
│  │    - Phase gates                │    │
│  │    - State machine              │    │
│  │    - Guardrails                 │    │
│  └─────────────────────────────────┘    │
│                  │                       │
│  ┌─────────────────────────────────┐    │
│  │    contextd MCP Client          │    │
│  │    (mark3labs/mcp-go)           │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
```

**Pros**:
- Full control over agent behavior
- Official SDK stability
- Custom phase gate implementation
- Tight contextd integration

**Cons**:
- Significant implementation effort
- Must build agent loop from scratch
- Tool execution management required

**Recommendation**: ⭐⭐⭐⭐⭐ **Best Option**

### Option 2: Claude Code CLI Wrapper

```
┌─────────────────────────────────────────┐
│         Go Orchestrator                 │
│  ┌─────────────────────────────────┐    │
│  │  schlunsen/claude-agent-sdk-go  │    │
│  │    - CLI subprocess             │    │
│  │    - Hook system                │    │
│  │    - Permission callbacks       │    │
│  └─────────────────────────────────┘    │
│                  │                       │
│  ┌─────────────────────────────────┐    │
│  │    Phase Gate Middleware        │    │
│  │    - PreToolUse hooks           │    │
│  │    - Tool filtering             │    │
│  └─────────────────────────────────┘    │
│                  │                       │
│  ┌─────────────────────────────────┐    │
│  │    contextd (via MCP)           │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
```

**Pros**:
- Leverages existing Claude Code capabilities
- Built-in tool management
- Hook system for interception
- Zero stdlib dependencies

**Cons**:
- Subprocess management complexity
- Claude Code CLI must be installed
- Unofficial - may break with CLI updates
- Less control over agent behavior

**Recommendation**: ⭐⭐⭐ Good for rapid prototyping

### Option 3: Hybrid (Ingenimax SDK)

```
┌─────────────────────────────────────────┐
│         Go Orchestrator                 │
│  ┌─────────────────────────────────┐    │
│  │    Ingenimax/agent-sdk-go       │    │
│  │    - Agent framework            │    │
│  │    - MCP integration            │    │
│  │    - Multi-model                │    │
│  └─────────────────────────────────┘    │
│                  │                       │
│  ┌─────────────────────────────────┐    │
│  │    Custom Task Definitions      │    │
│  │    - YAML workflows             │    │
│  │    - Phase constraints          │    │
│  └─────────────────────────────────┘    │
│                  │                       │
│  ┌─────────────────────────────────┐    │
│  │    contextd MCP Server          │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
```

**Pros**:
- Agent loop already implemented
- MCP support built-in
- Sub-agent support
- Enterprise features

**Cons**:
- Third-party dependency
- Multi-model abstraction overhead
- Less customization flexibility

**Recommendation**: ⭐⭐⭐⭐ Good balance of effort/features

---

## Recommended Architecture

### Primary Recommendation: Option 1 (Direct API)

Build a custom orchestrator using the official Anthropic SDK with contextd integration:

```
cmd/orchestrator/
├── main.go                    # Entry point
├── agent/
│   ├── loop.go                # Agent execution loop
│   ├── state.go               # State machine
│   └── message.go             # Message handling
├── phase/
│   ├── gate.go                # Phase gate implementation
│   ├── tdd.go                 # TDD enforcement
│   └── verification.go        # Output verification
├── tools/
│   ├── registry.go            # Tool registry
│   ├── executor.go            # Tool execution
│   └── contextd.go            # contextd MCP client
└── guardrails/
    ├── compliance.go          # Workflow compliance
    ├── reporting.go           # Progress reporting
    └── audit.go               # Decision logging
```

### Agent State Machine

```
┌───────────────────────────────────────────────────────────────┐
│                       AGENT STATES                            │
├───────────────────────────────────────────────────────────────┤
│                                                               │
│   ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐   │
│   │  INIT   │───▶│ ANALYZE │───▶│  PLAN   │───▶│  TEST   │   │
│   └─────────┘    └─────────┘    └─────────┘    └─────────┘   │
│                                                     │         │
│                                                     ▼         │
│   ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐   │
│   │COMPLETE │◀───│ VERIFY  │◀───│ COMMIT  │◀───│IMPLEMENT│   │
│   └─────────┘    └─────────┘    └─────────┘    └─────────┘   │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

### Phase Gate Interface

```go
// phase/gate.go
package phase

type Phase string

const (
    PhaseInit      Phase = "init"
    PhaseAnalyze   Phase = "analyze"
    PhasePlan      Phase = "plan"
    PhaseTest      Phase = "test"      // TDD: tests first
    PhaseImplement Phase = "implement"
    PhaseCommit    Phase = "commit"
    PhaseVerify    Phase = "verify"
    PhaseComplete  Phase = "complete"
)

type Gate interface {
    // CurrentPhase returns the current workflow phase
    CurrentPhase() Phase

    // CanTransition checks if transition to target phase is allowed
    CanTransition(target Phase) bool

    // Transition moves to the target phase if allowed
    Transition(ctx context.Context, target Phase) error

    // Requirements returns unfulfilled requirements for transition
    Requirements(target Phase) []Requirement
}

type Requirement struct {
    Type        string // "test_exists", "tests_pass", "commit_message"
    Description string
    Satisfied   bool
}
```

### TDD Enforcement

```go
// phase/tdd.go
package phase

type TDDEnforcer struct {
    gate     Gate
    memory   *contextd.MemoryClient
    registry *tools.Registry
}

func (e *TDDEnforcer) BeforeImplementation(ctx context.Context) error {
    // Verify we're in TEST phase with passing tests before IMPLEMENT
    if e.gate.CurrentPhase() != PhaseTest {
        return fmt.Errorf("TDD violation: must write tests before implementation")
    }

    // Check tests exist
    reqs := e.gate.Requirements(PhaseImplement)
    for _, req := range reqs {
        if req.Type == "test_exists" && !req.Satisfied {
            return fmt.Errorf("TDD violation: no tests found")
        }
    }

    // Record compliance in contextd memory
    e.memory.Record(ctx, &contextd.Memory{
        Title:      "TDD Compliance",
        Content:    "Tests written before implementation",
        Confidence: 1.0,
        Tags:       []string{"tdd", "compliance"},
    })

    return nil
}
```

### Contextd Integration

```go
// tools/contextd.go
package tools

import (
    "context"
    "github.com/mark3labs/mcp-go/client"
)

type ContextdClient struct {
    mcpClient *client.Client
    projectID string
    sessionID string
}

func NewContextdClient(socketPath string) (*ContextdClient, error) {
    // Connect via MCP stdio or HTTP
    c, err := client.NewStdioClient("contextd", []string{"--mcp"})
    if err != nil {
        return nil, err
    }
    return &ContextdClient{mcpClient: c}, nil
}

// Memory operations for audit trail
func (c *ContextdClient) RecordDecision(ctx context.Context, decision Decision) error {
    return c.mcpClient.CallTool(ctx, "memory_record", map[string]interface{}{
        "project_id": c.projectID,
        "title":      decision.Title,
        "content":    decision.Rationale,
        "confidence": decision.Confidence,
        "tags":       decision.Tags,
    })
}

// Checkpoint for phase transitions
func (c *ContextdClient) SavePhaseCheckpoint(ctx context.Context, phase Phase) error {
    return c.mcpClient.CallTool(ctx, "checkpoint_save", map[string]interface{}{
        "tenant_id":  c.projectID,
        "session_id": c.sessionID,
        "summary":    fmt.Sprintf("Phase: %s completed", phase),
    })
}

// Search for relevant past decisions
func (c *ContextdClient) SearchPriorDecisions(ctx context.Context, query string) ([]Memory, error) {
    result, err := c.mcpClient.CallTool(ctx, "memory_search", map[string]interface{}{
        "project_id": c.projectID,
        "query":      query,
        "limit":      5,
    })
    // Parse and return memories
}
```

### Agent Loop Implementation

```go
// agent/loop.go
package agent

import (
    "context"
    "github.com/anthropics/anthropic-sdk-go"
)

type Agent struct {
    client    *anthropic.Client
    contextd  *tools.ContextdClient
    gate      *phase.Gate
    guardrail *guardrails.Compliance
    tools     *tools.Registry
}

func (a *Agent) Run(ctx context.Context, task string) error {
    // Initialize session
    a.contextd.RecordDecision(ctx, Decision{
        Title:   "Task Started",
        Content: task,
    })

    messages := []anthropic.MessageParam{
        anthropic.NewUserMessage(anthropic.NewTextBlock(task)),
    }

    for {
        // Get model response
        resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
            Model:     anthropic.ModelClaudeSonnet4_5_20250929,
            MaxTokens: 4096,
            Messages:  messages,
            Tools:     a.tools.Definitions(),
            System:    a.buildSystemPrompt(),
        })
        if err != nil {
            return err
        }

        // Process response
        for _, block := range resp.Content {
            switch b := block.(type) {
            case anthropic.TextBlock:
                // Report progress
                a.guardrail.ReportProgress(ctx, b.Text)

            case anthropic.ToolUseBlock:
                // Check phase gate before tool execution
                if err := a.gate.ValidateTool(b.Name); err != nil {
                    // Tool not allowed in current phase
                    a.contextd.RecordDecision(ctx, Decision{
                        Title:   "Tool Blocked",
                        Content: fmt.Sprintf("Tool %s blocked: %v", b.Name, err),
                    })
                    continue
                }

                // Execute tool
                result, err := a.tools.Execute(ctx, b.Name, b.Input)
                if err != nil {
                    a.contextd.RecordRemediation(ctx, err)
                }

                // Append tool result
                messages = append(messages,
                    anthropic.NewAssistantMessage(resp.Content...),
                    anthropic.NewUserMessage(
                        anthropic.NewToolResultBlock(b.ID, result, err != nil),
                    ),
                )
            }
        }

        // Check for phase transition
        if a.gate.ShouldTransition() {
            a.contextd.SavePhaseCheckpoint(ctx, a.gate.CurrentPhase())
            a.gate.Transition(ctx, a.gate.NextPhase())
        }

        // Check completion
        if resp.StopReason == "end_turn" && a.gate.CurrentPhase() == phase.PhaseComplete {
            break
        }
    }

    return nil
}

func (a *Agent) buildSystemPrompt() []anthropic.TextBlockParam {
    return []anthropic.TextBlockParam{
        anthropic.NewTextBlock(fmt.Sprintf(`You are a workflow-compliant agent.

Current Phase: %s
Allowed Tools: %v
Requirements for next phase: %v

CRITICAL RULES:
1. TDD: Always write tests BEFORE implementation
2. Never skip phases without explicit documentation
3. Report progress at each step
4. Commit tests and implementation separately
`,
            a.gate.CurrentPhase(),
            a.gate.AllowedTools(),
            a.gate.Requirements(a.gate.NextPhase()),
        )),
    }
}
```

---

## Specialized Agent Roles

### 1. Memory Agent

**Purpose**: Search and record operations using contextd's ReasoningBank

```go
type MemoryAgent struct {
    contextd *tools.ContextdClient
}

func (a *MemoryAgent) Tools() []Tool {
    return []Tool{
        {Name: "memory_search", Handler: a.search},
        {Name: "memory_record", Handler: a.record},
        {Name: "memory_feedback", Handler: a.feedback},
    }
}
```

### 2. Remediation Agent

**Purpose**: Error diagnosis and fix pattern lookup

```go
type RemediationAgent struct {
    contextd *tools.ContextdClient
}

func (a *RemediationAgent) Tools() []Tool {
    return []Tool{
        {Name: "remediation_search", Handler: a.searchFixes},
        {Name: "remediation_record", Handler: a.recordFix},
        {Name: "troubleshoot_diagnose", Handler: a.diagnose},
    }
}
```

### 3. Task Runner Agent

**Purpose**: Guardrailed execution with phase gates

```go
type TaskRunner struct {
    agent     *Agent
    gate      *phase.Gate
    guardrail *guardrails.Compliance
}

func (t *TaskRunner) Execute(ctx context.Context, task Task) (*Result, error) {
    // Set up phase gates based on task type
    t.gate.Configure(task.Type)

    // Execute with enforcement
    return t.agent.Run(ctx, task.Description)
}
```

---

## Guardrail Implementation

### Phase-Allowed Tools

```go
var phaseTools = map[Phase][]string{
    PhaseInit:      {"memory_search", "repository_search"},
    PhaseAnalyze:   {"memory_search", "repository_search", "Read", "Glob"},
    PhasePlan:      {"memory_record", "checkpoint_save"},
    PhaseTest:      {"Write", "Bash"}, // Test files only
    PhaseImplement: {"Write", "Edit", "Bash"},
    PhaseCommit:    {"Bash"}, // git only
    PhaseVerify:    {"Bash", "Read"}, // test run, review
    PhaseComplete:  {"memory_record", "checkpoint_save"},
}
```

### Compliance Checking

```go
type Compliance struct {
    violations []Violation
    reporter   Reporter
}

func (c *Compliance) CheckTDDCompliance(ctx context.Context, phase Phase, tool string) error {
    if phase == PhaseImplement {
        // Check if tests exist and pass
        if !c.testsExist() {
            c.recordViolation(Violation{
                Type:    "TDD",
                Message: "Implementation started without tests",
                Phase:   phase,
            })
            return ErrTDDViolation
        }
    }
    return nil
}

func (c *Compliance) ReportProgress(ctx context.Context, message string) {
    c.reporter.Report(ProgressReport{
        Phase:    c.currentPhase,
        Message:  message,
        Timestamp: time.Now(),
    })
}
```

---

## Integration with Existing contextd Architecture

The orchestrator integrates cleanly with contextd's existing patterns:

### Service Registry Integration

```go
// Orchestrator can use the same registry pattern
type OrchestratorRegistry interface {
    services.Registry  // Embed contextd's registry

    // Additional orchestrator-specific services
    PhaseGate() *phase.Gate
    Compliance() *guardrails.Compliance
    AgentLoop() *agent.Agent
}
```

### Hook Integration

```go
// Register orchestrator hooks with contextd
func (o *Orchestrator) RegisterHooks(hookMgr *hooks.HookManager) {
    hookMgr.RegisterHandler(hooks.HookSessionStart, o.onSessionStart)
    hookMgr.RegisterHandler(hooks.HookSessionEnd, o.onSessionEnd)
    hookMgr.RegisterHandler(hooks.HookContextThreshold, o.onThreshold)
}

func (o *Orchestrator) onSessionStart(ctx context.Context, data map[string]interface{}) error {
    // Prime with relevant memories for this task type
    memories, _ := o.contextd.SearchPriorDecisions(ctx, o.currentTask)
    o.agent.PrimeContext(memories)
    return nil
}
```

---

## Implementation Phases

### Phase 1: Core Agent Loop (Week 1-2)
- [ ] Set up project structure
- [ ] Implement basic agent loop with official SDK
- [ ] Add streaming support
- [ ] Implement tool registry

### Phase 2: Phase Gates (Week 2-3)
- [ ] Implement state machine
- [ ] Add phase-based tool filtering
- [ ] Implement TDD enforcement
- [ ] Add requirement checking

### Phase 3: contextd Integration (Week 3-4)
- [ ] Implement MCP client using mark3labs/mcp-go
- [ ] Add memory operations for audit trail
- [ ] Integrate checkpoint save on phase transitions
- [ ] Add remediation search for error handling

### Phase 4: Guardrails & Reporting (Week 4)
- [ ] Implement compliance checking
- [ ] Add progress reporting
- [ ] Implement violation detection
- [ ] Add decision logging

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| TDD Compliance | 100% | Tests committed before implementation |
| Phase Progression | Enforced | No skipped phases without documentation |
| Decision Logging | Complete | All decisions recorded in contextd |
| Progress Reporting | Per-phase | Clear feedback at each phase |
| Violation Detection | Real-time | Immediate blocking on violations |

---

## Conclusion

**Recommended Approach**: Build a custom orchestrator using the official Anthropic Go SDK with tight contextd integration via mark3labs/mcp-go.

**Key Benefits**:
1. Full control over agent behavior and phase gates
2. Official SDK stability and support
3. Native contextd integration for audit trails
4. Custom guardrails for TDD enforcement
5. Leverages existing contextd patterns (registry, hooks)

**Next Steps**:
1. Create `cmd/orchestrator` directory structure
2. Implement core agent loop with official SDK
3. Add phase gate state machine
4. Integrate contextd MCP client
5. Implement TDD and compliance guardrails

---

## Sources

- [Anthropic Go SDK](https://github.com/anthropics/anthropic-sdk-go)
- [Building agents with the Claude Agent SDK](https://www.anthropic.com/engineering/building-agents-with-the-claude-agent-sdk)
- [Ingenimax/agent-sdk-go](https://github.com/Ingenimax/agent-sdk-go)
- [schlunsen/claude-agent-sdk-go](https://github.com/schlunsen/claude-agent-sdk-go)
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- [MCP-Go Documentation](https://mcp-go.dev/getting-started/)
