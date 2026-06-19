#!/usr/bin/env bash
# zig-cxx.sh - C++ compiler wrapper that dispatches to `zig c++` for the target
# GOOS/GOARCH currently being built. Companion to zig-cc.sh (see it for full
# rationale on zig version, macOS SDK, and cache requirements).
#
# Usage (set as CXX):  CXX=<abs>/scripts/zig-cxx.sh
set -euo pipefail

goos="${GOOS:-linux}"
goarch="${GOARCH:-amd64}"

export ZIG_GLOBAL_CACHE_DIR="${ZIG_GLOBAL_CACHE_DIR:-${TMPDIR:-/tmp}/zig-cache}"
export ZIG_LOCAL_CACHE_DIR="${ZIG_LOCAL_CACHE_DIR:-${ZIG_GLOBAL_CACHE_DIR}}"
mkdir -p "${ZIG_GLOBAL_CACHE_DIR}"

case "${goos}/${goarch}" in
  linux/amd64)   target="x86_64-linux-gnu" ;;
  linux/arm64)   target="aarch64-linux-gnu" ;;
  darwin/amd64)  target="x86_64-macos" ;;
  darwin/arm64)  target="aarch64-macos" ;;
  windows/amd64) target="x86_64-windows-gnu" ;;
  windows/arm64) target="aarch64-windows-gnu" ;;
  *)
    echo "zig-cxx.sh: unsupported GOOS/GOARCH: ${goos}/${goarch}" >&2
    exit 1
    ;;
esac

extra=()
post=()
if [[ "${goos}" == "darwin" ]]; then
  sdk="${MACOS_SDK_ROOT:-}"
  if [[ -z "${sdk}" || ! -d "${sdk}" ]]; then
    echo "zig-cxx.sh: darwin target requires MACOS_SDK_ROOT pointing at an extracted macOS SDK (run scripts/fetch-macos-sdk.sh)" >&2
    exit 1
  fi
  extra+=(
    -isysroot "${sdk}"
    -I"${sdk}/usr/include"
    -L"${sdk}/usr/lib"
    -F"${sdk}/System/Library/Frameworks"
  )
  post+=(
    -Wno-error
    -Wno-nullability-completeness
    -Wno-expansion-to-defined
    -Wno-macro-redefined
    -Wno-availability
  )
fi

exec zig c++ -target "${target}" "${extra[@]}" "$@" "${post[@]}"
