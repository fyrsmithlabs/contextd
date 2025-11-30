# Configuration Architecture

**Parent**: @./SPEC.md

---

## Loading Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Configuration Loading                     │
│                                                              │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐ │
│  │ Defaults │ → │   File   │ → │   Env    │ → │  Flags   │ │
│  │ (code)   │   │ (YAML)   │   │ (vars)   │   │ (CLI)    │ │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘ │
│        │              │              │              │        │
│        └──────────────┴──────────────┴──────────────┘        │
│                              │                                │
│                              ▼                                │
│                    ┌─────────────────┐                       │
│                    │  Koanf Merge    │                       │
│                    └────────┬────────┘                       │
│                             │                                 │
│                             ▼                                 │
│                    ┌─────────────────┐                       │
│                    │   Unmarshal     │                       │
│                    │   to Struct     │                       │
│                    └────────┬────────┘                       │
│                             │                                 │
│                             ▼                                 │
│                    ┌─────────────────┐                       │
│                    │   Validate      │                       │
│                    │ (go-playground) │                       │
│                    └────────┬────────┘                       │
│                             │                                 │
│                             ▼                                 │
│                    ┌─────────────────┐                       │
│                    │  Config Struct  │                       │
│                    │   (immutable)   │                       │
│                    └─────────────────┘                       │
└─────────────────────────────────────────────────────────────┘
```

---

## Technology Choice: Koanf

| Factor | Koanf | Viper | Decision |
|--------|-------|-------|----------|
| Binary size | 3x smaller | Larger | Koanf |
| Dependencies | Modular | All bundled | Koanf |
| Key handling | Preserves case | Forces lowercase | Koanf |
| API | Simple, clear | Feature-rich | Koanf |

**Decision**: Koanf v2 for simplicity, modularity, and correct key handling.

---

## Load Implementation

```go
// internal/config/config.go

import (
    "github.com/knadh/koanf/v2"
    "github.com/knadh/koanf/parsers/yaml"
    "github.com/knadh/koanf/providers/env"
    "github.com/knadh/koanf/providers/file"
    "github.com/knadh/koanf/providers/posflag"
    "github.com/knadh/koanf/providers/structs"
    "github.com/go-playground/validator/v10"
)

// Load loads configuration from all sources
func Load(configPath string, flags *pflag.FlagSet) (*Config, error) {
    k := koanf.New(".")

    // 1. Load defaults
    if err := k.Load(structs.Provider(DefaultConfig(), "koanf"), nil); err != nil {
        return nil, fmt.Errorf("load defaults: %w", err)
    }

    // 2. Load config file
    path := resolveConfigPath(configPath)
    if path != "" {
        if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
            return nil, fmt.Errorf("load config file %s: %w", path, err)
        }
    }

    // 3. Load environment variables
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

    // 4. Load CLI flags (highest priority)
    if flags != nil {
        if err := k.Load(posflag.Provider(flags, ".", k), nil); err != nil {
            return nil, fmt.Errorf("load flags: %w", err)
        }
    }

    // 5. Unmarshal to struct
    var cfg Config
    if err := k.Unmarshal("", &cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }

    // 6. Validate
    if err := Validate(&cfg); err != nil {
        return nil, fmt.Errorf("validate config: %w", err)
    }

    return &cfg, nil
}
```

---

## Config Path Resolution

```go
func resolveConfigPath(explicit string) string {
    if explicit != "" {
        return explicit
    }

    paths := []string{
        "./config.yaml",
        "./contextd.yaml",
        filepath.Join(os.Getenv("HOME"), ".config/contextd/config.yaml"),
        "/etc/contextd/config.yaml",
    }

    for _, p := range paths {
        if _, err := os.Stat(p); err == nil {
            return p
        }
    }

    return "" // No config file found, use defaults + env
}
```

---

## CLI Integration

```go
// cmd/contextd/main.go

import "github.com/spf13/pflag"

func main() {
    // Define flags
    flags := pflag.NewFlagSet("contextd", pflag.ExitOnError)
    configPath := flags.StringP("config", "c", "", "config file path")
    flags.Int("server.grpc.port", 0, "gRPC server port")
    flags.String("logging.level", "", "log level")
    flags.Parse(os.Args[1:])

    // Load config
    cfg, err := config.Load(*configPath, flags)
    if err != nil {
        log.Fatalf("config error: %v", err)
    }

    // Use config
    server := NewServer(cfg)
    server.Start()
}
```

---

## Immutability

Once loaded, config is immutable:

- No setter methods
- Struct fields not exported for modification
- Use `Load()` to create new config instance

---

## Error Handling

Config errors are fatal at startup:

| Error Type | Behavior |
|------------|----------|
| File not found | OK if using defaults + env |
| Parse error | Fatal, log path and line |
| Validation error | Fatal, log field and constraint |
| Cross-field error | Fatal, log dependency |
