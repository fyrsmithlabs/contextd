# Config Package Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the foundational config package with Koanf loading, shared types, and ServerConfig using interface-first TDD.

**Architecture:** Distributed config where `internal/config/` provides Load(), shared types (Secret, Duration), and root Config struct. Feature-specific configs live in their respective packages and compose into root Config. All behavior exposed via interfaces for testability.

**Tech Stack:** Go 1.23+, Koanf v2, go-playground/validator v10, gomock, pflag

**Design Principles:**
- Interface-first: Define behavior contracts before implementation
- TDD: Failing test → minimal implementation → refactor
- Mockable: All external dependencies behind interfaces
- CHANGELOG: Update after each feature commit

---

## Prerequisites

Before starting:
- Go 1.23+ installed
- Git repository initialized

---

## Task 1: Initialize Go Module and CHANGELOG

**Files:**
- Create: `go.mod`
- Create: `go.sum`
- Create: `CHANGELOG.md`

**Step 1: Initialize the Go module**

```bash
cd /home/dahendel/projects/contextd-reasoning
go mod init github.com/contextd/contextd
```

Expected: `go.mod` created

**Step 2: Add core dependencies**

```bash
go get github.com/knadh/koanf/v2@latest
go get github.com/knadh/koanf/parsers/yaml@latest
go get github.com/knadh/koanf/providers/file@latest
go get github.com/knadh/koanf/providers/env@latest
go get github.com/knadh/koanf/providers/structs@latest
go get github.com/go-playground/validator/v10@latest
go get github.com/stretchr/testify@latest
go get go.uber.org/mock/mockgen@latest
```

**Step 3: Install mockgen CLI**

```bash
go install go.uber.org/mock/mockgen@latest
```

**Step 4: Create CHANGELOG**

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial Go module with config dependencies
```

**Step 5: Verify and commit**

```bash
go mod tidy && go mod verify
git add go.mod go.sum CHANGELOG.md
git commit -m "chore: initialize go module with config dependencies"
```

---

## Task 2: Define Config Interfaces

**Files:**
- Create: `internal/config/interfaces.go`
- Create: `internal/config/interfaces_test.go`

**Step 1: Write interface contract tests**

```go
// internal/config/interfaces_test.go
package config_test

import (
    "testing"

    "github.com/contextd/contextd/internal/config"
)

// Compile-time interface compliance checks
var (
    _ config.Loader    = (*config.KoanfLoader)(nil)
    _ config.Validator = (*config.StructValidator)(nil)
)

func TestLoader_Interface(t *testing.T) {
    // Interface defines Load behavior
    // Implementations will be tested separately
}

func TestValidator_Interface(t *testing.T) {
    // Interface defines Validate behavior
    // Implementations will be tested separately
}
```

**Step 2: Run test to verify it fails**

```bash
mkdir -p internal/config
go test ./internal/config/... -v
```

Expected: FAIL - types not defined

**Step 3: Define interfaces**

```go
// internal/config/interfaces.go
package config

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

// Loader loads configuration from various sources.
type Loader interface {
    // Load loads configuration and returns the populated Config.
    // If configPath is empty, searches standard locations.
    // If configPath is provided but doesn't exist, returns error.
    Load(configPath string) (*Config, error)
}

// Validator validates configuration structs.
type Validator interface {
    // Validate validates the config and returns validation errors.
    Validate(cfg *Config) error
}

// Validatable allows types to provide custom validation.
type Validatable interface {
    // Validate performs custom validation beyond struct tags.
    Validate() error
}
```

**Step 4: Create placeholder types for compile check**

```go
// internal/config/loader.go
package config

// KoanfLoader implements Loader using Koanf.
type KoanfLoader struct{}

// Load implements Loader.
func (l *KoanfLoader) Load(configPath string) (*Config, error) {
    panic("not implemented")
}

// internal/config/validator.go
package config

// StructValidator implements Validator using go-playground/validator.
type StructValidator struct{}

// Validate implements Validator.
func (v *StructValidator) Validate(cfg *Config) error {
    panic("not implemented")
}
```

**Step 5: Create minimal Config for compile**

```go
// internal/config/config.go
package config

