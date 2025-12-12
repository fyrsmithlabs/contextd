#!/usr/bin/env bash
#
# contextd MCP Server Setup (Docker)
# Configures contextd to run via Docker container
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; exit 1; }

# Check if Docker is available
check_docker() {
    if ! command -v docker &>/dev/null; then
        error "Docker is not installed or not in PATH.

Please install Docker:
  - macOS: https://docs.docker.com/desktop/install/mac-install/
  - Linux: https://docs.docker.com/engine/install/
  - Windows: https://docs.docker.com/desktop/install/windows-install/"
    fi

    if ! docker info &>/dev/null; then
        error "Docker daemon is not running. Please start Docker Desktop or the Docker service."
    fi

    info "Docker is available"
}

# Pull the latest contextd image
pull_image() {
    local tag="${CONTEXTD_VERSION:-latest}"
    local image="ghcr.io/fyrsmithlabs/contextd:${tag}"

    info "Pulling contextd Docker image: $image"

    if ! docker pull "$image"; then
        error "Failed to pull image: $image

This may happen if:
  - The image tag does not exist
  - Network connectivity issues
  - GitHub Container Registry is temporarily unavailable

Try:
  docker pull ghcr.io/fyrsmithlabs/contextd:latest"
    fi

    info "Image pulled successfully"
    echo "$image"
}

# Generate MCP configuration for Docker
generate_mcp_config() {
    local image="$1"

    cat <<EOF
{
  "contextd": {
    "type": "stdio",
    "command": "docker",
    "args": [
      "run",
      "-i",
      "--rm",
      "-v",
      "contextd-data:/data",
      "-v",
      "\${HOME}:\${HOME}",
      "$image"
    ],
    "env": {}
  }
}
EOF
}

# Main
main() {
    echo "========================================"
    echo "  contextd MCP Server Setup (Docker)"
    echo "========================================"
    echo

    # Check Docker
    check_docker

    # Pull image
    local image=$(pull_image)

    echo
    echo "========================================"
    echo "  MCP Configuration"
    echo "========================================"
    echo
    echo "Add this to your Claude Code MCP settings:"
    echo
    generate_mcp_config "$image"
    echo
    echo "========================================"
    echo "  Notes"
    echo "========================================"
    echo
    echo "1. Data persists in Docker volume: contextd-data"
    echo "   To backup: docker run --rm -v contextd-data:/data -v \$(pwd):/backup alpine tar czf /backup/contextd-backup.tar.gz /data"
    echo
    echo "2. The \${HOME}:\${HOME} mount allows contextd to index"
    echo "   repositories in your home directory."
    echo
    echo "3. To use a specific version:"
    echo "   CONTEXTD_VERSION=v0.2.0-rc7 ./mcp-setup-docker.sh"
    echo
}

# Run if executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
