# CLAUDE.md Structure Guide

## Overview

The contextd project uses a modular CLAUDE.md hierarchy to provide context-specific guidance to Claude Code. This structure prevents ambiguity, reduces context bloat, and ensures developers find relevant information quickly.

## Hierarchy

```
contextd/
├── CLAUDE.md                      # Root: Project overview, quick start, architecture
├── cmd/
│   ├── contextd/
│   │   └── CLAUDE.md              # Server: Routes, middleware, MCP, lifecycle
│   └── ctxd/
│       └── CLAUDE.md              # Client: CLI commands, installation, setup
├── pkg/
│   └── CLAUDE.md                  # Packages: Design patterns, testing, interfaces
└── internal/
    └── CLAUDE.md                  # Internal: Handlers, middleware, API design
```

## File Purposes

### Root CLAUDE.md

**Purpose**: High-level project overview and navigation hub

**Contents**:
- Project philosophy and goals
- Quick start guide
- Architecture overview
- Implementation status
- Navigation to nested CLAUDE.md files
- Common troubleshooting
- Reference documentation links

**Target Length**: ~400 lines (reduced from 586)

**Audience**: New developers, Claude Code (high-level context)

### cmd/contextd/CLAUDE.md

**Purpose**: Server implementation details

**Contents**:
- Server startup and lifecycle
- Middleware stack (order is CRITICAL)
- Route structure and registration
- API mode implementation
- MCP mode implementation
- Service initialization order
- Unix socket security
- Authentication flow
- Error handling patterns
- Configuration management
- Observability setup

**Target Length**: ~400 lines

**Audience**: Developers working on server code, adding endpoints/tools

### cmd/ctxd/CLAUDE.md

**Purpose**: Client CLI implementation

**Contents**:
- Command structure and patterns
- Platform-specific code (Linux/macOS)
- Installation workflow
- Claude Code setup workflow
- Health checking
- Debug tools
- TUI implementation
- Repository indexing
- Backup management
- User experience guidelines
- Error message formatting

**Target Length**: ~450 lines

**Audience**: Developers working on CLI, adding commands

### pkg/CLAUDE.md

**Purpose**: Package design guidelines

**Contents**:
- Package philosophy (public vs internal)
- Standard package layout
- Public API design patterns
- Package dependency rules
- Service pattern
- Interface-based design
- Testing guidelines
- Key package overviews (auth, config, checkpoint, etc.)
- Adding new packages
- Performance considerations

**Target Length**: ~500 lines

**Audience**: Developers creating new packages, refactoring code

### internal/CLAUDE.md

**Purpose**: Internal implementation patterns

**Contents**:
- Internal vs pkg distinction
- Handler pattern
- Request/response models
- Middleware patterns
- Error handling
- Pagination and search patterns
- Testing strategies
- Security considerations
- Performance optimizations
- Observability

**Target Length**: ~400 lines

**Audience**: Developers working on HTTP handlers, middleware

## Design Principles

### 1. Single Responsibility

Each CLAUDE.md file focuses on its directory's specific concerns. No duplication across files.

**Example**:
- Root: "Server is dual-mode (API + MCP). See cmd/contextd/CLAUDE.md"
- cmd/contextd/CLAUDE.md: Full details on both modes

### 2. Clear Navigation

Each nested file references the root and related files:

```markdown
See [../../CLAUDE.md](../../CLAUDE.md) for project overview.
See [../pkg/CLAUDE.md](../pkg/CLAUDE.md) for package guidelines.
```

### 3. Avoid Ambiguity

Use declarative, unambiguous language:

**Good**:
```markdown
Middleware order MUST NOT be changed:
1. Logger - Must be first
2. Recover - Must be early
```

**Bad**:
```markdown
You should probably keep middleware in this order:
1. Logger - logs things
2. Recover - recovers panics
```

### 4. Context-Appropriate Detail

Root provides overview, nested files provide implementation details:

**Root**:
```markdown
Authentication via pkg/auth with Bearer tokens.
```

**pkg/CLAUDE.md**:
```markdown
### pkg/auth
- Token generation: crypto/rand (32 bytes → hex)
- Permissions: MUST be 0600
- Comparison: constant-time to prevent timing attacks
- Storage: ~/.config/contextd/token
```

### 5. Code Examples

Include concrete, tested examples:

```go
// Handler pattern
func (h *Handler) HandleCreate(c echo.Context) error {
    var req CreateRequest
    if err := validation.ValidateRequest(c, &req); err != nil {
        return err
    }
    // ...
}
```

## When to Add Information

### Add to Root if:
- It's essential for understanding the project
- New developers need it immediately
- It's a project-wide decision or philosophy
- It's frequently referenced

### Add to Nested File if:
- It's specific to that directory's code
- It's implementation detail
- It's only relevant when working in that area
- It's a "how-to" for adding/modifying code

## Maintenance Guidelines

### Keep Files Updated

When making changes:
1. Update the relevant CLAUDE.md file(s)
2. Ensure cross-references remain accurate
3. Remove outdated information
4. Update code examples if APIs change

### Prevent Duplication

Before adding information:
1. Check if it exists elsewhere
2. If it exists, reference it instead of duplicating
3. If it needs to be in multiple places, keep one authoritative source and reference it

### Regular Audits

Quarterly review checklist:
- [ ] Remove outdated information
- [ ] Fix broken cross-references
- [ ] Update code examples
- [ ] Verify file lengths stay reasonable (<500 lines)
- [ ] Check for duplication
- [ ] Ensure navigability

## Integration with Claude Code

### How Claude Code Uses These Files

