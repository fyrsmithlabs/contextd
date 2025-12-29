# Configuration Validation

**Parent**: @./SPEC.md

Validation rules, custom validators, and cross-field validation.

---

## Validation Framework

Using `go-playground/validator` for struct validation.

```go
// internal/config/validation.go

import "github.com/go-playground/validator/v10"

var validate *validator.Validate

func init() {
    validate = validator.New(validator.WithRequiredStructEnabled())

    // Register custom validators
    validate.RegisterValidation("file", validateFileExists)
    validate.RegisterValidation("dir", validateDirExists)
    validate.RegisterValidation("listen_addr", validateListenAddr)
}

func Validate(cfg *Config) error {
    if err := validate.Struct(cfg); err != nil {
        return formatValidationErrors(err)
    }

    // Cross-field validations
    if err := validateCrossFields(cfg); err != nil {
        return err
    }

    return nil
}
```

---

## Validation Tags Reference

| Tag | Usage | Example |
|-----|-------|---------|
| `required` | Field must be set | `validate:"required"` |
| `min`, `max` | Numeric bounds | `validate:"min=1,max=65535"` |
| `oneof` | Enum values | `validate:"oneof=grpc http"` |
| `required_if` | Conditional required | `validate:"required_if=Enabled true"` |
| `file` | Path is existing file | `validate:"file"` |
| `dir` | Path is existing directory | `validate:"dir"` |
| `omitempty` | Skip if empty | `validate:"omitempty,file"` |

---

## Cross-Field Validation

```go
func validateCrossFields(cfg *Config) error {
    // TLS requires cert and key
    if cfg.Qdrant.TLS.Enabled {
        if cfg.Qdrant.TLS.CertFile == "" || cfg.Qdrant.TLS.KeyFile == "" {
            return fmt.Errorf("qdrant.tls: cert_file and key_file required when tls enabled")
        }
    }

    // OTEL output requires telemetry enabled
    if cfg.Logging.Output.OTEL && !cfg.Telemetry.Enabled {
        return fmt.Errorf("logging.output.otel requires telemetry.enabled=true")
    }

    // Multi-tenancy requires JWT secret
    if cfg.Tenancy.Enabled && cfg.Tenancy.JWT.Secret == "" {
        return fmt.Errorf("tenancy.jwt.secret required when tenancy enabled")
    }

    return nil
}
```

---

## Custom Validators

```go
// File path validators
func validateFileExists(fl validator.FieldLevel) bool {
    path := fl.Field().String()
    if path == "" {
        return true // Empty is valid (use required for mandatory)
    }
    info, err := os.Stat(path)
    return err == nil && !info.IsDir()
}

func validateDirExists(fl validator.FieldLevel) bool {
    path := fl.Field().String()
    if path == "" {
        return true
    }
    info, err := os.Stat(path)
    return err == nil && info.IsDir()
}

func validateListenAddr(fl validator.FieldLevel) bool {
    addr := fl.Field().String()
    if addr == "" {
        return true
    }
    _, _, err := net.SplitHostPort(addr)
    return err == nil
}
```

---

## Error Formatting

```go
func formatValidationErrors(err error) error {
    var errs validator.ValidationErrors
    if errors.As(err, &errs) {
        var messages []string
        for _, e := range errs {
            messages = append(messages, fmt.Sprintf(
                "%s: %s (value: %v)",
                e.Namespace(), e.Tag(), e.Value(),
            ))
        }
        return fmt.Errorf("validation failed:\n  %s", strings.Join(messages, "\n  "))
    }
    return err
}
```

---

## Error Output Example

```
Error: validate config: validation failed:
  Server.GRPC.Port: required (value: 0)
  Qdrant.Host: required (value: )
  Telemetry.Sampling.Rate: max (value: 1.5)
```

---

## Validation Rules by Section

### Server

| Field | Rules |
|-------|-------|
| `grpc.port` | required, 1-65535 |
| `http.port` | required, 1-65535 |
| `shutdown_timeout` | required, min 1s |

### Qdrant

| Field | Rules |
|-------|-------|
| `host` | required |
| `port` | required, 1-65535 |
| `timeout` | min 1s |
| `tls.cert_file` | required if tls.enabled, must exist |
| `tls.key_file` | required if tls.enabled, must exist |

### Telemetry

| Field | Rules |
|-------|-------|
| `service_name` | required |
| `endpoint` | required if enabled |
| `protocol` | oneof: grpc, http |
| `sampling.rate` | 0.0-1.0 |

### Logging

| Field | Rules |
|-------|-------|
| `level` | oneof: trace, debug, info, warn, error |
| `format` | oneof: json, console |
| `stacktrace.level` | oneof: trace, debug, info, warn, error |

---

## Adding Custom Validators

1. Implement `func(validator.FieldLevel) bool`
2. Register in `init()`: `validate.RegisterValidation("name", fn)`
3. Use in struct tag: `validate:"name"`

```go
// Example: validate port range
func validatePortRange(fl validator.FieldLevel) bool {
    port := fl.Field().Int()
    return port >= 1024 && port <= 49151
}

// Register
validate.RegisterValidation("user_port", validatePortRange)

// Use
type Config struct {
    Port int `validate:"user_port"`
}
```
