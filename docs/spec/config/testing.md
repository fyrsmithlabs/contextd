# Configuration Testing

**Parent**: @./SPEC.md

Test helpers and testing patterns for configuration.

---

## Test Helpers

```go
// internal/config/testing.go

// TestConfig returns a config suitable for tests
func TestConfig() *Config {
    cfg := DefaultConfig()
    cfg.Telemetry.Enabled = false
    cfg.Logging.Level = "debug"
    cfg.Logging.Format = "console"
    cfg.Logging.Sampling.Enabled = false
    return cfg
}

// WithOverrides applies test-specific overrides
func (c *Config) WithOverrides(overrides map[string]interface{}) *Config {
    // Apply overrides to config
    return c
}

// MustLoad loads config or panics (for tests)
func MustLoad(path string) *Config {
    cfg, err := Load(path, nil)
    if err != nil {
        panic(err)
    }
    return cfg
}
```

---

## Test Examples

### Default Loading

```go
func TestLoad_DefaultsApplied(t *testing.T) {
    // No config file, no env vars
    cfg, err := config.Load("", nil)
    require.NoError(t, err)

    assert.Equal(t, 50051, cfg.Server.GRPC.Port)
    assert.Equal(t, "localhost", cfg.Qdrant.Host)
    assert.Equal(t, "info", cfg.Logging.Level)
}
```

### Environment Override

```go
func TestLoad_EnvOverridesFile(t *testing.T) {
    // Set env var
    t.Setenv("CONTEXTD_SERVER_GRPC_PORT", "9090")

    cfg, err := config.Load("testdata/config.yaml", nil)
    require.NoError(t, err)

    // Env should override file
    assert.Equal(t, 9090, cfg.Server.GRPC.Port)
}
```

### Validation Failure

```go
func TestLoad_ValidationFails(t *testing.T) {
    t.Setenv("CONTEXTD_SERVER_GRPC_PORT", "-1")

    _, err := config.Load("", nil)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "validation failed")
}
```

### Secret Redaction

```go
func TestSecret_RedactedInLogs(t *testing.T) {
    secret := config.Secret("super-secret-value")

    assert.Equal(t, "[REDACTED]", secret.String())
    assert.Equal(t, "super-secret-value", secret.Value())

    json, _ := json.Marshal(secret)
    assert.Equal(t, `"[REDACTED]"`, string(json))
}
```

---

## Test Fixtures

### testdata/config.yaml

```yaml
# Minimal test config
server:
  grpc:
    port: 50051
qdrant:
  host: localhost
  port: 6334
```

### testdata/invalid.yaml

```yaml
# Invalid for validation testing
server:
  grpc:
    port: -1  # Invalid
```

### testdata/complete.yaml

```yaml
# Complete config for full integration tests
server:
  grpc:
    port: 50051
    max_recv_msg_size: 16777216
  http:
    port: 8080
  shutdown_timeout: 30s
qdrant:
  host: localhost
  port: 6334
telemetry:
  enabled: true
  service_name: contextd-test
logging:
  level: debug
  format: console
```

---

## Table-Driven Tests

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        modify  func(*Config)
        wantErr string
    }{
        {
            name:    "valid defaults",
            modify:  nil,
            wantErr: "",
        },
        {
            name: "invalid port",
            modify: func(c *Config) {
                c.Server.GRPC.Port = 0
            },
            wantErr: "Port: required",
        },
        {
            name: "invalid log level",
            modify: func(c *Config) {
                c.Logging.Level = "invalid"
            },
            wantErr: "Level: oneof",
        },
        {
            name: "TLS without cert",
            modify: func(c *Config) {
                c.Qdrant.TLS.Enabled = true
            },
            wantErr: "cert_file and key_file required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg := config.DefaultConfig()
            if tt.modify != nil {
                tt.modify(cfg)
            }

            err := config.Validate(cfg)

            if tt.wantErr == "" {
                require.NoError(t, err)
            } else {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.wantErr)
            }
        })
    }
}
```

---

## Environment Isolation

```go
func TestLoad_IsolatedEnv(t *testing.T) {
    // Save and clear environment
    origEnv := os.Environ()
    os.Clearenv()
    defer func() {
        os.Clearenv()
        for _, e := range origEnv {
            kv := strings.SplitN(e, "=", 2)
            os.Setenv(kv[0], kv[1])
        }
    }()

    // Set only test env vars
    os.Setenv("CONTEXTD_SERVER_GRPC_PORT", "9999")

    cfg, err := config.Load("", nil)
    require.NoError(t, err)
    assert.Equal(t, 9999, cfg.Server.GRPC.Port)
}
```

---

## Coverage Requirements

| Area | Minimum |
|------|---------|
| Load function | 100% |
| Validation | 100% |
| Custom validators | 100% |
| Cross-field validation | 100% |
| Secret type | 100% |
| Duration type | 100% |
| **Overall** | **>80%** |