// Config is the root configuration struct.
type Config struct{}
```

**Step 6: Run test to verify interfaces compile**

```bash
go test ./internal/config/... -v
```

Expected: PASS (compile check passes)

**Step 7: Generate mocks**

```bash
cd internal/config
go generate ./...
```

Expected: `mocks/mocks.go` created

**Step 8: Update CHANGELOG and commit**

Add to CHANGELOG.md under `### Added`:
```markdown
- Config interfaces: Loader, Validator, Validatable
- Mock generation with gomock
```

```bash
git add internal/config/ CHANGELOG.md
git commit -m "feat(config): define Loader and Validator interfaces"
```

---

## Task 3: Create Duration Type

**Files:**
- Create: `internal/config/types.go`
- Create: `internal/config/types_test.go`

**Step 1: Write failing test for Duration**

```go
// internal/config/types_test.go
package config

import (
    "encoding/json"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestDuration_UnmarshalText(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected time.Duration
        wantErr  bool
    }{
        {"seconds", "30s", 30 * time.Second, false},
        {"minutes", "5m", 5 * time.Minute, false},
        {"hours", "2h", 2 * time.Hour, false},
        {"mixed", "1h30m", 90 * time.Minute, false},
        {"milliseconds", "500ms", 500 * time.Millisecond, false},
        {"invalid", "invalid", 0, true},
        {"empty", "", 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var d Duration
            err := d.UnmarshalText([]byte(tt.input))

            if tt.wantErr {
                assert.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.expected, d.Duration())
        })
    }
}

func TestDuration_MarshalText(t *testing.T) {
    d := Duration(30 * time.Second)
    text, err := d.MarshalText()
    require.NoError(t, err)
    assert.Equal(t, "30s", string(text))
}

func TestDuration_MarshalJSON(t *testing.T) {
    d := Duration(5 * time.Minute)
    data, err := json.Marshal(d)
    require.NoError(t, err)
    assert.Equal(t, `"5m0s"`, string(data))
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/config/... -v -run TestDuration
```

Expected: FAIL - Duration not defined

**Step 3: Write Duration implementation**

```go
// internal/config/types.go
package config

import (
    "encoding/json"
    "time"
)

// Duration wraps time.Duration for text unmarshaling (YAML, env vars).
type Duration time.Duration

// UnmarshalText implements encoding.TextUnmarshaler.
func (d *Duration) UnmarshalText(text []byte) error {
    parsed, err := time.ParseDuration(string(text))
    if err != nil {
        return err
    }
    *d = Duration(parsed)
    return nil
}

// MarshalText implements encoding.TextMarshaler.
func (d Duration) MarshalText() ([]byte, error) {
    return []byte(d.Duration().String()), nil
}

// MarshalJSON implements json.Marshaler.
func (d Duration) MarshalJSON() ([]byte, error) {
    return json.Marshal(d.Duration().String())
}

// Duration returns the underlying time.Duration.
func (d Duration) Duration() time.Duration {
    return time.Duration(d)
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/config/... -v -run TestDuration
```

Expected: PASS

**Step 5: Update CHANGELOG and commit**

Add to CHANGELOG.md under `### Added`:
```markdown
- Duration type with text/JSON marshaling
```

```bash
git add internal/config/types.go internal/config/types_test.go CHANGELOG.md
git commit -m "feat(config): add Duration type with text marshaling"
```

---

## Task 4: Create Secret Type

**Files:**
- Modify: `internal/config/types.go`
- Modify: `internal/config/types_test.go`

**Step 1: Write failing test for Secret**

Add to `internal/config/types_test.go`:

```go
func TestSecret_String(t *testing.T) {
    tests := []struct {
        name     string
        secret   Secret
        expected string
    }{
        {"non-empty", Secret("my-api-key"), "[REDACTED]"},
        {"empty", Secret(""), ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.expected, tt.secret.String())
        })
    }
}

func TestSecret_Value(t *testing.T) {
    s := Secret("my-secret-value")
    assert.Equal(t, "my-secret-value", s.Value())
}

func TestSecret_MarshalJSON(t *testing.T) {
    s := Secret("sensitive-data")
    data, err := json.Marshal(s)
    require.NoError(t, err)
    assert.Equal(t, `"[REDACTED]"`, string(data))
}

func TestSecret_MarshalText(t *testing.T) {
    s := Secret("sensitive-data")
    text, err := s.MarshalText()
    require.NoError(t, err)
    assert.Equal(t, "[REDACTED]", string(text))
}

func TestSecret_GoString(t *testing.T) {
    s := Secret("api-key")
    assert.Equal(t, "Secret([REDACTED])", s.GoString())
}

func TestSecret_IsSet(t *testing.T) {
    assert.True(t, Secret("value").IsSet())
    assert.False(t, Secret("").IsSet())
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/config/... -v -run TestSecret
```

