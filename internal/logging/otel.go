// internal/logging/otel.go
package logging

import (
	"fmt"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap/zapcore"
)

// newDualCore creates core with stdout and/or OTEL outputs.
func newDualCore(cfg *Config, otelProvider log.LoggerProvider) (zapcore.Core, error) {
	cores := make([]zapcore.Core, 0, 2)

	if cfg.Output.Stdout {
		baseEncoder := newEncoder(cfg.Format)
		encoder, err := NewRedactingEncoder(baseEncoder, cfg.Redaction)
		if err != nil {
			return nil, fmt.Errorf("failed to create redacting encoder: %w", err)
		}
		writer := zapcore.AddSync(os.Stdout)
		cores = append(cores, zapcore.NewCore(encoder, writer, cfg.Level))
	}

	if cfg.Output.OTEL && otelProvider != nil {
		otelCore := otelzap.NewCore("contextd",
			otelzap.WithLoggerProvider(otelProvider),
		)
		cores = append(cores, otelCore)
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("at least one output must be enabled and available")
	}

	var core zapcore.Core
	if len(cores) == 1 {
		core = cores[0]
	} else {
		core = zapcore.NewTee(cores...)
	}

	// Wrap with sampling if enabled
	core = newSampledCore(core, cfg.Sampling)

	return core, nil
}
