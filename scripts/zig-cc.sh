#!/usr/bin/env bash
# zig-cc.sh - C compiler wrapper that dispatches to `zig cc` for the target
# GOOS/GOARCH currently being built by `go build` / GoReleaser.
#
# GoReleaser sets GOOS and GOARCH in the environment for each build target, so a
# single CC wrapper can cross-compile every platform from one Linux runner.
# onnxruntime is loaded via dlopen at runtime (never linked at build time), so
# CGO here only needs a working C toolchain for the small cgo shims.
#
# Requirements / notes:
#   * zig 0.14.0+ is required. (0.13's Mach-O linker cannot link Go-generated
#     darwin objects: "symbol _runtime.covctrs not attached to any subsection".)
#   * darwin targets additionally need the macOS SDK headers/libs (e.g. for
#     prometheus/client_golang's <mach/mach_vm.h>). Point MACOS_SDK_ROOT at an
#     extracted SDK (scripts/fetch-macos-sdk.sh installs one). Use a pre-visionOS
#     SDK (<= 12.x); newer SDKs reference the "visionos" availability platform
#     that zig's bundled clang rejects.
#
# Usage (set as CC):  CC=<abs>/scripts/zig-cc.sh
set -euo pipefail

goos="${GOOS:-linux}"
goarch="${GOARCH:-amd64}"

# zig needs a writable cache dir. The default (~/.cache/zig) can hit AccessDenied
# on read-only Go module dirs (zig writes a stray <base>.o into cwd) and on
# networked / Windows-mounted filesystems, so pin it to a stable local path
# unless the caller already set one.
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
    echo "zig-cc.sh: unsupported GOOS/GOARCH: ${goos}/${goarch}" >&2
    exit 1
    ;;
esac

extra=()
post=()
if [[ "${goos}" == "darwin" ]]; then
  sdk="${MACOS_SDK_ROOT:-}"
  if [[ -z "${sdk}" || ! -d "${sdk}" ]]; then
    echo "zig-cc.sh: darwin target requires MACOS_SDK_ROOT pointing at an extracted macOS SDK (run scripts/fetch-macos-sdk.sh)" >&2
    exit 1
  fi
  extra+=(
    -isysroot "${sdk}"
    -I"${sdk}/usr/include"
    -L"${sdk}/usr/lib"
    -F"${sdk}/System/Library/Frameworks"
  )
  # Suppress SDK-header warnings that zig-clang would otherwise promote via
  # -Werror. Placed AFTER the caller's args so they win.
  post+=(
    -Wno-error
    -Wno-nullability-completeness
    -Wno-expansion-to-defined
    -Wno-macro-redefined
    -Wno-availability
  )
fi

exec zig cc -target "${target}" "${extra[@]}" "$@" "${post[@]}"
