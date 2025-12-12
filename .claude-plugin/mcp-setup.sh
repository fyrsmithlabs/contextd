#!/usr/bin/env bash
#
# contextd MCP Server Setup
# Downloads and installs the contextd binary for the current OS/architecture
#

set -e

# Input validation for VERSION
VERSION="${CONTEXTD_VERSION:-latest}"
if [[ ! "$VERSION" =~ ^[a-zA-Z0-9._-]+$ ]]; then
    echo -e "\033[0;31m[ERROR]\033[0m Invalid CONTEXTD_VERSION: contains disallowed characters" >&2
    exit 1
fi

# Input validation for INSTALL_DIR - must be within user home directory
INSTALL_DIR="${CONTEXTD_INSTALL_DIR:-$HOME/.local/bin}"
# Resolve to absolute path and check it's under HOME
INSTALL_DIR_RESOLVED=$(cd "$HOME" && realpath -m "$INSTALL_DIR" 2>/dev/null || echo "$INSTALL_DIR")
if [[ ! "$INSTALL_DIR_RESOLVED" =~ ^"$HOME"(/|$) ]]; then
    echo -e "\033[0;31m[ERROR]\033[0m CONTEXTD_INSTALL_DIR must be within user home directory" >&2
    exit 1
fi
INSTALL_DIR="$INSTALL_DIR_RESOLVED"

REPO="fyrsmithlabs/contextd"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; exit 1; }

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*)
            error "Windows is not yet supported via this installer.

For Windows installation:
  1. Visit: https://github.com/fyrsmithlabs/contextd/releases/latest
  2. Download the appropriate Windows binary manually
  3. Add to your PATH

See the releases page for manual installation instructions."
            ;;
        *) error "Unsupported operating system: $(uname -s)" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        armv7l) echo "arm" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# Get latest release version from GitHub
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
        | grep '"tag_name"' \
        | sed -E 's/.*"([^"]+)".*/\1/' \
        || echo ""
}

