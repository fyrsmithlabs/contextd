#!/usr/bin/env bash
# fetch-onnx-libs.sh - Download the Microsoft onnxruntime shared library for
# every release target and lay each one out under:
#
#   build/onnxlibs/<goos>/<goarch>/<platform-libname>
#
# NOTE: this lives under build/ (not dist/) because GoReleaser owns dist/ and
# refuses to run if dist/ is non-empty before it cleans it.
#
# GoReleaser's archive step then bundles the matching lib NEXT TO the binary for
# each platform, so contextd's GetONNXLibraryPath() resolves it with zero setup,
# fully offline. onnxruntime is dlopen'd at runtime (never linked), so we only
# need the .so/.dylib/.dll - not headers or import libs.
#
# Idempotent: skips a target whose expected lib already exists.
#
# Version: kept in sync with internal/embeddings.DefaultONNXRuntimeVersion
# (currently "1.23.0"). Update both together when bumping onnxruntime_go.
set -euo pipefail

ONNX_VERSION="1.23.0"
BASE_URL="https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_ROOT="${ROOT_DIR}/build/onnxlibs"

# Each entry: goos goarch asset-slug archive-ext libname-in-archive bundled-libname [sidecar...]
#   - asset-slug:        the "<slug>" in onnxruntime-<slug>-<version>.<ext>
#   - libname-in-archive: the file inside <prefix>/lib/ to extract as the core lib
#   - bundled-libname:    the EXACT name the binary expects beside it
#   - sidecar(s):         extra files to also extract (windows providers DLL)
fetch_one() {
  local goos="$1" goarch="$2" slug="$3" ext="$4" archive_lib="$5" bundled_lib="$6"
  shift 6
  local sidecars=("$@")

  local out_dir="${OUT_ROOT}/${goos}/${goarch}"
  local out_lib="${out_dir}/${bundled_lib}"

  if [[ -f "${out_lib}" ]]; then
    echo "  [skip] ${goos}/${goarch}: ${bundled_lib} already present"
    return 0
  fi

  mkdir -p "${out_dir}"

  local asset="onnxruntime-${slug}-${ONNX_VERSION}.${ext}"
  local url="${BASE_URL}/${asset}"
  local prefix="onnxruntime-${slug}-${ONNX_VERSION}/lib"

  local tmp
  tmp="$(mktemp -d)"
  trap 'rm -rf "${tmp}"' RETURN

  echo "  [fetch] ${goos}/${goarch}: ${asset}"
  curl -fsSL -o "${tmp}/${asset}" "${url}"

  if [[ "${ext}" == "zip" ]]; then
    unzip -q -o "${tmp}/${asset}" -d "${tmp}/extracted"
  else
    mkdir -p "${tmp}/extracted"
    tar -xzf "${tmp}/${asset}" -C "${tmp}/extracted"
  fi

  # Copy the core lib (follow symlinks: -L grabs the real file, not a dangling link).
  cp -L "${tmp}/extracted/${prefix}/${archive_lib}" "${out_lib}"
  echo "    -> ${out_lib} ($(du -h "${out_lib}" | cut -f1))"

  # Copy any sidecar libs (windows providers).
  local sc
  for sc in "${sidecars[@]}"; do
    if [[ -f "${tmp}/extracted/${prefix}/${sc}" ]]; then
      cp -L "${tmp}/extracted/${prefix}/${sc}" "${out_dir}/${sc}"
      echo "    -> ${out_dir}/${sc} (sidecar)"
    else
      echo "    !! sidecar ${sc} not found in ${asset}" >&2
    fi
  done
}

echo "Fetching onnxruntime v${ONNX_VERSION} libs into ${OUT_ROOT}"

fetch_one linux   amd64 linux-x64     tgz libonnxruntime.so     libonnxruntime.so
fetch_one linux   arm64 linux-aarch64 tgz libonnxruntime.so     libonnxruntime.so
fetch_one darwin  amd64 osx-x86_64    tgz libonnxruntime.dylib  libonnxruntime.dylib
fetch_one darwin  arm64 osx-arm64     tgz libonnxruntime.dylib  libonnxruntime.dylib
fetch_one windows amd64 win-x64       zip onnxruntime.dll       onnxruntime.dll       onnxruntime_providers_shared.dll
fetch_one windows arm64 win-arm64     zip onnxruntime.dll       onnxruntime.dll       onnxruntime_providers_shared.dll

echo "Done. Layout:"
find "${OUT_ROOT}" -type f | sort
