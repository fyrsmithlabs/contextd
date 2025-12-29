# pkg/api/v1

Generated gRPC code from protobuf definitions.

**Last Updated**: 2025-11-25

---

## What This Package Is

**Purpose**: Generated Go code from contextd.proto

**Proto**: @../../../docs/spec/interface/api/contextd.proto

**Generation**: `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc`

**Visibility**: Public (imported by clients)

---

## Generated Files

| File | Generated From | Contains |
|------|----------------|----------|
| `contextd.pb.go` | contextd.proto | Message types |
| `contextd_grpc.pb.go` | contextd.proto | Service interfaces, stubs |

**DO NOT EDIT**: These files are generated. Edit the proto instead.

---

## Regeneration

```bash
# From project root
protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  docs/spec/interface/api/contextd.proto
```

**Triggers**: Proto file changes, version bumps

---

## Versioning

**Package**: `contextd.v1` (Go import path)
**Proto**: `package contextd.v1;`

**Breaking changes**: New major version (v2, v3, etc.)

---

## Testing

**Coverage**: N/A (generated code, tested via integration)

**Integration tests**: Use generated stubs to test gRPC services

---

## References

- Proto: @../../../docs/spec/interface/api/contextd.proto
- gRPC Go: https://grpc.io/docs/languages/go/quickstart/
- Protobuf: https://protobuf.dev/getting-started/gotutorial/
