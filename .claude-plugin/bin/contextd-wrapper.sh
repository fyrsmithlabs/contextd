#!/usr/bin/env bash
#
# contextd MCP wrapper - downloads binary on first run
#

set -e

CONTEXTD_VERSION="${CONTEXTD_VERSION:-latest}"
INSTALL_DIR="${HOME}/.local/bin"
BINARY_PATH="${INSTALL_DIR}/contextd"

# Detect platform
detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "${arch}" in
    x86_64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "Unsupported architecture: ${arch}" >&2; exit 1 ;;
  esac

  echo "${os}_${arch}"
}

# Get latest version from GitHub
get_latest_version() {
  curl -fsSL "https://api.github.com/repos/fyrsmithlabs/contextd/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install contextd
install_contextd() {
  local platform version version_no_v url temp_dir

  platform="$(detect_platform)"

  if [ "${CONTEXTD_VERSION}" = "latest" ]; then
    version="$(get_latest_version)"
  else
    version="${CONTEXTD_VERSION}"
  fi

  # Remove v prefix for filename
  version_no_v="${version#v}"

  url="https://github.com/fyrsmithlabs/contextd/releases/download/${version}/contextd_${version_no_v}_${platform}.tar.gz"

  echo "Downloading contextd ${version} for ${platform}..." >&2
  echo "URL: ${url}" >&2

  # Create install directory
  mkdir -p "${INSTALL_DIR}"

  # Download and extract
  temp_dir="$(mktemp -d)"
  trap "rm -rf '${temp_dir}'" EXIT

  if ! curl -fsSL "${url}" | tar -xz -C "${temp_dir}"; then
    echo "Failed to download contextd. Please install manually:" >&2
    echo "  brew install fyrsmithlabs/tap/contextd" >&2
    exit 1
  fi

  # Move binary to install dir
  mv "${temp_dir}/contextd" "${BINARY_PATH}"
  chmod +x "${BINARY_PATH}"

  echo "✓ contextd installed to ${BINARY_PATH}" >&2
}

# Check if contextd exists and is executable
if [ ! -x "${BINARY_PATH}" ]; then
  # Also check if it's in PATH
  if ! command -v contextd &>/dev/null; then
    install_contextd
  else
    BINARY_PATH="$(command -v contextd)"
  fi
fi

# Execute contextd with MCP args
# Note: --no-http added in v0.2.0-rc8+, check if supported
if "${BINARY_PATH}" --help 2>&1 | grep -q "no-http"; then
  exec "${BINARY_PATH}" --mcp --no-http "$@"
else
  exec "${BINARY_PATH}" --mcp "$@"
fi
