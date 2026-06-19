#!/usr/bin/env bash
# fetch-macos-sdk.sh - Download and extract a macOS SDK for cross-compiling the
# darwin targets with `zig cc`. Darwin cgo (e.g. prometheus/client_golang's
# memory collector via <mach/mach_vm.h>) needs the macOS SDK headers + the .tbd
# stub libs; zig does not bundle them.
#
# We pin a pre-visionOS SDK (12.3). Newer SDKs reference the "visionos"
# availability platform that zig 0.14's bundled clang rejects, breaking the cgo
# preprocessor probes.
#
# Prints the extracted SDK path on stdout so callers can capture it, e.g.:
#   export MACOS_SDK_ROOT="$(bash scripts/fetch-macos-sdk.sh)"
#
# Idempotent: re-uses an already-extracted SDK.
set -euo pipefail

MACOS_SDK_VERSION="${MACOS_SDK_VERSION:-12.3}"
SDK_URL="https://github.com/joseluisq/macosx-sdks/releases/download/${MACOS_SDK_VERSION}/MacOSX${MACOS_SDK_VERSION}.sdk.tar.xz"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SDK_BASE="${ROOT_DIR}/build/macos-sdk"
SDK_DIR="${SDK_BASE}/MacOSX${MACOS_SDK_VERSION}.sdk"

if [[ -f "${SDK_DIR}/usr/include/mach/mach_vm.h" ]]; then
  echo "${SDK_DIR}"
  exit 0
fi

mkdir -p "${SDK_BASE}"
tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT

echo "Fetching macOS SDK ${MACOS_SDK_VERSION}..." >&2
curl -fsSL -o "${tmp}/sdk.tar.xz" "${SDK_URL}"
tar -xf "${tmp}/sdk.tar.xz" -C "${SDK_BASE}"

if [[ ! -f "${SDK_DIR}/usr/include/mach/mach_vm.h" ]]; then
  echo "fetch-macos-sdk.sh: SDK extracted but mach/mach_vm.h missing under ${SDK_DIR}" >&2
  exit 1
fi

echo "macOS SDK ready at ${SDK_DIR}" >&2
echo "${SDK_DIR}"
