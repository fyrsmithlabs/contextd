#!/usr/bin/env bash
#
# contextd MCP wrapper - downloads binary on first run
#

set -e

CONTEXTD_VERSION="${CONTEXTD_VERSION:-latest}"
INSTALL_DIR="${HOME}/.local/bin"
BINARY_PATH="${INSTALL_DIR}/contextd"
CURL_TIMEOUT="${CURL_TIMEOUT:-60}"

# Detect platform
detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "${arch}" in
    x86_64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "❌ Unsupported architecture: ${arch}" >&2; exit 1 ;;
  esac

  echo "${os}_${arch}"
}

# Get latest version from GitHub
get_latest_version() {
  curl -fsSL --connect-timeout 10 --max-time 30 \
    "https://api.github.com/repos/fyrsmithlabs/contextd/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"([^"]+)".*/\1/'
}

# Show platform-appropriate install instructions
show_install_help() {
  local platform="$1"
  echo "" >&2
  echo "Please install manually:" >&2
  case "${platform}" in
    darwin_*)
      echo "  brew install fyrsmithlabs/tap/contextd" >&2
      ;;
    linux_*)
      echo "  # Download from GitHub releases:" >&2
      echo "  https://github.com/fyrsmithlabs/contextd/releases/latest" >&2
      echo "" >&2
      echo "  # Or build from source:" >&2
      echo "  go install github.com/fyrsmithlabs/contextd/cmd/contextd@latest" >&2
      ;;
    *)
      echo "  https://github.com/fyrsmithlabs/contextd/releases/latest" >&2
      ;;
  esac
}

# Download and install contextd
install_contextd() {
  local platform version version_no_v url temp_dir os

  platform="$(detect_platform)"
  os="${platform%%_*}"

  echo "⏳ Checking latest contextd version..." >&2

  if [ "${CONTEXTD_VERSION}" = "latest" ]; then
    version="$(get_latest_version)"
    if [ -z "${version}" ]; then
      echo "❌ Failed to get latest version from GitHub API" >&2
      echo "   This may be due to rate limiting or network issues." >&2
      echo "   Try setting CONTEXTD_VERSION=v0.2.0-rc7 (or desired version)" >&2
      show_install_help "${platform}"
      exit 1
    fi
  else
    version="${CONTEXTD_VERSION}"
  fi

  # Remove v prefix for filename
  version_no_v="${version#v}"

  url="https://github.com/fyrsmithlabs/contextd/releases/download/${version}/contextd_${version_no_v}_${platform}.tar.gz"

  echo "📦 Downloading contextd ${version} for ${platform}..." >&2

  # Create install directory
  if ! mkdir -p "${INSTALL_DIR}" 2>/dev/null; then
    echo "❌ Cannot create directory: ${INSTALL_DIR}" >&2
    echo "   Check permissions or set INSTALL_DIR to a writable location." >&2
    exit 1
  fi

  # Download and extract
  temp_dir="$(mktemp -d)"
  trap "rm -rf '${temp_dir}'" EXIT INT TERM

  # Download with progress bar and timeout
  if ! curl -fSL --connect-timeout 10 --max-time "${CURL_TIMEOUT}" \
       --progress-bar "${url}" | tar -xz -C "${temp_dir}"; then
    echo "" >&2
    echo "❌ Failed to download contextd" >&2
    show_install_help "${platform}"
    exit 1
  fi

  # Verify binary was extracted
  if [ ! -f "${temp_dir}/contextd" ]; then
    echo "❌ Downloaded archive does not contain contextd binary" >&2
    show_install_help "${platform}"
    exit 1
  fi

  # Move binary to install dir
  mv "${temp_dir}/contextd" "${BINARY_PATH}"
  chmod +x "${BINARY_PATH}"

  echo "✅ contextd ${version} installed to ${BINARY_PATH}" >&2

  # Warn if not in PATH
  if ! echo "${PATH}" | tr ':' '\n' | grep -qx "${INSTALL_DIR}"; then
    echo "" >&2
    echo "⚠️  Note: ${INSTALL_DIR} is not in your PATH" >&2
    echo "   Add to your shell config (~/.bashrc or ~/.zshrc):" >&2
    echo "   export PATH=\"\$HOME/.local/bin:\$PATH\"" >&2
  fi
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
