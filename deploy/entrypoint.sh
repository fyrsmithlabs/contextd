#!/bin/sh
set -e

# Working directories (pre-created in Dockerfile, ensure they exist for volume mounts)
mkdir -p /data/logs /data/chromem /data/models 2>/dev/null || true

# Disable telemetry by default (no OTEL collector in container)
# Override with -e TELEMETRY_ENABLED=true to enable
export TELEMETRY_ENABLED=${TELEMETRY_ENABLED:-false}

# Set ONNX runtime path for FastEmbed
export ONNX_PATH=${ONNX_PATH:-/usr/local/lib/libonnxruntime.so}

# Set embeddings cache directory
export EMBEDDINGS_CACHE_DIR=${EMBEDDINGS_CACHE_DIR:-/data/models}

# Check vectorstore provider (default: chromem)
VECTORSTORE_PROVIDER=${CONTEXTD_VECTORSTORE_PROVIDER:-chromem}

if [ "$VECTORSTORE_PROVIDER" = "qdrant" ]; then
    # Create Qdrant directories
    mkdir -p /data/qdrant/storage 2>/dev/null || true
    export QDRANT__STORAGE__STORAGE_PATH=/data/qdrant/storage

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
else
    # chromem (default) - embedded database, no external service needed
    export CONTEXTD_VECTORSTORE_CHROMEM_PATH=${CONTEXTD_VECTORSTORE_CHROMEM_PATH:-/data/chromem}
fi

# Start contextd (stdio mode for MCP)
exec /usr/local/bin/contextd -mcp
