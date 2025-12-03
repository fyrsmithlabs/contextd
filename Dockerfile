# Stage 1: Build contextd
FROM golang:1.24rc1-bookworm AS builder

# Allow toolchain upgrade
ENV GOTOOLCHAIN=auto

WORKDIR /build

# Install ONNX runtime for CGO build
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Download ONNX runtime (v1.23.2 matches onnxruntime_go v1.23.0)
RUN wget -q https://github.com/microsoft/onnxruntime/releases/download/v1.23.2/onnxruntime-linux-x64-1.23.2.tgz \
    && tar xzf onnxruntime-linux-x64-1.23.2.tgz \
    && cp onnxruntime-linux-x64-1.23.2/lib/*.so* /usr/lib/ \
    && ldconfig \
    && rm -rf onnxruntime-linux-x64-1.23.2*

# Copy go modules first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with CGO for ONNX support
ENV CGO_ENABLED=1
RUN go build -o contextd ./cmd/contextd

# Pre-download FastEmbed model during build
RUN mkdir -p /models && \
    ./contextd --download-models 2>/dev/null || true

# Stage 2: Runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    supervisor \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Install ONNX runtime (v1.23.2 matches onnxruntime_go v1.23.0)
RUN wget -q https://github.com/microsoft/onnxruntime/releases/download/v1.23.2/onnxruntime-linux-x64-1.23.2.tgz \
    && tar xzf onnxruntime-linux-x64-1.23.2.tgz \
    && cp onnxruntime-linux-x64-1.23.2/lib/*.so* /usr/lib/ \
    && ldconfig \
    && rm -rf onnxruntime-linux-x64-1.23.2*

# Install Qdrant
RUN wget -q https://github.com/qdrant/qdrant/releases/download/v1.12.1/qdrant-x86_64-unknown-linux-gnu.tar.gz \
    && tar xzf qdrant-x86_64-unknown-linux-gnu.tar.gz \
    && mv qdrant /usr/local/bin/ \
    && rm qdrant-x86_64-unknown-linux-gnu.tar.gz

# Copy contextd binary
COPY --from=builder /build/contextd /usr/local/bin/contextd

# Copy pre-downloaded models
COPY --from=builder /models /root/.cache/contextd/models

# Copy config files
COPY deploy/supervisord.conf /etc/supervisor/conf.d/contextd.conf
COPY deploy/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Data volume
VOLUME /data

# Expose ports (optional, for debugging)
EXPOSE 6333 6334 9090

# Environment defaults
ENV QDRANT_HOST=localhost \
    QDRANT_PORT=6334 \
    CONTEXTD_DATA_PATH=/data \
    EMBEDDINGS_PROVIDER=fastembed \
    EMBEDDINGS_MODEL=BAAI/bge-small-en-v1.5 \
    ONNX_PATH=/usr/lib/libonnxruntime.so

ENTRYPOINT ["/entrypoint.sh"]