1. **Initial Context**: Loads root CLAUDE.md for project overview
2. **Directory Context**: When working in a directory, loads that directory's CLAUDE.md
3. **Cross-Reference**: Follows links to related CLAUDE.md files as needed
4. **Decision Making**: Uses CLAUDE.md to understand patterns and make consistent changes

### Optimizing for Token Efficiency

**Good**:
- Clear section headers (Claude can navigate)
- Concise, declarative statements
- Code examples over prose
- Cross-references instead of duplication

**Bad**:
- Long explanatory paragraphs
- Repeated information
- Vague language requiring inference
- Missing structure

## File Size Guidelines

Target line counts (with flexibility):

| File | Target | Maximum | Current |
|------|--------|---------|---------|
| Root | 400 | 500 | 415 |
| cmd/contextd/ | 400 | 500 | 453 |
| cmd/ctxd/ | 450 | 550 | 496 |
| pkg/ | 500 | 600 | 567 |
| internal/ | 400 | 500 | 476 |

**Total**: ~2,400 lines (vs ~586 single file)

## Benefits of This Structure

### For Developers

1. **Context-Specific**: Find relevant information immediately
2. **Reduced Noise**: No unrelated information cluttering view
3. **Clear Patterns**: Each directory has clear examples
4. **Easy Navigation**: Cross-references guide you
5. **Maintainable**: Update one file, not scattered information

### For Claude Code

1. **Token Efficiency**: Load only relevant context
2. **Reduced Ambiguity**: Clear, declarative guidance
3. **Pattern Recognition**: Consistent structure across files
4. **Scalability**: Add directories without bloating root

### For Project

1. **Documentation as Code**: Lives with the code it documents
2. **Versioned**: Changes tracked with code changes
3. **Discoverable**: Standard locations (CLAUDE.md in each directory)
4. **Enforceable**: Can lint, check length, verify links

## Example Usage Scenarios

### Scenario 1: Adding a New Endpoint

Developer (or Claude Code) workflow:

1. **Start**: Root CLAUDE.md
   - "Adding New Features" section
   - Points to three files

2. **Service Layer**: pkg/CLAUDE.md
   - "Adding New Packages" section
   - Service pattern example

3. **Handler**: internal/CLAUDE.md
   - "Adding New Handlers" section
   - Handler pattern example

4. **Route**: cmd/contextd/CLAUDE.md
   - "Adding New Endpoints" section
   - Route registration example

Total context loaded: ~1,400 lines of relevant information (vs 586 lines of mixed content)

### Scenario 2: Debugging Server Startup

Developer workflow:

1. **Root**: Quick commands, troubleshooting
2. **cmd/contextd/CLAUDE.md**: Server lifecycle, initialization order
3. **pkg/CLAUDE.md**: Config system, telemetry init

Focused context, no CLI or handler information needed.

### Scenario 3: Creating New CLI Command

Developer workflow:

1. **Root**: Quick commands overview
2. **cmd/ctxd/CLAUDE.md**: Command pattern, testing, platform considerations

No server or package internals needed.

## Migration Notes

### From Single File to Hierarchy

**Completed**:
- Extracted server details → cmd/contextd/CLAUDE.md
- Extracted client details → cmd/ctxd/CLAUDE.md
- Extracted package guidelines → pkg/CLAUDE.md
- Created internal guidelines → internal/CLAUDE.md
- Streamlined root → CLAUDE.md

**Root Reduction**:
- Before: 586 lines
- After: 415 lines
- Reduction: 29% (171 lines)

**Total Documentation**:
- Before: 586 lines (single file)
- After: 2,407 lines (5 files)
- Added: 1,821 lines of detailed, context-specific guidance

## Future Enhancements

### Potential Additions

1. **docs/CLAUDE.md** - Documentation writing guidelines
2. **scripts/CLAUDE.md** - Script usage and development
3. **tests/CLAUDE.md** - Testing strategy and patterns
4. **.github/CLAUDE.md** - CI/CD, workflows, GitHub automation

### Automation

Potential tooling:
- `make validate-docs` - Check CLAUDE.md files
  - Verify file sizes
  - Check cross-references
  - Lint for ambiguous language
  - Test code examples
- `make docs-stats` - Show documentation metrics
- `make docs-graph` - Generate navigation diagram

## Questions and Answers

**Q: Why not use a single comprehensive README?**
A: READMEs are for users. CLAUDE.md is for Claude Code and developers. Different audiences, different needs.

**Q: What if information is relevant in multiple places?**
A: Keep one authoritative source and cross-reference it.

**Q: How do I decide where to add new information?**
A: Ask: "When would someone need this?" If it's while working in a specific directory, add it there. If it's project-wide, add to root.

**Q: Can I add more nested levels (e.g., pkg/auth/CLAUDE.md)?**
A: Yes, if the package is complex enough. Follow the same principles.

**Q: What about duplicate information in global ~/.claude/CLAUDE.md?**
A: That file is for user-wide configuration. Project CLAUDE.md is for project-specific guidance. Minimal overlap is acceptable.

## Related Documentation

- **Root**: [../../CLAUDE.md](../../CLAUDE.md) - Start here
- **Server**: [../../cmd/contextd/CLAUDE.md](../../cmd/contextd/CLAUDE.md)
- **Client**: [../../cmd/ctxd/CLAUDE.md](../../cmd/ctxd/CLAUDE.md)
- **Packages**: [../../pkg/CLAUDE.md](../../pkg/CLAUDE.md)
- **Internal**: [../../internal/CLAUDE.md](../../internal/CLAUDE.md)
