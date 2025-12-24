// internal/logging/sampling.go
package logging

import (
	"go.uber.org/zap/zapcore"
)

// newSampledCore wraps core with level-aware sampling.
// Error and above are never sampled.
func newSampledCore(core zapcore.Core, cfg SamplingConfig) zapcore.Core {
	if !cfg.Enabled {
		return core
	}

	// Errors and above always pass through
	errorCore := &levelFilterCore{
		Core:     core,
		minLevel: zapcore.ErrorLevel,
	}

	// Below error gets sampled
	belowErrorCore := &levelFilterCore{
		Core:     core,
		maxLevel: zapcore.WarnLevel,
	}

	// Get sampling config for Info level (default)
	infoSampling := cfg.Levels[zapcore.InfoLevel]

	sampledCore := zapcore.NewSamplerWithOptions(
		belowErrorCore,
		cfg.Tick.Duration(),
		infoSampling.Initial,
		infoSampling.Thereafter,
	)

	return zapcore.NewTee(errorCore, sampledCore)
}

// levelFilterCore filters logs by level range.
type levelFilterCore struct {
	zapcore.Core
	minLevel zapcore.Level // only log >= minLevel (0 = no min)
	maxLevel zapcore.Level // only log <= maxLevel (0 = no max)
}

func (c *levelFilterCore) Enabled(lvl zapcore.Level) bool {
	if c.minLevel != 0 && lvl < c.minLevel {
		return false
	}
	if c.maxLevel != 0 && lvl > c.maxLevel {
		return false
	}
	return c.Core.Enabled(lvl)
}

func (c *levelFilterCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(e.Level) {
		return ce
	}
	return c.Core.Check(e, ce)
}

// With creates a child logger that preserves level filtering.
func (c *levelFilterCore) With(fields []zapcore.Field) zapcore.Core {
	return &levelFilterCore{
		Core:     c.Core.With(fields),
		minLevel: c.minLevel,
		maxLevel: c.maxLevel,
	}
}
