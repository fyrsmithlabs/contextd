# Palette's Journal

## 2025-10-26 - Progress Bar for Large Downloads
**Learning:** Users lack feedback during large file downloads in CLI tools (like ONNX runtime setup), leading to uncertainty about whether the process is hung.
**Action:** Always include a visual progress indicator (like a progress bar) for operations taking > 1s, especially network requests. Used `github.com/schollz/progressbar/v3` which was already in dependency tree.