Expected: FAIL - Secret not defined

**Step 3: Write Secret implementation**

Add to `internal/config/types.go`:

```go
// Secret wraps strings that should be redacted in logs and serialization.
// Use Value() to access the actual secret value.
type Secret string

// String implements fmt.Stringer. Always returns redacted value.
func (s Secret) String() string {
    if s == "" {
        return ""
    }
    return "[REDACTED]"
}

// GoString implements fmt.GoStringer for %#v formatting.
func (s Secret) GoString() string {
    return "Secret([REDACTED])"
}

// Value returns the actual secret value. Use sparingly.
func (s Secret) Value() string {
    return string(s)
}

// IsSet returns true if the secret has a non-empty value.
func (s Secret) IsSet() bool {
    return s != ""
}

// MarshalJSON implements json.Marshaler. Always returns redacted value.
func (s Secret) MarshalJSON() ([]byte, error) {
    if s == "" {
        return json.Marshal("")
    }
    return json.Marshal("[REDACTED]")
}

// MarshalText implements encoding.TextMarshaler. Always returns redacted value.
func (s Secret) MarshalText() ([]byte, error) {
    if s == "" {
        return []byte(""), nil
    }
    return []byte("[REDACTED]"), nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/config/... -v -run TestSecret
```

Expected: PASS

**Step 5: Update CHANGELOG and commit**

Add to CHANGELOG.md under `### Added`:
```markdown
- Secret type with auto-redaction in logs/JSON
```

```bash
git add internal/config/types.go internal/config/types_test.go CHANGELOG.md
git commit -m "feat(config): add Secret type with auto-redaction"
```

---

## Task 5: Create ServerConfig

**Files:**
- Create: `internal/config/server.go`
- Create: `internal/config/server_test.go`

**Step 1: Write failing test for ServerConfig**

```go
// internal/config/server_test.go
package config

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestServerConfig_Defaults(t *testing.T) {
    cfg := DefaultServerConfig()

    // gRPC defaults
    assert.Equal(t, 50051, cfg.GRPC.Port)
    assert.Equal(t, 16*1024*1024, cfg.GRPC.MaxRecvMsgSize)
    assert.Equal(t, 16*1024*1024, cfg.GRPC.MaxSendMsgSize)
    assert.Equal(t, 30*time.Second, cfg.GRPC.Keepalive.Time.Duration())

    // HTTP defaults
    assert.Equal(t, 8080, cfg.HTTP.Port)
    assert.Equal(t, 30*time.Second, cfg.HTTP.ReadTimeout.Duration())

    // Shutdown
    assert.Equal(t, 30*time.Second, cfg.ShutdownTimeout.Duration())
}

func TestGRPCConfig_Address(t *testing.T) {
    tests := []struct {
        name     string
        host     string
        port     int
        expected string
    }{
        {"no host", "", 50051, ":50051"},
        {"with host", "0.0.0.0", 50051, "0.0.0.0:50051"},
        {"localhost", "localhost", 9000, "localhost:9000"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg := GRPCConfig{Host: tt.host, Port: tt.port}
            assert.Equal(t, tt.expected, cfg.Address())
        })
    }
}

func TestHTTPConfig_Address(t *testing.T) {
    tests := []struct {
        name     string
        host     string
        port     int
        expected string
    }{
        {"no host", "", 8080, ":8080"},
        {"with host", "localhost", 8080, "localhost:8080"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg := HTTPConfig{Host: tt.host, Port: tt.port}
            assert.Equal(t, tt.expected, cfg.Address())
        })
    }
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/config/... -v -run TestServerConfig -run TestGRPCConfig -run TestHTTPConfig
```

