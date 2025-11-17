---
name: github-actions-workflows
description: Use when creating or modifying GitHub Actions workflows - provides security patterns, common gotchas, performance optimizations, and debugging techniques for workflow development
---

# GitHub Actions Workflows

## Overview

**GitHub Actions workflows are infrastructure code.** They execute third-party code with credential access, making security and correctness critical.

**Core principle:** Explicit is better than implicit. Define permissions, pin versions, validate inputs.

## When to Use

Use this skill when:
- Creating new workflow files in `.github/workflows/`
- Modifying existing workflows
- Debugging workflow failures
- Optimizing workflow performance
- Reviewing workflow security

## Quick Reference

| Task | Pattern |
|------|---------|
| **Minimal workflow** | `on` + `jobs` + `steps` + `runs-on` |
| **Secure secrets** | Use `${{ secrets.NAME }}` never hardcode |
| **Script injection** | Use env vars: `env: VAR: ${{ github.event.* }}` |
| **Pin actions** | Use commit SHA: `uses: actions/checkout@a1b2c3d4` |
| **Set permissions** | Explicit: `permissions: contents: read` |
| **Path filtering** | Combine with `branches` for precision |
| **Caching** | Use `actions/cache` for dependencies |
| **Matrix builds** | `strategy: matrix:` for parallel tests |
| **Conditionals** | `if: ${{ }}` with expressions |
| **Debug failures** | Enable debug logging, check annotations |

## Workflow Structure

### Minimal Complete Workflow

```yaml
name: Test

on:
  pull_request:
    branches: [main]
    paths:
      - '**.go'
      - 'go.mod'

permissions:
  contents: read

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true

      - name: Run tests
        run: go test -v ./...
```

### Key Components

**Triggers (`on`)**
- `push`: On commits to branches
- `pull_request`: On PR open/update
- `workflow_dispatch`: Manual trigger
- `schedule`: Cron-based runs
- `issues`: On issue events (labeled, opened, etc.)

**Jobs (`jobs`)**
- Run in parallel by default
- Use `needs` for dependencies
- Each gets fresh runner environment

**Steps**
- Sequential within a job
- Use `uses` for actions, `run` for commands
- Share workspace but not environment

## Security Patterns

### 1. Prevent Script Injection

**Never do this:**
```yaml
# ❌ DANGEROUS: Attacker controls PR title
run: echo "PR: ${{ github.event.pull_request.title }}"
```

**Always do this:**
```yaml
# ✅ SAFE: Use environment variable
env:
  PR_TITLE: ${{ github.event.pull_request.title }}
run: echo "PR: $PR_TITLE"
```

### 2. Minimal Permissions

**Principle:** Explicit least-privilege permissions

```yaml
permissions:
  contents: read        # Read repo
  pull-requests: write  # Comment on PRs
  issues: write         # Update issues
```

**Common permission scopes:**
- `contents`: Repository files
- `pull-requests`: PR operations
- `issues`: Issue operations
- `checks`: Check run operations
- `id-token`: OIDC token (for cloud auth)

### 3. Pin Action Versions

**Don't use tags:**
```yaml
# ❌ Mutable: Tag can be moved
uses: actions/checkout@v4
```

**Use commit SHAs:**
```yaml
# ✅ Immutable: Specific commit
uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11  # v4.1.1
```

**Why:** Tags can be force-pushed by compromised accounts. Commit SHAs are immutable.

### 4. Secrets Management

```yaml
- name: Login to Docker Hub
  uses: docker/login-action@v3
  with:
    username: ${{ secrets.DOCKER_HUB_USERNAME }}
    password: ${{ secrets.DOCKER_HUB_TOKEN }}
```

**Rules:**
- Never hardcode credentials
- Use GitHub Secrets UI for storage
- Prefer OIDC over long-lived secrets
- Secrets automatically masked in logs
- Don't print JSON/XML containing secrets (bypasses masking)

## Common Gotchas

### 1. Path Filtering Limit

**Problem:** Only first 300 changed files checked

```yaml
on:
  push:
    paths:
      - 'src/**'  # May miss changes if >300 files modified
```

**Solution:** Use specific patterns or accept all changes

### 2. Matrix + Conditionals Order

**Problem:** Job-level `if` evaluates before matrix expansion

```yaml
jobs:
  test:
    if: ${{ github.event_name == 'push' }}  # Evaluates once
    strategy:
      matrix:
        os: [ubuntu, macos, windows]
```

**Solution:** Use step-level conditionals for matrix-aware logic

### 3. Concurrency Cancellation

**Problem:** Order not guaranteed when canceling

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true  # Arbitrary cancellation order
```

**Solution:** Design jobs to be cancellation-safe (idempotent)

### 4. Expression Syntax

**Problem:** YAML reserves `!` symbol

```yaml
# ❌ Syntax error
if: !startsWith(github.ref, 'refs/tags/')