# Get latest pre-release version (for rc releases)
get_latest_prerelease() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases" 2>/dev/null \
        | grep '"tag_name"' \
        | head -1 \
        | sed -E 's/.*"([^"]+)".*/\1/' \
        || echo ""
}

# Check if contextd is already installed and get version
check_existing() {
    if command -v contextd &>/dev/null; then
        local existing_path=$(command -v contextd)
        info "contextd already installed at: $existing_path"
        if contextd --version 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Download and install
install_contextd() {
    local os=$(detect_os)
    local arch=$(detect_arch)

    info "Detected: OS=$os, ARCH=$arch"

    # Resolve version
    if [[ "$VERSION" == "latest" ]]; then
        VERSION=$(get_latest_version)
        if [[ -z "$VERSION" ]]; then
            # Fall back to pre-release if no stable release
            VERSION=$(get_latest_prerelease)
        fi
    fi

    if [[ -z "$VERSION" ]]; then
        error "Could not determine version to install"
    fi

    # Strip 'v' prefix for filename
    local version_num="${VERSION#v}"

    info "Installing contextd ${VERSION}..."

    # Build download URL
    local filename="contextd_${version_num}_${os}_${arch}.tar.gz"
    local url="https://github.com/${REPO}/releases/download/${VERSION}/${filename}"

    info "Downloading from: $url"

    # Create temp directory with safe cleanup
    local tmpdir
    tmpdir=$(mktemp -d)
    # Safety checks for tmpdir before cleanup
    cleanup_tmpdir() {
        if [[ -n "$tmpdir" && -d "$tmpdir" && "$tmpdir" == /tmp/* ]]; then
            rm -rf "$tmpdir"
        fi
    }
    trap cleanup_tmpdir EXIT

    # Download binary archive
    if ! curl -fsSL "$url" -o "$tmpdir/contextd.tar.gz"; then
        error "Failed to download contextd from:
  $url

This may happen if:
  - The release does not exist for ${os}/${arch}
  - Network connectivity issues
  - GitHub is temporarily unavailable

Manual installation:
  1. Visit: https://github.com/fyrsmithlabs/contextd/releases/latest
  2. Download: contextd_*_${os}_${arch}.tar.gz
  3. Extract and move binary to: $INSTALL_DIR/contextd
  4. Run: chmod +x $INSTALL_DIR/contextd"
    fi

    # Download and verify SHA256 checksum
    local checksum_url="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"
    info "Verifying integrity with SHA256 checksum..."
    if curl -fsSL "$checksum_url" -o "$tmpdir/checksums.txt" 2>/dev/null; then
        # Extract expected checksum for our file
        local expected_checksum
        expected_checksum=$(grep "$filename" "$tmpdir/checksums.txt" | awk '{print $1}')
        if [[ -n "$expected_checksum" ]]; then
            # Calculate actual checksum
            local actual_checksum
            if command -v sha256sum &>/dev/null; then
                actual_checksum=$(sha256sum "$tmpdir/contextd.tar.gz" | awk '{print $1}')
            elif command -v shasum &>/dev/null; then
                actual_checksum=$(shasum -a 256 "$tmpdir/contextd.tar.gz" | awk '{print $1}')
            else
                warn "sha256sum/shasum not available, skipping checksum verification"
                actual_checksum=""
            fi

            if [[ -n "$actual_checksum" ]]; then
                if [[ "$expected_checksum" != "$actual_checksum" ]]; then
                    error "Checksum verification failed! Expected: $expected_checksum, Got: $actual_checksum"
                fi
                info "Checksum verified successfully"
            fi
        else
            warn "Could not find checksum for $filename in checksums.txt"
        fi
    else
        warn "Could not download checksums.txt, skipping integrity verification"
    fi

    # Extract
    tar -xzf "$tmpdir/contextd.tar.gz" -C "$tmpdir"

    # Find binary
    local binary=""
    if [[ -f "$tmpdir/contextd" ]]; then
        binary="$tmpdir/contextd"
    elif [[ -f "$tmpdir/contextd.exe" ]]; then
        binary="$tmpdir/contextd.exe"
    else
        error "Binary not found in archive"
    fi

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    # Install binary
    local target="$INSTALL_DIR/contextd"
    [[ "$os" == "windows" ]] && target="$INSTALL_DIR/contextd.exe"

    mv "$binary" "$target"
    chmod +x "$target"

    info "Installed to: $target"

    # Also install ctxd if present
    if [[ -f "$tmpdir/ctxd" ]]; then
        mv "$tmpdir/ctxd" "$INSTALL_DIR/ctxd"
        chmod +x "$INSTALL_DIR/ctxd"
        info "Also installed: $INSTALL_DIR/ctxd"
    fi

    # Verify installation
    if "$target" --version 2>/dev/null; then
        info "Installation successful!"
    else
        warn "Binary installed but version check failed"
    fi

    # Check PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        warn "Add to your PATH: export PATH=\"\$PATH:$INSTALL_DIR\""
    fi

    echo "$target"
}

# Generate MCP configuration
generate_mcp_config() {
    local binary_path="$1"

    cat <<EOF
{
  "contextd": {
    "type": "stdio",
    "command": "$binary_path",
    "args": [],
    "env": {}
  }
}
EOF
}

# Main
main() {
    echo "========================================"
    echo "  contextd MCP Server Setup"
    echo "========================================"
    echo

    # Check for existing installation
    if check_existing; then
        read -p "Reinstall/upgrade? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            info "Using existing installation"
            echo "$(command -v contextd)"
            exit 0
        fi
    fi

    # Install
    local binary_path=$(install_contextd)

    echo
    echo "========================================"
    echo "  MCP Configuration"
    echo "========================================"
    echo
    echo "Add this to your Claude Code MCP settings:"
    echo
    generate_mcp_config "$binary_path"
    echo
    echo "========================================"
    echo "  Verification Steps"
    echo "========================================"
    echo
    echo "1. Verify PATH is set correctly:"
    echo "   which contextd"
    echo "   # Should output: $INSTALL_DIR/contextd"
    echo
    echo "2. If 'contextd' is not found, add to your shell profile:"
    echo "   # For bash (~/.bashrc or ~/.bash_profile):"
    echo "   echo 'export PATH=\"\$PATH:$INSTALL_DIR\"' >> ~/.bashrc"
    echo "   source ~/.bashrc"
    echo
    echo "   # For zsh (~/.zshrc):"
    echo "   echo 'export PATH=\"\$PATH:$INSTALL_DIR\"' >> ~/.zshrc"
    echo "   source ~/.zshrc"
    echo
    echo "3. Restart your terminal or run 'source ~/.bashrc' (or ~/.zshrc)"
    echo
    echo "4. Verify MCP config in Claude Code:"
    echo "   - Open Claude Code settings"
    echo "   - Navigate to MCP Servers section"
    echo "   - Add the configuration shown above"
    echo "   - Restart Claude Code to load the new MCP server"
    echo
}

# Run if executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