Expected: FAIL - types not defined

**Step 3: Write ServerConfig implementation**

```go
// internal/config/server.go
package config

import (
    "fmt"
    "time"
)

// ServerConfig holds gRPC and HTTP server configuration.
type ServerConfig struct {
    GRPC            GRPCConfig `koanf:"grpc"`
    HTTP            HTTPConfig `koanf:"http"`
    ShutdownTimeout Duration   `koanf:"shutdown_timeout" validate:"required,min=1s"`
}

// GRPCConfig holds gRPC server settings.
type GRPCConfig struct {
    Host           string          `koanf:"host"`
    Port           int             `koanf:"port" validate:"min=0,max=65535"`
    MaxRecvMsgSize int             `koanf:"max_recv_msg_size" validate:"min=0"`
    MaxSendMsgSize int             `koanf:"max_send_msg_size" validate:"min=0"`
    Keepalive      KeepaliveConfig `koanf:"keepalive"`
}

// Address returns the gRPC listen address.
func (c GRPCConfig) Address() string {
    if c.Host == "" {
        return fmt.Sprintf(":%d", c.Port)
    }
    return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// HTTPConfig holds HTTP server settings.
type HTTPConfig struct {
    Host         string   `koanf:"host"`
    Port         int      `koanf:"port" validate:"min=0,max=65535"`
    ReadTimeout  Duration `koanf:"read_timeout"`
    WriteTimeout Duration `koanf:"write_timeout"`
    IdleTimeout  Duration `koanf:"idle_timeout"`
}

// Address returns the HTTP listen address.
func (c HTTPConfig) Address() string {
    if c.Host == "" {
        return fmt.Sprintf(":%d", c.Port)
    }
    return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// KeepaliveConfig holds gRPC keepalive settings.
type KeepaliveConfig struct {
    Time    Duration `koanf:"time"`
    Timeout Duration `koanf:"timeout"`
    MinTime Duration `koanf:"min_time"`
}

// DefaultServerConfig returns server config with sensible defaults.
func DefaultServerConfig() ServerConfig {
    return ServerConfig{
        GRPC: GRPCConfig{
            Port:           50051,
            MaxRecvMsgSize: 16 * 1024 * 1024,
            MaxSendMsgSize: 16 * 1024 * 1024,
            Keepalive: KeepaliveConfig{
                Time:    Duration(30 * time.Second),
                Timeout: Duration(10 * time.Second),
                MinTime: Duration(10 * time.Second),
            },
        },
        HTTP: HTTPConfig{
            Port:         8080,
            ReadTimeout:  Duration(30 * time.Second),
            WriteTimeout: Duration(30 * time.Second),
            IdleTimeout:  Duration(120 * time.Second),
        },
        ShutdownTimeout: Duration(30 * time.Second),
    }
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/config/... -v -run "TestServerConfig|TestGRPCConfig|TestHTTPConfig"
```

Expected: PASS

**Step 5: Update CHANGELOG and commit**

Add to CHANGELOG.md under `### Added`:
```markdown
- ServerConfig with gRPC and HTTP settings
```

```bash
git add internal/config/server.go internal/config/server_test.go CHANGELOG.md
git commit -m "feat(config): add ServerConfig with gRPC and HTTP settings"
```

---

## Task 6: Implement StructValidator

**Files:**
- Modify: `internal/config/validator.go`
- Create: `internal/config/validator_test.go`

**Step 1: Write failing test for validation**

```go
// internal/config/validator_test.go
package config

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestStructValidator_Validate_ValidConfig(t *testing.T) {
    v := NewStructValidator()
    cfg := &Config{Server: DefaultServerConfig()}

    err := v.Validate(cfg)
    assert.NoError(t, err)
}

func TestStructValidator_Validate_InvalidPort(t *testing.T) {
    v := NewStructValidator()
    cfg := &Config{Server: DefaultServerConfig()}
    cfg.Server.GRPC.Port = 70000 // Invalid

    err := v.Validate(cfg)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "port")
}

func TestStructValidator_Validate_InvalidShutdownTimeout(t *testing.T) {
    v := NewStructValidator()
    cfg := &Config{Server: DefaultServerConfig()}
    cfg.Server.ShutdownTimeout = 0

    err := v.Validate(cfg)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "shutdown_timeout")
}

func TestStructValidator_Validate_DurationMin(t *testing.T) {
    v := NewStructValidator()
    cfg := &Config{Server: DefaultServerConfig()}
    cfg.Server.ShutdownTimeout = Duration(500) // 500ns < 1s

    err := v.Validate(cfg)
    require.Error(t, err)
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/config/... -v -run TestStructValidator
```

