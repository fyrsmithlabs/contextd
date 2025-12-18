#!/bin/sh
# Wait for test services to be ready

set -e

TIMEOUT=${TIMEOUT:-60}
QDRANT_HOST=${QDRANT_HOST:-localhost}
# Use REST port (6333) for readiness check, not gRPC port
QDRANT_REST_PORT=${QDRANT_REST_PORT:-6333}

echo "Waiting for Qdrant at $QDRANT_HOST:$QDRANT_REST_PORT..."

start_time=$(date +%s)
while true; do
    if curl -sf "http://$QDRANT_HOST:$QDRANT_REST_PORT/readyz" > /dev/null 2>&1; then
        echo "Qdrant is ready!"
        break
    fi

    current_time=$(date +%s)
    elapsed=$((current_time - start_time))

    if [ $elapsed -ge $TIMEOUT ]; then
        echo "Timeout waiting for Qdrant after ${TIMEOUT}s"
        exit 1
    fi

    echo "  Waiting... (${elapsed}s elapsed)"
    sleep 2
done

echo "All services are ready!"
