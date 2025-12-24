// Package secrets provides secret detection and redaction using gitleaks.
//
// All contextd output passes through scrubbing to prevent secret leakage via
// gRPC interceptor and direct API. Preserves metrics (rule IDs, counts) while
// redacting sensitive content.
//
// See CLAUDE.md for scrubbing rules and integration patterns.
package secrets