Expected: FAIL - NewStructValidator not defined

**Step 3: Write StructValidator implementation**

```go
// internal/config/validator.go
package config

import (
    "fmt"
    "reflect"
    "strings"
    "time"

    "github.com/go-playground/validator/v10"
)

// StructValidator implements Validator using go-playground/validator.
type StructValidator struct {
    validate *validator.Validate
}

// NewStructValidator creates a new StructValidator with custom validations.
func NewStructValidator() *StructValidator {
    v := validator.New()

    // Register custom Duration min validation
    v.RegisterCustomTypeFunc(func(field reflect.Value) interface{} {
        if d, ok := field.Interface().(Duration); ok {
            return d.Duration()
        }
        return nil
    }, Duration{})

    // Use koanf tag names in error messages
    v.RegisterTagNameFunc(func(fld reflect.StructField) string {
        name := strings.SplitN(fld.Tag.Get("koanf"), ",", 2)[0]
        if name == "-" || name == "" {
            return fld.Name
        }
        return name
    })

    return &StructValidator{validate: v}
}

// Validate implements Validator.
func (v *StructValidator) Validate(cfg *Config) error {
    if err := v.validate.Struct(cfg); err != nil {
        if validationErrors, ok := err.(validator.ValidationErrors); ok {
            return formatValidationErrors(validationErrors)
        }
        return err
    }

    // Run custom validation if implemented
    if validatable, ok := interface{}(cfg).(Validatable); ok {
        if err := validatable.Validate(); err != nil {
            return err
        }
    }

    return nil
}

// formatValidationErrors formats errors for user display.
func formatValidationErrors(errs validator.ValidationErrors) error {
    var msgs []string
    for _, err := range errs {
        field := err.Namespace()
        // Remove Config. prefix
        field = strings.TrimPrefix(field, "Config.")
        field = strings.ToLower(field)

        msg := fmt.Sprintf("%s: failed %s validation", field, err.Tag())
        if err.Param() != "" {
            msg += fmt.Sprintf(" (required: %s)", err.Param())
        }
        msgs = append(msgs, msg)
    }
    return fmt.Errorf("config validation failed:\n  - %s", strings.Join(msgs, "\n  - "))
}
```

**Step 4: Update Config struct**

```go
// internal/config/config.go
package config

// Config is the root configuration struct.
type Config struct {
    Server ServerConfig `koanf:"server"`
}
```

**Step 5: Run test to verify it passes**

```bash
go test ./internal/config/... -v -run TestStructValidator
```

Expected: PASS

**Step 6: Regenerate mocks**

```bash
cd internal/config && go generate ./...
```

**Step 7: Update CHANGELOG and commit**

Add to CHANGELOG.md under `### Added`:
```markdown
- StructValidator with go-playground/validator integration
- Custom Duration validation
```

```bash
git add internal/config/validator.go internal/config/validator_test.go internal/config/config.go internal/config/mocks/ CHANGELOG.md
git commit -m "feat(config): implement StructValidator with custom Duration validation"
```

---

## Task 7: Implement KoanfLoader

**Files:**
- Modify: `internal/config/loader.go`
- Create: `internal/config/loader_test.go`

**Step 1: Write failing test for KoanfLoader**

```go
// internal/config/loader_test.go
package config

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestKoanfLoader_Load_Defaults(t *testing.T) {
    loader := NewKoanfLoader(NewStructValidator())

    cfg, err := loader.Load("")
    require.NoError(t, err)

    assert.Equal(t, 50051, cfg.Server.GRPC.Port)
    assert.Equal(t, 8080, cfg.Server.HTTP.Port)
}

func TestKoanfLoader_Load_FromFile(t *testing.T) {
    dir := t.TempDir()
    configPath := filepath.Join(dir, "config.yaml")
    content := `
