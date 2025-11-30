package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResource(t *testing.T) {
	cfg := NewDefaultConfig()

	res, err := newResource(cfg)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Resource should contain service name attribute
	attrs := res.Attributes()
	var foundServiceName bool
	for _, attr := range attrs {
		if string(attr.Key) == "service.name" {
			assert.Equal(t, cfg.ServiceName, attr.Value.AsString())
			foundServiceName = true
		}
	}
	assert.True(t, foundServiceName, "service.name attribute not found")
}

func TestTracerProviderOption(t *testing.T) {
	opts := &tracerProviderOptions{}

	// Default should be nil
	assert.Nil(t, opts.exporter)

	// WithTraceExporter should set exporter
	WithTraceExporter(nil)(opts)
	// Since we passed nil, it should still be nil
	assert.Nil(t, opts.exporter)
}

func TestMeterProviderOption(t *testing.T) {
	opts := &meterProviderOptions{}

	// Default should be nil
	assert.Nil(t, opts.exporter)

	// WithMetricExporter should set exporter
	WithMetricExporter(nil)(opts)
	// Since we passed nil, it should still be nil
	assert.Nil(t, opts.exporter)
}
