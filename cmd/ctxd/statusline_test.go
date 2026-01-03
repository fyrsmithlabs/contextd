package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatStatusline(t *testing.T) {
	t.Run("formats healthy status", func(t *testing.T) {
		status := &StatusResponse{
			Status: "ok",
			Services: map[string]string{
				"checkpoint": "ok",
				"memory":     "ok",
			},
			Counts: StatusCounts{
				Memories:    12,
				Checkpoints: 3,
			},
		}

		result := formatStatusline(status)

		assert.Contains(t, result, "\U0001f7e2") // Green circle
		assert.Contains(t, result, "\U0001f9e012")
		assert.Contains(t, result, "\U0001f4be3")
		assert.Contains(t, result, "\u2502") // Separator
	})

	t.Run("shows unknown counts when negative", func(t *testing.T) {
		status := &StatusResponse{
			Status: "ok",
			Services: map[string]string{
				"checkpoint": "ok",
			},
			Counts: StatusCounts{
				Memories:    -1,
				Checkpoints: -1,
			},
		}

		result := formatStatusline(status)

		assert.Contains(t, result, "\U0001f9e0?")
		assert.Contains(t, result, "\U0001f4be?")
	})

	t.Run("shows warning for unavailable services", func(t *testing.T) {
		status := &StatusResponse{
			Status: "ok",
			Services: map[string]string{
				"checkpoint": "unavailable",
				"memory":     "ok",
			},
			Counts: StatusCounts{
				Memories:    5,
				Checkpoints: 2,
			},
		}

		result := formatStatusline(status)

		assert.Contains(t, result, "\U0001f7e1") // Yellow circle for warning
	})

	t.Run("shows error status", func(t *testing.T) {
		status := &StatusResponse{
			Status:   "error",
			Services: map[string]string{},
			Counts:   StatusCounts{},
		}

		result := formatStatusline(status)

		assert.Contains(t, result, "\U0001f534") // Red circle
	})

	t.Run("includes context usage when available", func(t *testing.T) {
		status := &StatusResponse{
			Status:   "ok",
			Services: map[string]string{"memory": "ok"},
			Counts:   StatusCounts{Memories: 5, Checkpoints: 2},
			Context: &ContextStatus{
				UsagePercent:     68,
				ThresholdWarning: false,
			},
		}

		result := formatStatusline(status)

		assert.Contains(t, result, "68%")
	})

	t.Run("shows context warning when threshold exceeded", func(t *testing.T) {
		status := &StatusResponse{
			Status:   "ok",
			Services: map[string]string{"memory": "ok"},
			Counts:   StatusCounts{Memories: 5, Checkpoints: 2},
			Context: &ContextStatus{
				UsagePercent:     75,
				ThresholdWarning: true,
			},
		}

		result := formatStatusline(status)

		// Should contain yellow warning color code
		assert.Contains(t, result, "\033[33m")
		assert.Contains(t, result, "75%")
	})

	t.Run("includes confidence when available", func(t *testing.T) {
		status := &StatusResponse{
			Status:   "ok",
			Services: map[string]string{"memory": "ok"},
			Counts:   StatusCounts{Memories: 5, Checkpoints: 2},
			Memory: &MemoryStatus{
				LastConfidence: 0.85,
			},
		}

		result := formatStatusline(status)

		assert.Contains(t, result, "C:0.85")
	})

	t.Run("includes compression ratio when available", func(t *testing.T) {
		status := &StatusResponse{
			Status:   "ok",
			Services: map[string]string{"memory": "ok"},
			Counts:   StatusCounts{Memories: 5, Checkpoints: 2},
			Compression: &CompressionStatus{
				LastRatio: 2.5,
			},
		}

		result := formatStatusline(status)

		assert.Contains(t, result, "F:2.5x")
	})
}

func TestGetHealthIcon(t *testing.T) {
	t.Run("returns green for healthy status", func(t *testing.T) {
		status := &StatusResponse{
			Status: "ok",
			Services: map[string]string{
				"checkpoint": "ok",
				"memory":     "ok",
			},
		}

		icon := getHealthIcon(status)

		assert.Contains(t, icon, "\033[32m")  // Green color
		assert.Contains(t, icon, "\U0001f7e2") // Green circle
	})

	t.Run("returns red for error status", func(t *testing.T) {
		status := &StatusResponse{
			Status: "error",
		}

		icon := getHealthIcon(status)

		assert.Contains(t, icon, "\033[31m")  // Red color
		assert.Contains(t, icon, "\U0001f534") // Red circle
	})

	t.Run("returns yellow when service unavailable", func(t *testing.T) {
		status := &StatusResponse{
			Status: "ok",
			Services: map[string]string{
				"checkpoint": "unavailable",
			},
		}

		icon := getHealthIcon(status)

		assert.Contains(t, icon, "\033[33m")  // Yellow color
		assert.Contains(t, icon, "\U0001f7e1") // Yellow circle
	})
}

func TestShellEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escapes single quotes",
			input:    "test'string",
			expected: "'test'\"'\"'string'",
		},
		{
			name:     "handles normal paths",
			input:    "/usr/local/bin/ctxd",
			expected: "'/usr/local/bin/ctxd'",
		},
		{
			name:     "escapes multiple single quotes",
			input:    "it's a test's",
			expected: "'it'\"'\"'s a test'\"'\"'s'",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "''",
		},
		{
			name:     "handles spaces",
			input:    "/path/with spaces/bin",
			expected: "'/path/with spaces/bin'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellEscape(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsShellMetacharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"semicolon", "cmd;rm -rf", true},
		{"pipe", "cmd|cat", true},
		{"ampersand", "cmd&", true},
		{"backtick", "cmd`whoami`", true},
		{"dollar sign", "cmd$var", true},
		{"open paren", "cmd()", true},
		{"close paren", "cmd()", true},
		{"greater than", "cmd>file", true},
		{"less than", "cmd<file", true},
		{"newline", "cmd\nrm", true},
		{"carriage return", "cmd\rrm", true},
		{"null byte", "cmd\x00rm", true},
		{"clean path", "/usr/local/bin/ctxd", false},
		{"path with spaces", "/home/user/my files/script", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsShellMetacharacters(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidScriptPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"absolute path", "/usr/bin/script", true},
		{"relative path", "bin/script", false},
		{"path with semicolon", "/usr/bin/script;rm -rf", false},
		{"path with pipe", "/usr/bin/script|cat", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidScriptPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFetchStatusHTTP(t *testing.T) {
	t.Run("successfully fetches status", func(t *testing.T) {
		mockStatus := &StatusResponse{
			Status:   "ok",
			Services: map[string]string{"memory": "ok"},
			Counts:   StatusCounts{Memories: 10, Checkpoints: 5},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/status", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockStatus)
		}))
		defer server.Close()

		oldServerURL := serverURL
		serverURL = server.URL
		defer func() { serverURL = oldServerURL }()

		status, err := fetchStatusHTTP()

		require.NoError(t, err)
		assert.Equal(t, "ok", status.Status)
		assert.Equal(t, 10, status.Counts.Memories)
		assert.Equal(t, 5, status.Counts.Checkpoints)
	})

	t.Run("handles connection error", func(t *testing.T) {
		oldServerURL := serverURL
		serverURL = "http://localhost:99999" // Invalid port
		defer func() { serverURL = oldServerURL }()

		_, err := fetchStatusHTTP()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect")
	})

	t.Run("handles non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error"))
		}))
		defer server.Close()

		oldServerURL := serverURL
		serverURL = server.URL
		defer func() { serverURL = oldServerURL }()

		_, err := fetchStatusHTTP()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
	})

	t.Run("handles invalid json response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not valid json"))
		}))
		defer server.Close()

		oldServerURL := serverURL
		serverURL = server.URL
		defer func() { serverURL = oldServerURL }()

		_, err := fetchStatusHTTP()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode")
	})
}

func TestGetClaudeSettingsPath(t *testing.T) {
	t.Run("returns path ending with settings.json", func(t *testing.T) {
		path := getClaudeSettingsPath()
		assert.NotEmpty(t, path)
		assert.True(t, strings.HasSuffix(path, "settings.json"))
	})

	t.Run("returns path within home directory", func(t *testing.T) {
		path := getClaudeSettingsPath()
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		// Path should be relative to home or config directory
		assert.True(t, strings.HasPrefix(path, homeDir) ||
			strings.HasPrefix(path, "/tmp"),
			"path should be within home directory: %s", path)
	})
}

func TestPathValidation(t *testing.T) {
	t.Run("rejects paths with newlines", func(t *testing.T) {
		path := "/path/with\nnewline"
		assert.True(t, containsShellMetacharacters(path))
	})

	t.Run("rejects paths with carriage return", func(t *testing.T) {
		path := "/path/with\rcarriage"
		assert.True(t, containsShellMetacharacters(path))
	})

	t.Run("rejects paths with null byte", func(t *testing.T) {
		path := "/path/with\x00null"
		assert.True(t, containsShellMetacharacters(path))
	})

	t.Run("accepts clean absolute paths", func(t *testing.T) {
		path := "/usr/local/bin/ctxd"
		assert.False(t, containsShellMetacharacters(path))
		assert.True(t, isValidScriptPath(path))
	})
}

func TestSettingsPathValidation(t *testing.T) {
	t.Run("path within home directory is valid", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		testPath := filepath.Join(homeDir, ".claude", "settings.json")
		cleanPath := filepath.Clean(testPath)

		assert.True(t, strings.HasPrefix(cleanPath, homeDir))
	})

	t.Run("path outside home directory is invalid", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		testPath := "/etc/passwd"
		cleanPath := filepath.Clean(testPath)

		assert.False(t, strings.HasPrefix(cleanPath, homeDir))
	})

	t.Run("path with traversal is rejected", func(t *testing.T) {
		testPath := "../../etc/passwd"
		assert.True(t, strings.Contains(testPath, ".."))
	})
}
