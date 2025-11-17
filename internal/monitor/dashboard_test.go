package monitor

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewModel(t *testing.T) {
	model := NewModel("http://localhost:8428", 5*time.Second)
	assert.Equal(t, "http://localhost:8428", model.vmURL)
	assert.Equal(t, 5*time.Second, model.interval)
	assert.False(t, model.quitting)
}

func TestModel_Init(t *testing.T) {
	model := NewModel("http://localhost:8428", 5*time.Second)
	cmd := model.Init()

	// Init should return a tick command to start auto-refresh
	assert.NotNil(t, cmd)
}

func TestModel_Update_QuitKey(t *testing.T) {
	model := NewModel("http://localhost:8428", 5*time.Second)

	// Send 'q' key message
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel, cmd := model.Update(keyMsg)

	// Model should be marked as quitting
	m := updatedModel.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd) // Should return tea.Quit
}

func TestModel_Update_RefreshKey(t *testing.T) {
	model := NewModel("http://localhost:8428", 5*time.Second)

	// Send 'r' key message
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updatedModel, cmd := model.Update(keyMsg)

	// Should trigger metrics fetch
	m := updatedModel.(Model)
	assert.False(t, m.quitting)
	assert.NotNil(t, cmd) // Should return fetchMetrics command
}

func TestModel_Update_TickMsg(t *testing.T) {
	model := NewModel("http://localhost:8428", 5*time.Second)

	// Send tick message
	msg := tickMsg(time.Now())
	updatedModel, cmd := model.Update(msg)

	// Should schedule next tick and fetch metrics
	m := updatedModel.(Model)
	assert.False(t, m.quitting)
	assert.NotNil(t, cmd) // Should return batch command (tick + fetchMetrics)
}

func TestModel_Update_MetricsMsg(t *testing.T) {
	model := NewModel("http://localhost:8428", 5*time.Second)

	// Send metrics message
	metrics := metricsMsg(MetricsSnapshot{
		HTTPRate:       45.7,
		HTTPLatencyP95: 0.0123,
		EmbeddingRate:  120.0,
	})
	updatedModel, cmd := model.Update(metrics)

	// Model should update metrics and lastUpdate time
	m := updatedModel.(Model)
	assert.Equal(t, 45.7, m.metrics.HTTPRate)
	assert.Equal(t, 0.0123, m.metrics.HTTPLatencyP95)
	assert.Equal(t, 120.0, m.metrics.EmbeddingRate)
	assert.False(t, m.lastUpdate.IsZero())
	assert.Nil(t, cmd) // No command needed after metrics update
}

func TestModel_Update_ErrMsg(t *testing.T) {
	model := NewModel("http://localhost:8428", 5*time.Second)

	// Send error message
	msg := errMsg(fmt.Errorf("connection refused"))
	updatedModel, cmd := model.Update(msg)

	// Model should store error
	m := updatedModel.(Model)
	assert.NotNil(t, m.err)
	assert.Contains(t, m.err.Error(), "connection refused")
	assert.Nil(t, cmd)
}

func TestModel_View_WithMetrics(t *testing.T) {
	model := NewModel("http://localhost:8428", 5*time.Second)
	model.metrics = MetricsSnapshot{
		HTTPRate:        45.7,
		HTTPLatencyP95:  0.0123,
		EmbeddingRate:   120.0,
		EmbeddingTokens: 15200.0,
		EmbeddingCost:   0.0034,
		Uptime:          8100, // 2h 15m
		Goroutines:      42,
		MemoryMB:        24.5,
	}
	model.lastUpdate = time.Date(2024, 1, 1, 12, 34, 56, 0, time.UTC)

	view := model.View()

	// Verify view contains expected elements
	assert.Contains(t, view, "contextd Monitor")
	assert.Contains(t, view, "12:34:56")
	assert.Contains(t, view, "HTTP Requests")
	assert.Contains(t, view, "45.7 req/min")
	assert.Contains(t, view, "12.3ms")
	assert.Contains(t, view, "Embeddings")
	assert.Contains(t, view, "120.0 req/min")
	assert.Contains(t, view, "$0.0034/min")
	assert.Contains(t, view, "System")
	assert.Contains(t, view, "42")
	assert.Contains(t, view, "[q]")
	assert.Contains(t, view, "[r]")
}

func TestModel_View_WithError(t *testing.T) {
	model := NewModel("http://localhost:8428", 5*time.Second)
	model.err = fmt.Errorf("connection refused")

	view := model.View()

	// Verify error message is displayed
	assert.Contains(t, view, "Cannot connect to VictoriaMetrics")
	assert.Contains(t, view, "connection refused")
	assert.Contains(t, view, "http://localhost:8428")
	assert.Contains(t, view, "[q]")
	assert.Contains(t, view, "[r]")
}

func TestModel_View_NoData(t *testing.T) {
	model := NewModel("http://localhost:8428", 5*time.Second)
	// No metrics, no error

	view := model.View()

	// Should show waiting message or empty metrics
	assert.Contains(t, view, "contextd Monitor")
	assert.Contains(t, view, "[q]")
}
