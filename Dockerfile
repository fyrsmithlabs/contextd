# Stage 1: Build contextd
# Multi-arch builds use QEMU emulation for arm64 because CGO requires native compilation
FROM golang:1.24rc1-bookworm AS builder

ARG TARGETARCH
ARG TARGETOS

# Allow toolchain upgrade
ENV GOTOOLCHAIN=auto

WORKDIR /build

# Install ONNX runtime for CGO build (architecture-aware)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Download ONNX runtime (v1.23.2 matches onnxruntime_go v1.23.0)
RUN set -ex; \
    case "${TARGETARCH}" in \
        amd64) ONNX_ARCH="x64" ;; \
        arm64) ONNX_ARCH="aarch64" ;; \
        *) echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac; \
    wget -q "https://github.com/microsoft/onnxruntime/releases/download/v1.23.2/onnxruntime-linux-${ONNX_ARCH}-1.23.2.tgz" && \
    tar xzf "onnxruntime-linux-${ONNX_ARCH}-1.23.2.tgz" && \
    cp onnxruntime-linux-${ONNX_ARCH}-1.23.2/lib/*.so* /usr/lib/ && \
    ldconfig && \
    rm -rf onnxruntime-linux-${ONNX_ARCH}-1.23.2*

# Copy go modules first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with CGO for ONNX support
ENV CGO_ENABLED=1
RUN go build -o contextd ./cmd/contextd

# Pre-download FastEmbed model during build
RUN mkdir -p /models && ./contextd --download-models 2>/dev/null || true

# Stage 2: Runtime
FROM debian:bookworm-slim

ARG TARGETARCH

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Install ONNX runtime (v1.23.2 matches onnxruntime_go v1.23.0)
RUN set -ex; \
    case "${TARGETARCH}" in \
        amd64) ONNX_ARCH="x64" ;; \
        arm64) ONNX_ARCH="aarch64" ;; \
        *) echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac; \
    wget -q "https://github.com/microsoft/onnxruntime/releases/download/v1.23.2/onnxruntime-linux-${ONNX_ARCH}-1.23.2.tgz" && \
    tar xzf "onnxruntime-linux-${ONNX_ARCH}-1.23.2.tgz" && \
    cp onnxruntime-linux-${ONNX_ARCH}-1.23.2/lib/*.so* /usr/lib/ && \
    ldconfig && \
    rm -rf onnxruntime-linux-${ONNX_ARCH}-1.23.2*

# Install Qdrant (architecture-aware) - use musl builds for static linking (no glibc deps)
RUN set -ex; \
    QDRANT_VERSION="1.16.1"; \
    case "${TARGETARCH}" in \
        amd64) QDRANT_ARCH="x86_64-unknown-linux-musl" ;; \
        arm64) QDRANT_ARCH="aarch64-unknown-linux-musl" ;; \
        *) echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac; \
    wget -q "https://github.com/qdrant/qdrant/releases/download/v${QDRANT_VERSION}/qdrant-${QDRANT_ARCH}.tar.gz" && \
    tar xzf "qdrant-${QDRANT_ARCH}.tar.gz" && \
    mv qdrant /usr/local/bin/ && \
    rm "qdrant-${QDRANT_ARCH}.tar.gz"

# Copy contextd binary
COPY --from=builder /build/contextd /usr/local/bin/contextd

# Copy pre-downloaded models (may be empty on cross-compile)
COPY --from=builder /models /root/.cache/contextd/models

# Copy config files
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