server:
  grpc:
    port: 9000
  http:
    port: 9080
`
    err := os.WriteFile(configPath, []byte(content), 0644)
    require.NoError(t, err)

    loader := NewKoanfLoader(NewStructValidator())
    cfg, err := loader.Load(configPath)
    require.NoError(t, err)

    assert.Equal(t, 9000, cfg.Server.GRPC.Port)
    assert.Equal(t, 9080, cfg.Server.HTTP.Port)
}

func TestKoanfLoader_Load_EnvOverride(t *testing.T) {
    t.Setenv("CONTEXTD_SERVER_GRPC_PORT", "7000")

    loader := NewKoanfLoader(NewStructValidator())
    cfg, err := loader.Load("")
    require.NoError(t, err)

    assert.Equal(t, 7000, cfg.Server.GRPC.Port)
}

func TestKoanfLoader_Load_FileNotFound(t *testing.T) {
    loader := NewKoanfLoader(NewStructValidator())

    _, err := loader.Load("/nonexistent/config.yaml")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not found")
}

func TestKoanfLoader_Load_InvalidYAML(t *testing.T) {
    dir := t.TempDir()
    configPath := filepath.Join(dir, "config.yaml")
    err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644)
    require.NoError(t, err)

    loader := NewKoanfLoader(NewStructValidator())
    _, err = loader.Load(configPath)
    assert.Error(t, err)
}

func TestKoanfLoader_Load_ValidationFails(t *testing.T) {
    dir := t.TempDir()
    configPath := filepath.Join(dir, "config.yaml")
    content := `
server:
  grpc:
    port: 70000  # Invalid port
`
    err := os.WriteFile(configPath, []byte(content), 0644)
    require.NoError(t, err)

    loader := NewKoanfLoader(NewStructValidator())
    _, err = loader.Load(configPath)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "validation")
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/config/... -v -run TestKoanfLoader
```

Expected: FAIL - NewKoanfLoader not defined

**Step 3: Write KoanfLoader implementation**

```go
// internal/config/loader.go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/knadh/koanf/parsers/yaml"
    "github.com/knadh/koanf/providers/env"
    "github.com/knadh/koanf/providers/file"
    "github.com/knadh/koanf/providers/structs"
    "github.com/knadh/koanf/v2"
)

// KoanfLoader implements Loader using Koanf.
type KoanfLoader struct {
    validator Validator
}

// NewKoanfLoader creates a new KoanfLoader with the given validator.
func NewKoanfLoader(v Validator) *KoanfLoader {
    return &KoanfLoader{validator: v}
}

// Load implements Loader.
func (l *KoanfLoader) Load(configPath string) (*Config, error) {
    k := koanf.New(".")

    // 1. Load defaults
    defaults := DefaultConfig()
    if err := k.Load(structs.Provider(defaults, "koanf"), nil); err != nil {
        return nil, fmt.Errorf("load defaults: %w", err)
    }

    // 2. Load config file
    path, err := l.resolveConfigPath(configPath)
    if err != nil {
        return nil, err
    }
    if path != "" {
        if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
            return nil, fmt.Errorf("load config file %s: %w", path, err)
        }
    }

    // 3. Load environment variables (CONTEXTD_ prefix)
    envProvider := env.Provider("CONTEXTD_", ".", func(s string) string {
        // CONTEXTD_SERVER_GRPC_PORT -> server.grpc.port
        return strings.Replace(
            strings.ToLower(strings.TrimPrefix(s, "CONTEXTD_")),
            "_", ".", -1,
        )
    })
    if err := k.Load(envProvider, nil); err != nil {
        return nil, fmt.Errorf("load env vars: %w", err)
    }

    // 4. Unmarshal to struct
    var cfg Config
    if err := k.Unmarshal("", &cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }

    // 5. Validate
    if l.validator != nil {
        if err := l.validator.Validate(&cfg); err != nil {
            return nil, fmt.Errorf("validation failed: %w", err)
        }
    }

    return &cfg, nil
}

// resolveConfigPath finds the config file.
func (l *KoanfLoader) resolveConfigPath(explicit string) (string, error) {
    if explicit != "" {
        if _, err := os.Stat(explicit); err != nil {
            return "", fmt.Errorf("config file not found: %s", explicit)
        }
        return explicit, nil
    }

    // Search standard locations
    paths := []string{
        "./config.yaml",
        "./contextd.yaml",
    }

    if home, err := os.UserHomeDir(); err == nil {
        paths = append(paths, filepath.Join(home, ".config", "contextd", "config.yaml"))
    }

    paths = append(paths, "/etc/contextd/config.yaml")

    for _, p := range paths {
        if _, err := os.Stat(p); err == nil {
            return p, nil
        }
    }

    return "", nil // No config file, use defaults
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() *Config {
    return &Config{
        Server: DefaultServerConfig(),
    }
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/config/... -v -run TestKoanfLoader
```

