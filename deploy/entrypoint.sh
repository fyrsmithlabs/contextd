#!/bin/sh
set -e

# Create data directories
mkdir -p /data/qdrant/storage
mkdir -p /data/logs

# Set Qdrant storage path
export QDRANT__STORAGE__STORAGE_PATH=/data/qdrant/storage

# Disable telemetry by default (no OTEL collector in container)
# Override with -e TELEMETRY_ENABLED=true to enable
export TELEMETRY_ENABLED=${TELEMETRY_ENABLED:-false}

# Start Qdrant in background first
/usr/local/bin/qdrant &
QDRANT_PID=$!

# Wait for Qdrant to be ready
echo "Waiting for Qdrant..."
for i in $(seq 1 30); do
    if wget -q --spider http://localhost:6333/readyz 2>/dev/null; then
        echo "Qdrant is ready"
        break
    fi
    sleep 1
done

# Start contextd (stdio mode for MCP)
exec /usr/local/bin/contextd -mcp
