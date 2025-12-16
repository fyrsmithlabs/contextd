# Context-Folding Research Update (December 2025)

**Updated**: 2025-12-13
**Related Issue**: [#17 - Context Folding Implementation](https://github.com/fyrsmithlabs/contextd/issues/17)

---

## Executive Summary

Context folding in 2025 has evolved from theoretical concept to production-ready patterns. Key insight: **Claude Agent SDK already implements subagent spawning with isolated context** - contextd should extend this pattern with memory injection and budget enforcement.

## Key Research Papers

### 1. Context-Folding (arXiv:2510.11967)

**Core Innovation**: Procedural branch/fold pattern with reinforcement learning.

- Agents branch into sub-trajectories for subtasks
- "Fold" collapses intermediate steps while retaining concise summary
- **FoldGRPO**: End-to-end RL framework with process rewards for task decomposition
- **Results**: 10x smaller active context while matching ReAct baselines

**Relevance to contextd**: Validates our branch/return architecture. The "fold" operation = our `return(summary)` mechanism.

### 2. AgentFold (arXiv:2510.24699)

**Core Innovation**: Multi-scale context management inspired by human retrospective consolidation.

- Treats context as "dynamic cognitive workspace to be actively sculpted"
- Two folding operations:
  - **Granular condensation**: Preserves fine-grained critical details
  - **Deep consolidation**: Abstracts entire multi-step subtasks
- **Results**: 36.2% on BrowseComp, outperforms DeepSeek-V3.1-671B and OpenAI o4-mini

**Relevance to contextd**: Multi-scale folding aligns with our nested branch architecture. Consider adding granular vs. deep return modes.

### 3. ACON (arXiv:2510.00615)

**Core Innovation**: Gradient-free compression optimization.

- Separate compression for **history** and **observations** (tool outputs)
- Learn compression guidelines from success/failure contrastive analysis
- Compress when threshold exceeded, not every step

**Relevance to contextd**: Aligns with our existing compression package. ACON's selective compression validates our approach of compressing only when needed.

---

## Production Implementations (2025)

### Claude Agent SDK (September 2025)

**Key Features**:
- Subagent spawning with isolated context windows
- Automatic context compression and management
- Session resumption and forking
- Sandbox execution modes (Docker isolation, subprocess)

**Subagent Pattern**:
```python
# Claude Agent SDK - subagent approach
agent = Agent(
    tools=[my_tool],
    executor=ExecutorType.SUBPROCESS  # Isolated execution
)
# Subagent has own context, returns only relevant info to orchestrator
```

**Critical Insight**: Claude Agent SDK subagents **already solve context isolation**. They:
1. Spawn with independent context windows
2. Execute in parallel (when tasks permit)
3. Return only relevant information to parent

**Gap**: No memory injection from persistent store. No budget enforcement. No MCP tool interface.

### LangGraph DeepAgents

- Planning tool + filesystem backend + subagent spawning
- "General purpose version of Claude Code"
- Explicit workflow design with shared memory and parallelism

### Google ADK

**Design Principle**: "Scope by default - every model call and sub-agent sees minimum context required"

Architecture includes:
- Tiered storage
- Compiled views
- Pipeline processing
- Strict scoping

---

## Implementation Strategy for contextd

### Option A: Leverage Claude Agent SDK Subagents

**Approach**: Wrap Claude Agent SDK's subagent mechanism with contextd enhancements.

**Pros**:
- Battle-tested subprocess isolation
- Automatic context compression built-in
- Production-ready

**Cons**:
- Tight coupling to Claude ecosystem
- Limited customization of context passing

### Option B: MCP-Native Implementation (Recommended)

**Approach**: Implement branch/return as MCP tools that spawn isolated processes.

**Architecture**:
```
Parent Agent (main context)
    │
    ├── branch() MCP call to contextd
    │       │
    │       ├── contextd spawns isolated subprocess/goroutine
    │       ├── contextd injects ReasoningBank memories
    │       ├── contextd enforces budget/timeout
    │       └── Child process runs with isolated context
    │
    └── return() from child → parent receives summary only
```

**Implementation Details**:

1. **Branch Creation**:
   - contextd receives `branch(description, prompt, budget)` MCP call
   - Creates isolated execution context (goroutine with cancellation)
   - Queries ReasoningBank for relevant memories
   - Returns branch_id + injected_context to caller

2. **Context Isolation**:
   - Branch has NO access to parent context
   - Only receives: description, prompt, injected memories
   - Uses same MCP tools but with independent token tracking

3. **Return Mechanism**:
   - `return(message)` completes branch
   - Only summary returned to parent
   - Optionally queue for memory extraction

4. **Budget Enforcement**:
   - Track token usage per branch
   - Force return when budget exhausted
   - Timeout enforcement via context cancellation

### Option C: Hybrid with Claude Code Task Tool

**Approach**: Model after Claude Code's existing Task tool which spawns subagents.

**Key Insight**: Claude Code ALREADY has this pattern:
- Task tool spawns specialized agents (Explore, Plan, etc.)
- Each agent has isolated context
- Returns summarized results

contextd's `branch/return` could be a generalization of this pattern, exposed as MCP tools.

---

## Recommended Approach: Option B (MCP-Native)

**Rationale**:
1. **MCP Protocol Alignment**: contextd is an MCP server; keep features within MCP
2. **Platform Independence**: Works with any MCP client, not just Claude
3. **Memory Integration**: Direct access to ReasoningBank for injection
4. **Custom Budget Logic**: Full control over token tracking and enforcement

**Key Differences from Claude Agent SDK**:
- contextd manages the isolation, not the LLM client
- Memory injection from ReasoningBank (Claude SDK doesn't have this)
- Explicit budget enforcement with configurable limits
- MCP tool interface for any agent framework

---

## Success Metrics (from Research)

| Metric | Research Benchmark | contextd Target |
|--------|-------------------|-----------------|
| Context compression | 10x (Context-Folding) | >5x (80% reduction) |
| Latency overhead | Not specified | <100ms branch/return |
| Memory extraction | N/A | >50% of successful branches |
| Budget compliance | Implicit in RL | 100% explicit enforcement |

---

## Open Questions for Consensus Review

1. **Subprocess vs. Goroutine Isolation**: How do we actually isolate context?
   - True subprocess: Expensive but complete isolation
   - Goroutine + context tracking: Cheap but relies on discipline

2. **Token Tracking**: How do we count tokens in branches?
   - Proxy through contextd (intercept all LLM calls)?
   - Trust agent to report via `consume_tokens` tool?
   - External token counter service?

3. **Memory Injection Budget**: What percentage of branch budget for memories?
   - Spec says 20% - is this validated?

4. **Nested Branch Depth**: Default 3 - sufficient for most use cases?

5. **Failed Branch Handling**: How to extract anti-patterns?
   - Auto-extract on failure?
   - Require explicit flag?

---

## Sources

- [Context-Folding Paper (arXiv:2510.11967)](https://arxiv.org/abs/2510.11967)
- [AgentFold Paper (arXiv:2510.24699)](https://arxiv.org/abs/2510.24699)
- [ACON Paper (arXiv:2510.00615)](https://arxiv.org/abs/2510.00615)
- [Building Agents with Claude Agent SDK](https://www.anthropic.com/engineering/building-agents-with-the-claude-agent-sdk)
- [Claude Code Subagents - InfoQ](https://www.infoq.com/news/2025/08/claude-code-subagents/)
- [Google ADK Multi-Agent Framework](https://developers.googleblog.com/architecting-efficient-context-aware-multi-agent-framework-for-production/)
- [LangGraph DeepAgents](https://github.com/langchain-ai/deepagents)
- [Claude Agent SDK Best Practices](https://skywork.ai/blog/claude-agent-sdk-best-practices-ai-agents-2025/)