Expected: PASS

**Step 5: Regenerate mocks**

```bash
cd internal/config && go generate ./...
```

**Step 6: Update CHANGELOG and commit**

Add to CHANGELOG.md under `### Added`:
```markdown
- KoanfLoader with YAML file and environment variable support
- Config path resolution (./config.yaml, ~/.config/contextd/, /etc/contextd/)
```

```bash
git add internal/config/loader.go internal/config/loader_test.go internal/config/mocks/ CHANGELOG.md
git commit -m "feat(config): implement KoanfLoader with file and env loading"
```

---

## Task 8: Add Convenience Load Function

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write test for convenience function**

```go
// internal/config/config_test.go
package config

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoad_Convenience(t *testing.T) {
    cfg, err := Load("")
    require.NoError(t, err)
    assert.Equal(t, 50051, cfg.Server.GRPC.Port)
}

func TestLoad_WithFile(t *testing.T) {
    dir := t.TempDir()
    configPath := filepath.Join(dir, "config.yaml")
    content := `
server:
  grpc:
    port: 5555
`
    err := os.WriteFile(configPath, []byte(content), 0644)
    require.NoError(t, err)

    cfg, err := Load(configPath)
    require.NoError(t, err)
    assert.Equal(t, 5555, cfg.Server.GRPC.Port)
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/config/... -v -run "TestLoad_Convenience|TestLoad_WithFile"
```

Expected: FAIL - Load not defined

**Step 3: Add Load convenience function**

Add to `internal/config/config.go`:

```go
// Load is a convenience function that creates a KoanfLoader with
// StructValidator and loads configuration.
func Load(configPath string) (*Config, error) {
    loader := NewKoanfLoader(NewStructValidator())
    return loader.Load(configPath)
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/config/... -v -run "TestLoad_Convenience|TestLoad_WithFile"
```

Expected: PASS

**Step 5: Update CHANGELOG and commit**

Add to CHANGELOG.md under `### Added`:
```markdown
- Load() convenience function
```

```bash
git add internal/config/config.go internal/config/config_test.go CHANGELOG.md
git commit -m "feat(config): add Load convenience function"
```

---

## Task 9: Add Test Helpers

**Files:**
- Create: `internal/config/testing.go`
- Create: `internal/config/testing_test.go`

**Step 1: Write test for test helpers**

```go
// internal/config/testing_test.go
package config

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestTestConfig(t *testing.T) {
    cfg := TestConfig()

    // Should be valid
    v := NewStructValidator()
    err := v.Validate(cfg)
    require.NoError(t, err)

    // Should have test-friendly values (port 0 = random)
    assert.Equal(t, 0, cfg.Server.GRPC.Port)
    assert.Equal(t, 0, cfg.Server.HTTP.Port)
}

func TestTestConfigWith(t *testing.T) {
    cfg := TestConfigWith(func(c *Config) {
        c.Server.GRPC.Port = 12345
    })

    assert.Equal(t, 12345, cfg.Server.GRPC.Port)
}

func TestMustTestConfig_Valid(t *testing.T) {
    cfg := MustTestConfig(func(c *Config) {
        c.Server.GRPC.Port = 9999
    })
    assert.Equal(t, 9999, cfg.Server.GRPC.Port)
}

func TestMustTestConfig_Panics(t *testing.T) {
    defer func() {
        r := recover()
        assert.NotNil(t, r)
    }()

    MustTestConfig(func(c *Config) {
        c.Server.GRPC.Port = 70000 // Invalid
    })
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/config/... -v -run TestTestConfig -run TestMustTestConfig
```

