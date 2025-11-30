package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDualCore_StdoutOnly(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Output.Stdout = true
	cfg.Output.OTEL = false

	core, err := newDualCore(cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, core)
}

func TestNewDualCore_BothOutputs(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Output.Stdout = true
	cfg.Output.OTEL = true

	// For testing, pass nil provider
	// In production, would provide real OTEL provider
	core, err := newDualCore(cfg, nil)

	// Should succeed with stdout, skip OTEL if provider nil
	require.NoError(t, err)
	assert.NotNil(t, core)
}

func TestNewDualCore_NoOutputs(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Output.Stdout = false
	cfg.Output.OTEL = false

	_, err := newDualCore(cfg, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one output")
}
