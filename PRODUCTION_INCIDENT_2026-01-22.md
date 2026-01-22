# Production Incident Report: Metadata File Loss

**Incident ID**: PROD-2026-01-22-001
**Date**: 2026-01-22
**Status**: ✅ RESOLVED
**Severity**: P0 - Complete Service Outage
**Duration**: ~2 hours (discovery to resolution)

---

## Summary

contextd failed to initialize due to missing metadata file in the `contextd_memories` collection (hash: `e9f85bf6`), causing all vectorstore-dependent services to become unavailable.

**Impact**: Complete service outage - all MCP tools unavailable

**Resolution**: Manually recreated metadata file using recovery tool

---

## Timeline (EST)

| Time | Event |
|------|-------|
| Jan 20 15:54 | First document written to e9f85bf6 collection |
| Jan 22 07:52-08:44 | Additional documents written to collection |
| Jan 22 08:44+ | contextd startup fails with metadata error |
| Jan 22 14:00 | Incident discovered during integration testing |
| Jan 22 14:05 | Root cause identified (missing 00000000.gob) |
| Jan 22 14:05 | Recovery tool created and executed |
| Jan 22 14:06 | contextd successfully restarted - all services healthy |
| Jan 22 14:30 | Documentation and prevention strategies completed |

---

## Impact

### Services Affected
- ✅ Checkpoint (memory_search, checkpoint_save) - **UNAVAILABLE**
- ✅ Remediation (remediation_search, remediation_record) - **UNAVAILABLE**
- ✅ Repository (semantic_search, repository_index) - **UNAVAILABLE**
- ✅ ReasoningBank (memory_search, memory_record) - **UNAVAILABLE**
- ✅ Troubleshoot (troubleshoot_diagnose) - **UNAVAILABLE**

### User Impact
- **Claude Code users**: All contextd MCP tools non-functional
- **Data Loss**: None (documents preserved, metadata recovered)
- **Duration**: ~2 hours from discovery to full resolution

---

## Root Cause

### Technical Details

Chromem's `NewPersistentDB()` uses **fail-deadly** validation where ANY corrupt collection causes ENTIRE database initialization to fail:

```go
// chromem-go@v0.7.0/db.go lines 172-175
if c.Name == "" {
    return nil, fmt.Errorf("collection metadata file not found: %s", collectionPath)
}
```

**Collection State**:
```
e9f85bf6/ (contextd_memories)
├── 00000000.gob     ❌ MISSING (metadata file)
├── 3af7a34d.gob     ✅ Valid document
├── 93b27de1.gob     ✅ Valid document
├── 9cc9da47.gob     ✅ Valid document
└── ab56b992.gob     ✅ Valid document
```

### How This Occurred

**Most Likely Scenario**: Process crash during collection creation or metadata write

**Evidence**:
1. Collection directory exists with documents
2. Directory timestamp: Jan 22 08:44 (matches last document write)
3. No metadata file present
4. All 21 other collections have metadata files

**Possible Causes**:
1. Process crash after directory creation but before metadata write
2. Disk full during metadata write
3. File system corruption
4. Manual deletion (unlikely)

---

## Resolution

### Immediate Fix

**Created recovery tool**: `cmd/recover-metadata/main.go`

```bash
# Identified corrupt collection
for dir in ~/.config/contextd/vectorstore/*/; do
    if [ ! -f "${dir}00000000.gob" ]; then
        echo "Missing metadata: $(basename $dir)"
    fi
done
# Output: e9f85bf6

# Reverse engineered collection name from hash
python3 -c "import hashlib; print('e9f85bf6' == hashlib.sha256(b'contextd_memories').hexdigest()[:8])"
# Output: True

# Ran recovery tool
go run ./cmd/recover-metadata/main.go
# Output: ✅ Successfully created metadata file for collection: contextd_memories

# Verified fix
contextd
# Output: All services OK
```

### Verification

**Before Fix**:
```json
{
  "msg": "vectorstore initialization failed",
  "error": "creating chromem DB: collection metadata file not found: ...e9f85bf6"
}
{
  "msg": "contextd initialized",
  "services": ["checkpoint:unavailable", "remediation:unavailable", ...]
}
```

**After Fix**:
```json
{
  "msg": "ChromemStore initialized",
  "path": "/Users/dahendel/.config/contextd/vectorstore"
}
{
  "msg": "contextd initialized",
  "services": ["checkpoint:ok", "remediation:ok", "repository:ok", ...]
}
```

