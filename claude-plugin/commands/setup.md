Set up contextd runtime dependencies (ONNX runtime for local embeddings).

## When to Use

Use this command when:
- First installing contextd
- ONNX runtime errors occur during embedding operations
- Setting up a new machine or environment

## What It Does

Runs `ctxd init` to download and install the ONNX runtime library required for local FastEmbed embeddings.

**Installation location:** `~/.config/contextd/lib/`

## Instructions

Tell the user to run:

```bash
ctxd init
```

This will:
1. Download ONNX runtime v1.23.0 for their platform (linux/darwin, amd64/arm64)
2. Extract to `~/.config/contextd/lib/`
3. Verify installation

## Flags

- `--force` or `-f`: Re-download even if already installed

```bash
ctxd init --force
```

## Environment Override

Users can set `ONNX_PATH` to use their own ONNX installation:

```bash
export ONNX_PATH=/path/to/libonnxruntime.so
```

## Auto-Download Behavior

If the user hasn't run `ctxd init`, contextd will automatically download ONNX runtime on first use of FastEmbed. However, explicit setup is recommended for:
- Airgapped environments (download beforehand)
- CI/CD pipelines
- Docker builds

## Troubleshooting

If setup fails:
1. Check network connectivity (downloads from GitHub releases)
2. Verify write permissions to `~/.config/contextd/lib/`
3. Check platform support: linux/darwin on amd64/arm64

For manual installation, download from:
https://github.com/microsoft/onnxruntime/releases/tag/v1.23.0
