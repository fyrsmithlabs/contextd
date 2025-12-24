# Configuration Structs

**Parent**: @./SPEC.md

Go struct definitions for configuration.

---

## Root Config

```go
// internal/config/config.go

package config

import "time"

// Config is the root configuration struct
type Config struct {
    Server    ServerConfig    `koanf:"server"`
    Qdrant    QdrantConfig    `koanf:"qdrant"`
    Telemetry TelemetryConfig `koanf:"telemetry"`
    Logging   LoggingConfig   `koanf:"logging"`
    Scrubber  ScrubberConfig  `koanf:"scrubber"`
    Session   SessionConfig   `koanf:"session"`
    Memory    MemoryConfig    `koanf:"memory"`
    Tools     ToolsConfig     `koanf:"tools"`
    Tenancy   TenancyConfig   `koanf:"tenancy"`
}
```

---

## Server Config

```go
// internal/config/server.go

type ServerConfig struct {
    GRPC            GRPCConfig `koanf:"grpc"`
    HTTP            HTTPConfig `koanf:"http"`
    ShutdownTimeout Duration   `koanf:"shutdown_timeout" validate:"required,min=1s"`
}

type GRPCConfig struct {
    Port           int             `koanf:"port" validate:"required,min=1,max=65535"`
    MaxRecvMsgSize int             `koanf:"max_recv_msg_size" validate:"min=0"`
    MaxSendMsgSize int             `koanf:"max_send_msg_size" validate:"min=0"`
    Keepalive      KeepaliveConfig `koanf:"keepalive"`
}

type HTTPConfig struct {
    Port         int      `koanf:"port" validate:"required,min=1,max=65535"`
    ReadTimeout  Duration `koanf:"read_timeout"`
    WriteTimeout Duration `koanf:"write_timeout"`
    IdleTimeout  Duration `koanf:"idle_timeout"`
}

type KeepaliveConfig struct {
    Time    Duration `koanf:"time"`
    Timeout Duration `koanf:"timeout"`
    MinTime Duration `koanf:"min_time"`
}
```

---

## Qdrant Config

```go
// internal/config/qdrant.go

type QdrantConfig struct {
    Host    string     `koanf:"host" validate:"required"`
    Port    int        `koanf:"port" validate:"required,min=1,max=65535"`
    APIKey  Secret     `koanf:"api_key"`  // Redacted in logs
    TLS     TLSConfig  `koanf:"tls"`
    Timeout Duration   `koanf:"timeout" validate:"min=1s"`
    Pool    PoolConfig `koanf:"pool"`
}

type TLSConfig struct {
    Enabled  bool   `koanf:"enabled"`
    CertFile string `koanf:"cert_file" validate:"required_if=Enabled true,file"`
    KeyFile  string `koanf:"key_file" validate:"required_if=Enabled true,file"`
    CAFile   string `koanf:"ca_file" validate:"omitempty,file"`
}

type PoolConfig struct {
    MaxConnections int `koanf:"max_connections" validate:"min=1"`
    MinConnections int `koanf:"min_connections" validate:"min=0"`
}
```

---

## Telemetry Config

```go
// internal/config/telemetry.go

type TelemetryConfig struct {
    Enabled        bool   `koanf:"enabled"`
    ServiceName    string `koanf:"service_name" validate:"required"`
    ServiceVersion string `koanf:"service_version"`

    Endpoint string `koanf:"endpoint" validate:"required_if=Enabled true"`
    Protocol string `koanf:"protocol" validate:"oneof=grpc http"`
    Insecure bool   `koanf:"insecure"`

    Sampling          SamplingConfig          `koanf:"sampling"`
    Metrics           MetricsConfig           `koanf:"metrics"`
    Traces            TracesConfig            `koanf:"traces"`
    ExperienceMetrics ExperienceMetricsConfig `koanf:"experience_metrics"`
}

type SamplingConfig struct {
    Rate           float64 `koanf:"rate" validate:"min=0,max=1"`
    AlwaysOnErrors bool    `koanf:"always_on_errors"`
}

type MetricsConfig struct {
    Enabled        bool     `koanf:"enabled"`
    ExportInterval Duration `koanf:"export_interval" validate:"min=1s"`
}

type TracesConfig struct {
    Enabled bool `koanf:"enabled"`
}

type ExperienceMetricsConfig struct {
    Enabled       bool `koanf:"enabled"`
    RetentionDays int  `koanf:"retention_days" validate:"min=1,max=365"`
}
```