# ✅ Correct
if: ${{ !startsWith(github.ref, 'refs/tags/') }}
```

### 5. Multiline Run Commands

**Problem:** Unescaped special characters

```yaml
# ❌ Shell escaping issues
run: |
  PROMPT=$(cat .claude/prompts/test.md)
  PROMPT="${PROMPT//\{\{ var \}\}/${{ github.repository }}}"
```

**Solution:** Use heredoc or escape properly

```yaml
# ✅ Heredoc syntax
run: |
  PROMPT=$(cat <<'EOF'
  Content with ${{ variables }} escaped
  EOF
  )
```

## Performance Optimization

### 1. Caching Dependencies

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '1.21'
    cache: true  # Automatic go.mod caching

- uses: actions/cache@v4
  with:
    path: ~/.cache/custom
    key: ${{ runner.os }}-custom-${{ hashFiles('**/*.lock') }}
```

### 2. Path Filters

```yaml
on:
  pull_request:
    branches: [main]
    paths:
      - '**.go'
      - 'go.mod'
      - 'go.sum'
  paths-ignore:
    - 'docs/**'
    - '**.md'
```

### 3. Concurrency Groups

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true  # Cancel outdated runs
```

### 4. Job Parallelization

```yaml
jobs:
  test:
    # Runs immediately

  lint:
    # Runs in parallel with test

  deploy:
    needs: [test, lint]  # Waits for both
```

### 5. Matrix Strategy

```yaml
strategy:
  matrix:
    go-version: ['1.20', '1.21', '1.22']
    os: [ubuntu-latest, macos-latest]
  fail-fast: false  # Continue all combinations even if one fails
```

## Context Variables

**Most useful contexts:**

| Context | Example | Usage |
|---------|---------|-------|
| `github.event_name` | `push`, `pull_request` | Determine trigger type |
| `github.ref` | `refs/heads/main` | Branch/tag reference |
| `github.sha` | `a1b2c3d4...` | Commit SHA |
| `github.actor` | `username` | Triggering user |
| `github.repository` | `owner/repo` | Repository name |
| `github.event.*` | (varies) | Event-specific data |
| `runner.os` | `Linux`, `macOS` | Operating system |
| `secrets.*` | (redacted) | Repository secrets |
| `inputs.*` | (varies) | Workflow dispatch inputs |

## Debugging Workflows

### Enable Debug Logging

**Repository secrets:**
- `ACTIONS_STEP_DEBUG`: `true` (step-level)
- `ACTIONS_RUNNER_DEBUG`: `true` (runner-level)

### Common Failure Patterns

| Symptom | Likely Cause | Solution |
|---------|--------------|----------|
| "Resource not accessible" | Missing permissions | Add to `permissions:` |
| "Unexpected value" | Wrong expression syntax | Use `${{ }}` wrapper |
| "Command not found" | Tool not installed | Add setup action |
| Flaky failures | Race conditions | Use proper wait conditions |
| "No files found" | Wrong path or glob | Check working directory |
| Script injection error | Untrusted input | Use env vars |

### Check Workflow Annotations

GitHub highlights specific line numbers in workflow files when errors occur. Check:
- Workflow file annotations (syntax errors)
- Job annotations (permission errors)
- Step annotations (command failures)

## Common Mistakes

### ❌ Hardcoding Secrets
```yaml
env:
  API_KEY: abc123xyz  # Never do this
```

### ❌ Using Untrusted Input Directly
```yaml
run: echo "${{ github.event.issue.title }}"  # Script injection risk
```

### ❌ Over-Privileged Token
```yaml
permissions: write-all  # Too broad
```

### ❌ Mutable Action Versions
```yaml
uses: actions/checkout@main  # Can change unexpectedly
```

### ❌ No Path Filtering
```yaml
on: [push]  # Runs on every commit including docs
```

## Best Practices Checklist

When creating/modifying workflows:

- [ ] Explicit minimal `permissions:` defined
- [ ] Actions pinned to commit SHA (or use Dependabot)
- [ ] Untrusted inputs use env vars (no direct `${{}}`)
- [ ] Secrets referenced via `${{ secrets.* }}`, never hardcoded
- [ ] Path filters for efficiency (`paths:`, `paths-ignore:`)
- [ ] Caching enabled for dependencies
- [ ] Descriptive job and step names
- [ ] Conditionals use `${{ }}` syntax
- [ ] Shell commands properly escaped
- [ ] Matrix strategy for parallel testing (if applicable)
- [ ] Concurrency groups for cancellation (if applicable)
- [ ] Clear failure debugging information

## Real-World Impact

**Security:** Script injection vulnerabilities affect ~20% of workflows in public repos (2024 research)

**Performance:** Proper caching reduces CI time by 40-60% on average

**Reliability:** Path filtering prevents unnecessary workflow runs, reducing queue times by 30-50%

## Resources

- [Official Workflow Syntax](https://docs.github.com/en/actions/writing-workflows/workflow-syntax-for-github-actions)
- [Security Best Practices](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions)
- [Action Marketplace](https://github.com/marketplace?type=actions)
