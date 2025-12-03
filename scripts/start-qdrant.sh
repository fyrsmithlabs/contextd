#!/bin/bash
# Start Qdrant for contextd (Homebrew/binary installations)
# Usage: ./scripts/start-qdrant.sh [start|stop|status]

set -e

CONTAINER_NAME="contextd-qdrant"
VOLUME_NAME="contextd-qdrant-data"
QDRANT_VERSION="v1.12.1"

start_qdrant() {
    if docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Qdrant is already running"
        docker ps -f name="$CONTAINER_NAME" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
        return 0
    fi

    echo "Starting Qdrant $QDRANT_VERSION..."
    docker run -d \
        --name "$CONTAINER_NAME" \
        -p 6333:6333 \
        -p 6334:6334 \
        -v "$VOLUME_NAME:/qdrant/storage" \
        -e QDRANT__SERVICE__GRPC_PORT=6334 \
        --restart unless-stopped \
        "qdrant/qdrant:$QDRANT_VERSION"

    echo "Waiting for Qdrant to be ready..."
    for i in {1..30}; do
        if curl -sf http://localhost:6333/readyz > /dev/null 2>&1; then
            echo "Qdrant is ready!"
            echo ""
            echo "REST API: http://localhost:6333"
            echo "gRPC API: localhost:6334"
            echo ""
            echo "Data persisted in Docker volume: $VOLUME_NAME"
            return 0
        fi
        sleep 1
    done

    echo "Warning: Qdrant may not be fully ready yet"
    return 1
}

stop_qdrant() {
    if docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Stopping Qdrant..."
        docker stop "$CONTAINER_NAME"
        docker rm "$CONTAINER_NAME"
        echo "Qdrant stopped. Data preserved in volume: $VOLUME_NAME"
    else
        echo "Qdrant is not running"
    fi
}

status_qdrant() {
    if docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Qdrant is running"
        docker ps -f name="$CONTAINER_NAME" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
        echo ""
        curl -sf http://localhost:6333/readyz && echo "Health: OK" || echo "Health: Not ready"
    else
        echo "Qdrant is not running"
        if docker ps -aq -f name="$CONTAINER_NAME" | grep -q .; then
            echo "(Container exists but is stopped)"
        fi
    fi
}

case "${1:-start}" in
    start)
        start_qdrant
        ;;
    stop)
        stop_qdrant
        ;;
    status)
        status_qdrant
        ;;
    restart)
        stop_qdrant
        start_qdrant
        ;;
    *)
        echo "Usage: $0 [start|stop|status|restart]"
        exit 1
        ;;
esac
