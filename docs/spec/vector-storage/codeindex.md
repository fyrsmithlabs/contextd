# Codebase Indexing

## AST-Based Extraction

Uses tree-sitter to extract semantic units (not arbitrary chunks).

**Supported Unit Types**:

| Type | Description | Example |
|------|-------------|---------|
| function | Complete function | `func Search(ctx, query) {...}` |
| method | Method on type | `func (s *Service) Search(...)` |
| type | Struct/interface | `type Memory struct {...}` |
| const | Const/var block | `const ( A = 1; B = 2 )` |

**Why semantic units (not chunking)**:
- No data loss - each unit is complete
- No mixing - boundaries are semantic
- Natural granularity - developers think in functions

## Textify Conversion

Converts code to natural language for NLP embedding (per Qdrant tutorial):

```go
func textify(unit *SemanticUnit) string {
    // 1. Humanize names (camelCase → words)
    name := inflection.Humanize(inflection.Underscore(unit.Name))
    
    // 2. Include docstring
    docstring := ""
    if unit.Docstring != "" {
        docstring = fmt.Sprintf("that does %s ", unit.Docstring)
    }
    
    // 3. Add context
    context := fmt.Sprintf("module %s file %s", unit.Context.Module, unit.Context.FileName)
    
    // 4. Combine
    return fmt.Sprintf("%s %s %sdefined as %s %s",
        unit.UnitType, name, docstring, unit.Signature, context)
}
```

**Example Output**:
```
Function Search that does find memories by query defined as 
Fn search ctx context query string limit int defined in 
struct Memory manager module reasoning file manager go
```

## Git Integration (go-git)

### Delta Detection

```go
func GetChangedFiles(worktree, lastSHA string) []string {
    // Committed changes since last index
    committed := git.Diff(lastSHA, "HEAD", "--name-only")
    
    // Uncommitted changes (staged + unstaged)
    uncommitted := git.Status("--porcelain")
    
    return unique(append(committed, uncommitted...))
}
```

### Ref Watcher

```go
func (s *Session) watchGitEvents() {
    watcher := git.NewRefWatcher(s.worktree)  // Watches .git/HEAD, .git/refs/
    
    for {
        select {
        case event := <-watcher.Events:
            // Commit, checkout, merge, rebase, pull
            s.triggerIndexCheck(event)
        case <-time.After(10 * time.Minute):
            // Fallback: catch unstaged changes
            if s.isActive() {
                s.triggerIndexCheck(EventPoll)
            }
        }
    }
}
```

### Index Freshness Check

```go
type IndexMetadata struct {
    WorktreePath    string
    Branch          string
    LastIndexedSHA  string
    LastIndexedAt   time.Time
}

func NeedsReindex(meta IndexMetadata) bool {
    currentSHA := git.HEAD(meta.WorktreePath)
    return currentSHA != meta.LastIndexedSHA
}
```

## Large Function Handling

Functions >512 tokens use BM25 (no token limit, no truncation):

```go
func (i *Indexer) embedUnit(unit *SemanticUnit) *DocumentPoint {
    model := i.config.DefaultModel  // e.g., "sentence-transformers/all-minilm-l6-v2"
    
    if unit.TokenCount > i.config.LargeFunctionThreshold {
        model = "qdrant/bm25"  // Fallback: no token limit
    }
    
    return &DocumentPoint{
        ID:       unit.ID,
        Document: &Document{Text: unit.Content, Model: model},
        Payload:  unit.ToPayload(),
    }
}
```

## Supported Languages

| Language | tree-sitter Grammar | Status |
|----------|---------------------|--------|
| Go | `tree-sitter-go` | Phase 2 |
| TypeScript | `tree-sitter-typescript` | Phase 5 |
| Python | `tree-sitter-python` | Phase 5 |
| Rust | `tree-sitter-rust` | Phase 5 |
