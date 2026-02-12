# Contextd Memory Gap Incident Report

**Date:** 2026-01-30
**Session:** `418e6421-03ce-4c63-b6b1-240beb2ee08f`
**Severity:** Operational - Wasted effort, user frustration

## Summary

Claude had zero operational memory of how to build, run, or configure the DevPilot application despite the entire codebase being built with Claude + contextd over many sessions. When asked to run the app and test it with Chrome, Claude had to rediscover everything from scratch by reading source files.

## What Happened

### User Request (19:24 UTC)

> "Run the app and use claude-chrome (working now) to do a full test of the site as a user"

### Claude's Response: Complete Amnesia

Claude had no memory of how to run the app. It began reading files from scratch:

1. **Read `Makefile`** - to discover `make run`, `make build`, `make dev`
2. **Searched for `devpilot.yaml*`** - to find config files
3. **Read `devpilot.yaml.example`** - to understand configuration format
4. **Read `cmd/devpilot/main.go`** - to understand entry point and demo mode flag
5. **Read `internal/config/config.go`** - to understand config loading, env vars
6. **Read `internal/server/server.go`** - to understand server setup, demo mode routing

This took ~8 minutes of file reading before Claude could construct the build command:
```bash
DEMO_MODE=true make run
```

### User Frustration (19:32 UTC)

The user interrupted and asked:
> "How do you not remember this?"

Claude acknowledged the gap and searched contextd:

### Memory Search Results

**Query:** `"running the app demo mode startup build"` (limit: 5)

**Results returned - none relevant to operations:**

| # | Memory Title | Relevance to "run the app" |
|---|---|---|
| 1 | "GitHub planning: kagent.dev Integration" | None - project planning |
| 2 | "Code Quality Review: MCP Handlers and GitHub Cache" | None - code review |
| 3 | "A2A Security Fixes Applied" | None - security patches |
| 4 | "RALPH: Phase 2 Group 1 - MCP Config & GitHub API" | None - feature implementation |
| 5 | "Comprehensive Research: Best Practices 2026" | None - research output |

All 5 memories were architectural/feature-level. Zero operational memories existed.

### Checkpoint Search Results

**Query:** `checkpoint_list` (limit: 10)

**Result:** 1 checkpoint total

| # | Checkpoint | Relevance |
|---|---|---|
| 1 | "phase2-start" - Phase 2 orchestration start | None - workflow orchestration |

No checkpoints captured operational context like "app is running on port 8080" or "use DEMO_MODE=true for local dev."

### Resolution

Claude eventually constructed the run command from source code analysis:
```bash
DEMO_MODE=true make run
```

The app started successfully:
```
Building devpilot...
Running devpilot...
K8s client initialized for kagent agent provisioning
Rate limiting enabled: 10 requests/minute, burst 20
Temporal workflows disabled, using in-memory orchestrator only
DevPilot started on http://localhost:8080
```

## Root Cause Analysis

### 1. No Operational Memories Were Ever Recorded

Across all sessions that built this codebase, no one (neither Claude nor contextd automation) recorded memories about:
- How to build the app (`make build`, `make run`)
- How to run in demo mode (`DEMO_MODE=true`)
- What port the app runs on (`localhost:8080`)
- What demo mode does (bypasses OAuth, enables demo login)
- Config file location and format (`devpilot.yaml`)
- Required environment variables

### 2. Memory Bias Toward Architecture Over Operations

All 5 returned memories were about:
- Feature planning (kagent integration)
- Code review findings
- Security fixes
- Implementation details (RALPH framework)
- Research output

This reveals a systematic bias: memories are recorded during **implementation sessions** (design decisions, code review, security fixes) but never during **operational sessions** (building, running, debugging, configuring).

### 3. Checkpoints Don't Capture Running State

The single checkpoint was about orchestration workflow state. No checkpoint captured:
- "App is currently running on port 8080"
- "Using DEMO_MODE=true for local development"
- "Config file is at devpilot.yaml with demo_mode: true"

### 4. CLAUDE.md Had No Quick Start Section

At the time of the incident, CLAUDE.md contained project structure, technology stack, and security guidelines but no "Quick Start" or "How to Run" section. This was subsequently fixed by adding operational commands to CLAUDE.md.

## Impact

- ~8 minutes of wasted file-reading to rediscover known information
- User frustration ("How do you not remember this?")
- Undermines trust in contextd as a memory system
- Demonstrates that contextd memories skew architectural/feature and miss operational knowledge

## Remediation Suggestions for contextd

### Short-term (CLAUDE.md - already applied)

The Commands section in CLAUDE.md was updated with build/run/test commands. This ensures every session has baseline operational knowledge.

### Medium-term (contextd features)

1. **Auto-record operational memories on build/run success**
   - When a `make run`, `make build`, or similar command succeeds, auto-record a memory: "App built/ran successfully using: [command]"
   - Include environment variables that were set

2. **Session-end operational summary**
   - At session end, record "how I started/ran the app" as a memory
   - Include: command used, port, env vars, any config files touched

3. **Memory categories should include "operational"**
   - Current memories are implicitly: architecture, feature, security, review
   - Add explicit "operational" category for: build commands, run commands, config setup, debugging steps
   - Memory search should be able to filter by category

4. **Checkpoint should include process/runtime context**
   - When saving a checkpoint, capture any running background processes started during the session
   - Include environment variables that were set for those processes

### Long-term (contextd intelligence)

5. **Detect operational knowledge gaps**
   - If a memory search for "how to run" returns 0 relevant results, flag this as a gap
   - Prompt the agent to record operational knowledge after successfully running the app

6. **Cross-session operational knowledge propagation**
   - When session A runs the app successfully and session B asks "how to run the app," contextd should bridge this automatically
   - This is the core promise of a memory system that currently isn't being fulfilled

## Key Takeaway

Contextd's memory system is optimized for capturing **what was built** but completely blind to **how to operate what was built**. For a tool designed to maintain context across sessions, this is a critical gap - the most common cross-session question is "how do I run this?" not "what architecture decisions were made?"