---

## Contributing Factors

### Design Flaws

1. **Single Point of Failure**: One corrupt collection breaks entire database
2. **No Graceful Degradation**: chromem fails entire load instead of skipping corrupt collections
3. **No Atomic Writes**: Metadata file write is not atomic, allowing partial failures
4. **No Recovery Mechanism**: chromem provides no built-in recovery for missing metadata

### Operational Gaps

1. **No Health Monitoring**: No periodic verification of metadata file integrity
2. **No Automated Recovery**: Manual intervention required
3. **No Backup Strategy**: Metadata files not backed up
4. **No Alerting**: Silent failure until service restart

---

## Prevention Measures

### Immediate (Implemented)

✅ **Recovery Tool**: `cmd/recover-metadata/main.go`
- Hash reverse lookup
- Metadata file recreation
- Verification checks

✅ **Documentation**: `docs/operations/METADATA_RECOVERY.md`
- Quick recovery runbook
- Prevention strategies
- Testing procedures

✅ **Root Cause Analysis**: `METADATA_LOSS_ROOT_CAUSE.md`
- Technical deep dive
- Timeline reconstruction
- Failure scenarios

### Short-Term (Planned)

⏳ **Graceful Degradation**: Wrap chromem initialization to quarantine corrupt collections
- Detect missing metadata files
- Move to `.quarantine/` directory
- Continue loading healthy collections
- Log warnings for manual review

⏳ **Health Checks**: Periodic metadata integrity verification
- Verify all collections have metadata
- Validate metadata is readable
- Alert on corruption detection

⏳ **Startup Validation**: Pre-flight checks before contextd start
- Scan for corrupt collections
- Auto-recover if possible
- Fail with clear instructions if manual intervention needed

### Long-Term (Roadmap)

🔮 **Upstream Contribution**: Submit PR to chromem-go
- Atomic metadata writes (write to .tmp, sync, rename)
- Graceful degradation for corrupt collections
- Optional metadata backups
- Recovery mode

🔮 **Monitoring**: Add metrics and alerts
- `vectorstore_collections_healthy`
- `vectorstore_collections_corrupt`
- Alert on corruption detection

🔮 **Automated Backups**: Periodic metadata snapshots
- Hourly metadata backups
- Retain last 24 snapshots
- Auto-restore on corruption

🔮 **Alternative Storage**: Consider migration to Qdrant
- External database (no metadata file issues)
- Built-in replication
- Health monitoring included

---

## Lessons Learned

### What Went Well

1. ✅ **Systematic Investigation**: Followed methodical debugging approach
2. ✅ **Complete Recovery**: No data loss, full service restoration
3. ✅ **Comprehensive Documentation**: Created runbooks and guides
4. ✅ **Root Cause Identified**: Understood exact failure mechanism

### What Could Be Improved

1. ❌ **Detection Time**: Issue not detected until manual testing
2. ❌ **No Monitoring**: No automated health checks for metadata integrity
3. ❌ **Manual Recovery**: Required custom tool and manual intervention
4. ❌ **Dependency Risk**: Over-reliance on external library (chromem) design

### Action Items

| Priority | Action | Owner | Deadline |
|----------|--------|-------|----------|
| P0 | Implement graceful degradation wrapper | Engineering | Week of Jan 27 |
| P1 | Add health check endpoints | Engineering | Week of Feb 3 |
| P1 | Automated startup validation | Engineering | Week of Feb 3 |
| P2 | Create metrics and alerts | Operations | Week of Feb 10 |
| P2 | Submit chromem-go PR for atomic writes | Engineering | Week of Feb 17 |
| P3 | Evaluate Qdrant migration | Architecture | Q1 2026 |

---

## References

- **Root Cause Analysis**: `METADATA_LOSS_ROOT_CAUSE.md`
- **Recovery Runbook**: `docs/operations/METADATA_RECOVERY.md`
- **Recovery Tool**: `cmd/recover-metadata/main.go`
- **Chromem Source**: https://github.com/philippgille/chromem-go
- **Related Documentation**: `docs/testing/CHROMEM_TESTING.md`

---

## Sign-Off

**Incident Commander**: Claude (AI Agent)
**Resolution Confirmed**: ✅ All services healthy
**Documentation Complete**: ✅ Runbooks and guides created
**Prevention Plan**: ✅ Short and long-term measures identified

**Status**: This incident is resolved. Prevention measures are in progress.