Expected: FAIL - TestConfig not defined

**Step 3: Write test helpers**

```go
// internal/config/testing.go
package config

// TestConfig returns a valid configuration for testing.
// Uses port 0 so tests can pick random available ports.
func TestConfig() *Config {
    cfg := DefaultConfig()

    // Use port 0 for random port assignment in tests
    cfg.Server.GRPC.Port = 0
    cfg.Server.HTTP.Port = 0

    return cfg
}

// TestConfigWith returns a test config with modifications applied.
func TestConfigWith(modifiers ...func(*Config)) *Config {
    cfg := TestConfig()
    for _, mod := range modifiers {
        mod(cfg)
    }
    return cfg
}

// MustTestConfig returns a test config, panicking if modifications make it invalid.
func MustTestConfig(modifiers ...func(*Config)) *Config {
    cfg := TestConfigWith(modifiers...)
    v := NewStructValidator()
    if err := v.Validate(cfg); err != nil {
        panic("invalid test config: " + err.Error())
    }
    return cfg
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/config/... -v -run "TestTestConfig|TestMustTestConfig"
```

Expected: PASS

**Step 5: Update CHANGELOG and commit**

Add to CHANGELOG.md under `### Added`:
```markdown
- Test helpers: TestConfig(), TestConfigWith(), MustTestConfig()
```

```bash
git add internal/config/testing.go internal/config/testing_test.go CHANGELOG.md
git commit -m "feat(config): add test helpers for config creation"
```

---

## Task 10: Add Example Config and Documentation

**Files:**
- Create: `config.example.yaml`
- Update: `CHANGELOG.md` with version

**Step 1: Create example config**

```yaml
# config.example.yaml
# Contextd Configuration
# Copy to config.yaml and modify as needed.
# Environment variables override file values (prefix: CONTEXTD_)

server:
  grpc:
    host: ""              # Empty = all interfaces
    port: 50051
    max_recv_msg_size: 16777216  # 16MB
    max_send_msg_size: 16777216
    keepalive:
      time: 30s
      timeout: 10s
      min_time: 10s
  http:
    host: ""
    port: 8080
    read_timeout: 30s
    write_timeout: 30s
    idle_timeout: 120s
  shutdown_timeout: 30s

# Feature configs added by their packages:
# - logging (internal/logging)
# - telemetry (internal/telemetry)
# - qdrant (internal/qdrant)
# - scrubber (internal/scrubber)
```

**Step 2: Run all tests with coverage**

```bash
go test ./internal/config/... -v -cover -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Expected: All pass, >80% coverage

**Step 3: Verify no lint issues**

```bash
go vet ./internal/config/...
```

Expected: No issues

**Step 4: Update CHANGELOG version**

Change `## [Unreleased]` to `## [0.1.0] - 2025-11-24` and add summary.

**Step 5: Final commit**

```bash
go mod tidy
git add config.example.yaml CHANGELOG.md go.mod go.sum
git commit -m "docs: add example config and finalize v0.1.0"
```

---

## Summary

After completing all tasks:

```
internal/config/
├── interfaces.go       # Loader, Validator interfaces
├── config.go           # Config struct, Load(), DefaultConfig()
├── config_test.go      # Load tests
├── loader.go           # KoanfLoader implementation
├── loader_test.go      # Loader tests
├── validator.go        # StructValidator implementation
├── validator_test.go   # Validator tests
├── server.go           # ServerConfig
├── server_test.go      # Server tests
├── types.go            # Duration, Secret
├── types_test.go       # Type tests
├── testing.go          # Test helpers
├── testing_test.go     # Helper tests
└── mocks/
    └── mocks.go        # Generated mocks

config.example.yaml     # Example configuration
CHANGELOG.md            # Version history
```

**Interfaces for mocking:**
- `Loader` - mock config loading in tests
- `Validator` - mock validation in tests

**Ready for next phase:** Logging package can now import `config.Duration`, `config.Secret`, and define its own config struct.

---

Plan saved. Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?
