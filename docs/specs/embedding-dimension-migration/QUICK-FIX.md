# Quick Fix: Dimension Mismatch Issue

**Problem**: Checkpoint search failing with dimension mismatch error
**Solution**: Set correct `EMBEDDING_DIM` and recreate collections

## Option 1: Quick Fix (No Data Preservation)

⚠️ **WARNING**: This will DELETE all existing checkpoints, remediations, and skills.

```bash
# 1. Stop contextd
systemctl --user stop contextd

  python3 << 'EOF'
connections.connect(host='localhost', port='19530')
collections = ['checkpoints', 'remediations', 'skills', 'troubleshooting_knowledge', 'session_notes', 'research_documents']
for coll in collections:
    if utility.has_collection(coll):
        utility.drop_collection(coll)
        print(f'Dropped collection: {coll}')
EOF
"

# 3. Set correct dimension in systemd service
mkdir -p ~/.config/systemd/user/contextd.service.d/
cat > ~/.config/systemd/user/contextd.service.d/override.conf << 'EOF'
[Service]
Environment="EMBEDDING_DIM=384"
EOF

# 4. Reload and restart
systemctl --user daemon-reload
systemctl --user restart contextd

# 5. Verify
journalctl --user -u contextd -f
```

## Option 2: Data Preservation (Manual Export/Import)

### Step 1: Export Existing Data

```bash
# 1. Set dimension to 1536 temporarily (match existing collections)
export EMBEDDING_DIM=1536
export EMBEDDING_BASE_URL=http://localhost:8080/v1
export EMBEDDING_MODEL=BAAI/bge-small-en-v1.5

# 2. Use ctxd to export data (if available)
ctxd checkpoint list --limit=1000 > /tmp/checkpoints-export.json

  python3 << 'EOF'
import json

connections.connect(host='localhost', port='19530')

# Export checkpoints
coll = Collection('checkpoints')
coll.load()
results = coll.query(expr='id != \"\"', output_fields=['id', 'summary', 'description', 'project_path', 'tags', 'context', 'created_at'])

with open('/tmp/checkpoints-export.json', 'w') as f:
    json.dump(results, f, indent=2)

print(f'Exported {len(results)} checkpoints')
EOF
"

# Copy export out of container
```

### Step 2: Recreate Collections

```bash
# 1. Stop contextd
systemctl --user stop contextd

# 2. Drop collections (same as Option 1 step 2)
# ... (see above)

# 3. Set correct dimension
mkdir -p ~/.config/systemd/user/contextd.service.d/
cat > ~/.config/systemd/user/contextd.service.d/override.conf << 'EOF'
[Service]
Environment="EMBEDDING_DIM=384"
EOF

# 4. Restart contextd (will create collections with correct dimension)
systemctl --user daemon-reload
systemctl --user restart contextd
```

### Step 3: Re-import Data (Requires Migration Tool)

⚠️ This step requires the migration tool from the SPEC.md to be implemented.

```bash
# Once migration tool is built:
ctxd migrate --import=/tmp/checkpoints-export.json --dimension=384
```

## Option 3: Switch to OpenAI (Preserve Existing Collections)

If you want to use OpenAI API instead of TEI:

```bash
# 1. Stop contextd
systemctl --user stop contextd

# 2. Set OpenAI configuration
mkdir -p ~/.config/systemd/user/contextd.service.d/
cat > ~/.config/systemd/user/contextd.service.d/override.conf << 'EOF'
[Service]
Environment="EMBEDDING_DIM=1536"
Environment="EMBEDDING_BASE_URL="
Environment="EMBEDDING_MODEL=text-embedding-3-small"
Environment="OPENAI_API_KEY=sk-your-key-here"
EOF

# 3. Restart
systemctl --user daemon-reload
systemctl --user restart contextd

# 4. Test
journalctl --user -u contextd -f
```

## Verification

After applying any fix:

```bash
# 1. Check contextd status
systemctl --user status contextd

# 2. Test checkpoint search
echo '{"method":"tools/call","params":{"name":"checkpoint_search","arguments":{"query":"test"}}}' | \
  curl -s --unix-socket ~/.config/contextd/api.sock \
  -X POST -H "Content-Type: application/json" -d @-

# 3. Check dimension in logs
journalctl --user -u contextd --since "5 minutes ago" | grep -i dimension
```

## Recommended Approach

**For immediate fix**: Use **Option 1** (data loss acceptable) or **Option 3** (switch to OpenAI)

**For production**: Wait for migration tool implementation (SPEC.md Phase 1-3)

## Next Steps

1. **Immediate**: Choose Option 1 or 3 above
2. **Short-term**: Implement migration tool (see SPEC.md)
3. **Long-term**: Add automatic dimension detection to prevent this issue

## References

- Full Specification: `docs/specs/embedding-dimension-migration/SPEC.md`
- Embedding Config: `pkg/embedding/config.go`
- Vector Store Adapters: `pkg/vectorstore/adapter/*/adapter.go`
