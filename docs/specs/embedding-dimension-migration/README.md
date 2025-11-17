# Embedding Dimension Migration

**Status**: Ready for Implementation
**Issue**: Checkpoint search failing with dimension mismatch

## Quick Summary


**Root Cause**: Hardcoded dimensions in vector store adapters + missing `EMBEDDING_DIM` config.

**Solution**: Dimension detection + migration tooling + data preservation.

## Immediate Solutions

### Option 1: Quick Fix (No Data Preservation) ⚡

**Best for**: Development/testing environments where checkpoint data is not critical

```bash
./scripts/fix-dimension-mismatch.sh
```

This will:
- ✅ Auto-detect TEI dimension (384)
- ❌ DROP all existing checkpoints
- ✅ Recreate collections with correct dimension
- ✅ Restart contextd service

### Option 2: Data Preservation (Manual Re-import) 💾

**Best for**: Production environments with important checkpoint data

```bash
./scripts/fix-dimension-mismatch.sh --preserve-data
```

This will:
- ✅ Export all checkpoints to JSON
- ✅ Drop and recreate collections
- ⚠️ **Requires manual re-import** (migration tool not yet implemented)

### Option 3: Switch to OpenAI (Keep Existing Data) 🔄

**Best for**: Users who prefer OpenAI API over TEI

```bash
./scripts/fix-dimension-mismatch.sh --switch-openai
```

This will:
- ✅ Keep all existing checkpoint data (1536-dim compatible)
- ✅ Switch to OpenAI API provider
- ⚠️ Requires OpenAI API key (paid service)

## Documentation

### For Users

- **Quick Fix Guide**: [QUICK-FIX.md](./QUICK-FIX.md)
  - Step-by-step manual instructions
  - All three options explained
  - Verification steps

### For Developers

- **Full Specification**: [SPEC.md](./SPEC.md)
  - Complete architecture design
  - Migration algorithm
  - Implementation phases
  - Testing strategy

## Implementation Status

### ✅ Completed

- [x] Root cause analysis
- [x] Full specification (SPEC.md)
- [x] Quick fix script (fix-dimension-mismatch.sh)
- [x] User documentation (QUICK-FIX.md)

### 🚧 In Progress

- [ ] Dimension detection (Phase 1)
- [ ] Export/Import tooling (Phase 2)
- [ ] CLI migration tool (Phase 3)
- [ ] Integration tests (Phase 4)

### 📋 Planned

- [ ] Automatic dimension detection at startup
- [ ] Re-embedding with new provider
- [ ] Resumable migration for large datasets
- [ ] Migration progress reporting
- [ ] A/B testing for search quality validation

## Running the Fix

### Prerequisites

```bash
# 1. TEI container running
docker ps --filter "name=tei"


# 3. contextd service installed
systemctl --user status contextd
```

### Execute Fix

```bash
# Change to project directory
cd /home/dahendel/projects/contextd

# Run fix script (choose one option)
./scripts/fix-dimension-mismatch.sh              # Option 1: Quick fix
./scripts/fix-dimension-mismatch.sh --preserve-data  # Option 2: Preserve data
./scripts/fix-dimension-mismatch.sh --switch-openai  # Option 3: Switch to OpenAI

# Verify
journalctl --user -u contextd -f
```

### Verification

Test checkpoint search after fix:

```bash
# Via MCP (if Claude Code integration active)
checkpoint_search "test query"

# Via curl (direct API)
curl -s --unix-socket ~/.config/contextd/api.sock \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $(cat ~/.config/contextd/token)" \
  -d '{"method":"tools/call","params":{"name":"checkpoint_search","arguments":{"query":"test"}}}' | jq
```

## Next Steps

### For Users

1. **Choose immediate solution**: Option 1, 2, or 3 above
2. **Run fix script**: `./scripts/fix-dimension-mismatch.sh`
3. **Verify**: Test checkpoint search operations
4. **Report issues**: If problems persist, see Troubleshooting section

### For Developers

1. **Implement Phase 1**: Dimension detection (`pkg/embedding/detect.go`)
2. **Implement Phase 2**: Migration tooling (`pkg/migration/`)
3. **Implement Phase 3**: CLI tool (`cmd/migrate/`)
4. **Add tests**: Integration tests for TEI ↔ OpenAI migration
5. **Update docs**: Add migration guide to GETTING-STARTED.md

## Troubleshooting

### "dimension mismatch" error persists

```bash
# Check actual dimension configured
journalctl --user -u contextd | grep -i "embedding.*dim"

# Check environment variable
systemctl --user show contextd -p Environment | grep EMBEDDING_DIM

# Check TEI actual output
curl -s -X POST http://localhost:8080/embed \
  -H "Content-Type: application/json" \
  -d '{"inputs": "test"}' | jq '.[0] | length'
```

### Data export fails

```bash

# Check if collections exist
  connections.connect(); \
  print([c for c in utility.list_collections()])'
"
```

### Service fails to restart

```bash
# Check service logs
journalctl --user -u contextd -n 50

# Check socket permissions
ls -la ~/.config/contextd/

# Reset service state
systemctl --user reset-failed contextd
systemctl --user restart contextd
```

## References

- **SPEC.md**: Full specification and architecture
- **QUICK-FIX.md**: Step-by-step manual instructions
- **fix-dimension-mismatch.sh**: Automated fix script
- **Regression Test**: `pkg/embedding/embedding_regression_test.go:178-235`
- **Config**: `pkg/config/config.go:203`

## Contributing

See [SPEC.md Implementation Plan](./SPEC.md#implementation-plan) for development tasks.

To contribute:
1. Pick a phase from the Implementation Plan
2. Create feature branch: `git checkout -b feature/dimension-migration-phase-N`
3. Implement with tests (TDD)
4. Submit PR referencing this spec