---

## Logging Config

```go
// internal/config/logging.go

type LoggingConfig struct {
    Level  string `koanf:"level" validate:"oneof=trace debug info warn error"`
    Format string `koanf:"format" validate:"oneof=json console"`

    Output     LogOutputConfig   `koanf:"output"`
    Sampling   LogSamplingConfig `koanf:"sampling"`
    Caller     CallerConfig      `koanf:"caller"`
    Stacktrace StacktraceConfig  `koanf:"stacktrace"`
    Fields     map[string]string `koanf:"fields"`
    Redaction  RedactionConfig   `koanf:"redaction"`
}

type LogOutputConfig struct {
    Stdout bool `koanf:"stdout"`
    OTEL   bool `koanf:"otel"`
}

type LogSamplingConfig struct {
    Enabled bool                           `koanf:"enabled"`
    Tick    Duration                       `koanf:"tick"`
    Levels  map[string]LevelSamplingConfig `koanf:"levels"`
}

type LevelSamplingConfig struct {
    Initial    int `koanf:"initial" validate:"min=0"`
    Thereafter int `koanf:"thereafter" validate:"min=0"`
}

type CallerConfig struct {
    Enabled bool `koanf:"enabled"`
    Skip    int  `koanf:"skip" validate:"min=0"`
}

type StacktraceConfig struct {
    Level string `koanf:"level" validate:"oneof=trace debug info warn error"`
}

type RedactionConfig struct {
    Enabled  bool     `koanf:"enabled"`
    Fields   []string `koanf:"fields"`
    Patterns []string `koanf:"patterns"`
}
```

---

## Custom Types

```go
// internal/config/types.go

// Duration wraps time.Duration for YAML unmarshaling
type Duration time.Duration

func (d *Duration) UnmarshalText(text []byte) error {
    parsed, err := time.ParseDuration(string(text))
    if err != nil {
        return err
    }
    *d = Duration(parsed)
    return nil
}

func (d Duration) Duration() time.Duration {
    return time.Duration(d)
}

// Secret wraps strings that should be redacted in logs
type Secret string

func (s Secret) String() string {
    if s == "" {
        return ""
    }
    return "[REDACTED]"
}

func (s Secret) Value() string {
    return string(s)
}

func (s Secret) MarshalJSON() ([]byte, error) {
    return []byte(`"[REDACTED]"`), nil
}
```

---

## Defaults

```go
// internal/config/defaults.go

func DefaultConfig() *Config {
    return &Config{
        Server: ServerConfig{
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
        },
        Qdrant: QdrantConfig{
            Host:    "localhost",
            Port:    6334,
            Timeout: Duration(30 * time.Second),
            Pool: PoolConfig{
                MaxConnections: 10,
                MinConnections: 2,
            },
        },
        Telemetry: TelemetryConfig{
            Enabled:     true,
            ServiceName: "contextd",
            Endpoint:    "localhost:4317",
            Protocol:    "grpc",
            Insecure:    true,
            Sampling: SamplingConfig{
                Rate:           1.0,
                AlwaysOnErrors: true,
            },
            Metrics: MetricsConfig{
                Enabled:        true,
                ExportInterval: Duration(15 * time.Second),
            },
            Traces: TracesConfig{Enabled: true},
        },
        Logging: LoggingConfig{
            Level:  "info",
            Format: "json",
            Output: LogOutputConfig{Stdout: true, OTEL: true},
            Redaction: RedactionConfig{
                Enabled: true,
                Fields:  []string{"password", "secret", "token", "api_key"},
            },
        },
        // ... additional defaults
    }
}
```
