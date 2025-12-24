package secrets

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		s, err := New(nil)
		require.NoError(t, err)
		assert.NotNil(t, s)
		assert.True(t, s.IsEnabled())
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &Config{
			Enabled:         true,
			RedactionString: "[SCRUBBED]",
			Rules: []Rule{
				{
					ID:          "test-rule",
					Description: "Test rule",
					Pattern:     `secret123`,
					Severity:    "high",
				},
			},
		}
		s, err := New(cfg)
		require.NoError(t, err)
		assert.NotNil(t, s)
	})

	t.Run("with invalid pattern", func(t *testing.T) {
		cfg := &Config{
			Enabled: true,
			Rules: []Rule{
				{
					ID:      "bad-rule",
					Pattern: `[invalid`,
				},
			},
		}
		_, err := New(cfg)
		assert.Error(t, err)
	})

	t.Run("with missing ID", func(t *testing.T) {
		cfg := &Config{
			Enabled: true,
			Rules: []Rule{
				{
					Pattern: `test`,
				},
			},
		}
		_, err := New(cfg)
		assert.Error(t, err)
	})

	t.Run("with missing pattern", func(t *testing.T) {
		cfg := &Config{
			Enabled: true,
			Rules: []Rule{
				{
					ID: "test",
				},
			},
		}
		_, err := New(cfg)
		assert.Error(t, err)
	})

	t.Run("with invalid allow list pattern", func(t *testing.T) {
		cfg := &Config{
			Enabled:   true,
			Rules:     []Rule{{ID: "test", Pattern: `test`}},
			AllowList: []string{`[invalid`},
		}
		_, err := New(cfg)
		assert.Error(t, err)
	})
}

func TestMustNew(t *testing.T) {
	t.Run("panics on error", func(t *testing.T) {
		cfg := &Config{
			Enabled: true,
			Rules: []Rule{
				{ID: "bad", Pattern: `[invalid`},
			},
		}
		assert.Panics(t, func() {
			MustNew(cfg)
		})
	})

	t.Run("succeeds with valid config", func(t *testing.T) {
		assert.NotPanics(t, func() {
			s := MustNew(nil)
			assert.NotNil(t, s)
		})
	})
}

func TestScrubber_Scrub(t *testing.T) {
	s, err := New(nil)
	require.NoError(t, err)

	t.Run("detects AWS access key", func(t *testing.T) {
		content := "my key is AKIAIOSFODNN7EXAMPLE"
		result := s.Scrub(content)

		assert.True(t, result.HasFindings())
		assert.Equal(t, 1, result.TotalFindings)
		assert.Contains(t, result.Scrubbed, "[REDACTED]")
		assert.NotContains(t, result.Scrubbed, "AKIAIOSFODNN7EXAMPLE")
	})

	t.Run("detects GitHub token", func(t *testing.T) {
		content := "token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"
		result := s.Scrub(content)

		assert.True(t, result.HasFindings())
		assert.Contains(t, result.Scrubbed, "[REDACTED]")
	})

	t.Run("detects private key", func(t *testing.T) {
		content := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z3...
-----END RSA PRIVATE KEY-----`
		result := s.Scrub(content)

		assert.True(t, result.HasFindings())
	})

	t.Run("detects database URL with credentials", func(t *testing.T) {
		content := "DATABASE_URL=postgres://user:secretpass@localhost:5432/mydb"
		result := s.Scrub(content)

		assert.True(t, result.HasFindings())
	})

	t.Run("detects JWT token", func(t *testing.T) {
		content := "token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
		result := s.Scrub(content)

		assert.True(t, result.HasFindings())
	})

	t.Run("detects Stripe key", func(t *testing.T) {
		content := "stripe_key: sk_live_abcdefghijklmnopqrstuvwxyz"
		result := s.Scrub(content)

		assert.True(t, result.HasFindings())
	})

	t.Run("detects Slack token", func(t *testing.T) {
		content := "slack_token: xoxb-123456789012-abcdefghijkl"
		result := s.Scrub(content)

		assert.True(t, result.HasFindings())
	})

	t.Run("detects generic api key", func(t *testing.T) {
		content := "api_key = abc123def456ghi789jkl012mno"
		result := s.Scrub(content)

		assert.True(t, result.HasFindings())
	})

	t.Run("detects generic secret", func(t *testing.T) {
		content := "password: mysupersecretpassword123"
		result := s.Scrub(content)

		assert.True(t, result.HasFindings())
	})

	t.Run("no findings for clean content", func(t *testing.T) {
		content := "This is just regular text with no secrets."
		result := s.Scrub(content)

		assert.False(t, result.HasFindings())
		assert.Equal(t, content, result.Scrubbed)
	})

	t.Run("handles empty content", func(t *testing.T) {
		result := s.Scrub("")
		assert.False(t, result.HasFindings())
		assert.Equal(t, "", result.Scrubbed)
	})

	t.Run("multiple secrets in content", func(t *testing.T) {
		content := `
AWS_KEY=AKIAIOSFODNN7EXAMPLE
GITHUB_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij
`
		result := s.Scrub(content)

		assert.True(t, result.HasFindings())
		assert.GreaterOrEqual(t, result.TotalFindings, 2)
		assert.Contains(t, result.Scrubbed, "[REDACTED]")
		assert.NotContains(t, result.Scrubbed, "AKIAIOSFODNN7EXAMPLE")
		assert.NotContains(t, result.Scrubbed, "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij")
	})

	t.Run("tracks line numbers", func(t *testing.T) {
		content := "line1\nline2\nkey: AKIAIOSFODNN7EXAMPLE\nline4"
		result := s.Scrub(content)

		require.True(t, result.HasFindings())
		assert.Equal(t, 3, result.Findings[0].Line)
	})

	t.Run("reports duration", func(t *testing.T) {
		result := s.Scrub("some content")
		assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
	})

	t.Run("tracks by rule", func(t *testing.T) {
		content := "key: AKIAIOSFODNN7EXAMPLE"
		result := s.Scrub(content)

		assert.NotEmpty(t, result.ByRule)
	})
}

func TestScrubber_ScrubBytes(t *testing.T) {
	s, err := New(nil)
	require.NoError(t, err)

	content := []byte("api_key: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij")
	result := s.ScrubBytes(content)

	assert.True(t, result.HasFindings())
	assert.Contains(t, result.Scrubbed, "[REDACTED]")
}

func TestScrubber_Check(t *testing.T) {
	s, err := New(nil)
	require.NoError(t, err)

	content := "api_key: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"
	result := s.Check(content)

	assert.True(t, result.HasFindings())
	// Check mode should NOT redact
	assert.Equal(t, content, result.Scrubbed)
}

func TestScrubber_Disabled(t *testing.T) {
	cfg := &Config{
		Enabled: false,
	}
	s, err := New(cfg)
	require.NoError(t, err)

	assert.False(t, s.IsEnabled())

	content := "api_key: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"
	result := s.Scrub(content)

	assert.False(t, result.HasFindings())
	assert.Equal(t, content, result.Scrubbed)
}

func TestScrubber_AllowList(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		RedactionString: "[REDACTED]",
		Rules: []Rule{
			{
				ID:      "test",
				Pattern: `secret_\w+`,
			},
		},
		AllowList: []string{`secret_allowed`},
	}

	s, err := New(cfg)
	require.NoError(t, err)

	t.Run("allows whitelisted patterns", func(t *testing.T) {
		content := "secret_allowed is fine"
		result := s.Scrub(content)

		assert.False(t, result.HasFindings())
		assert.Equal(t, content, result.Scrubbed)
	})

	t.Run("still catches non-whitelisted", func(t *testing.T) {
		content := "secret_forbidden is not"
		result := s.Scrub(content)

		assert.True(t, result.HasFindings())
		assert.Contains(t, result.Scrubbed, "[REDACTED]")
	})
}

func TestScrubber_Keywords(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		RedactionString: "[REDACTED]",
		Rules: []Rule{
			{
				ID:       "with-keyword",
				Pattern:  `[A-Z]{20}`,
				Keywords: []string{"aws", "key"},
			},
		},
	}

	s, err := New(cfg)
	require.NoError(t, err)

	t.Run("matches when keyword present", func(t *testing.T) {
		content := "aws key: ABCDEFGHIJKLMNOPQRST"
		result := s.Scrub(content)

		assert.True(t, result.HasFindings())
	})

	t.Run("no match without keyword", func(t *testing.T) {
		content := "random: ABCDEFGHIJKLMNOPQRST"
		result := s.Scrub(content)

		assert.False(t, result.HasFindings())
	})
}

func TestScrubber_CustomRedaction(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		RedactionString: "***HIDDEN***",
		Rules: []Rule{
			{
				ID:      "test",
				Pattern: `secret123`,
			},
		},
	}

	s, err := New(cfg)
	require.NoError(t, err)

	content := "my secret123 value"
	result := s.Scrub(content)

	assert.True(t, result.HasFindings())
	assert.Contains(t, result.Scrubbed, "***HIDDEN***")
	assert.NotContains(t, result.Scrubbed, "secret123")
}

func TestNoopScrubber(t *testing.T) {
	s := &NoopScrubber{}

	assert.False(t, s.IsEnabled())

	content := "api_key: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"

	t.Run("Scrub returns unchanged", func(t *testing.T) {
		result := s.Scrub(content)
		assert.Equal(t, content, result.Scrubbed)
		assert.False(t, result.HasFindings())
	})

	t.Run("ScrubBytes returns unchanged", func(t *testing.T) {
		result := s.ScrubBytes([]byte(content))
		assert.Equal(t, content, result.Scrubbed)
	})

	t.Run("Check returns unchanged", func(t *testing.T) {
		result := s.Check(content)
		assert.Equal(t, content, result.Scrubbed)
	})
}

func TestResult_Methods(t *testing.T) {
	result := &Result{
		TotalFindings: 3,
		Findings: []Finding{
			{RuleID: "rule1", Severity: "high"},
			{RuleID: "rule2", Severity: "medium"},
			{RuleID: "rule3", Severity: "high"},
		},
		ByRule: map[string]int{
			"rule1": 1,
			"rule2": 1,
			"rule3": 1,
		},
	}

	t.Run("HasFindings", func(t *testing.T) {
		assert.True(t, result.HasFindings())
		assert.False(t, (&Result{}).HasFindings())
	})

	t.Run("FindingsBySeverity", func(t *testing.T) {
		high := result.FindingsBySeverity("high")
		assert.Len(t, high, 2)

		medium := result.FindingsBySeverity("medium")
		assert.Len(t, medium, 1)

		low := result.FindingsBySeverity("low")
		assert.Len(t, low, 0)
	})

	t.Run("RuleIDs", func(t *testing.T) {
		ids := result.RuleIDs()
		assert.Len(t, ids, 3)
	})

	t.Run("Summary", func(t *testing.T) {
		assert.Contains(t, result.Summary(), "high severity")

		noFindings := &Result{}
		assert.Equal(t, "no secrets detected", noFindings.Summary())

		mediumOnly := &Result{
			TotalFindings: 1,
			Findings: []Finding{
				{Severity: "medium"},
			},
		}
		assert.Contains(t, mediumOnly.Summary(), "medium severity")

		lowOnly := &Result{
			TotalFindings: 1,
			Findings: []Finding{
				{Severity: "low"},
			},
		}
		assert.Contains(t, lowOnly.Summary(), "low severity")

		noSeverity := &Result{
			TotalFindings: 1,
			Findings: []Finding{
				{Severity: ""},
			},
		}
		assert.Equal(t, "secrets redacted", noSeverity.Summary())
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "[REDACTED]", cfg.RedactionString)
	assert.NotEmpty(t, cfg.Rules)
}

func TestDefaultRules(t *testing.T) {
	rules := DefaultRules()

	assert.NotEmpty(t, rules)

	// Check that all rules have required fields
	for _, rule := range rules {
		assert.NotEmpty(t, rule.ID, "rule must have ID")
		assert.NotEmpty(t, rule.Pattern, "rule %s must have pattern", rule.ID)
		assert.NotEmpty(t, rule.Description, "rule %s must have description", rule.ID)
	}

	// Check for specific rules
	ruleIDs := make(map[string]bool)
	for _, rule := range rules {
		ruleIDs[rule.ID] = true
	}

	expectedRules := []string{
		"aws-access-key-id",
		"github-token",
		"private-key",
		"generic-api-key",
		"jwt",
		"stripe-key",
		"slack-token",
	}

	for _, expected := range expectedRules {
		assert.True(t, ruleIDs[expected], "expected rule %s to be present", expected)
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Run("disabled config skips validation", func(t *testing.T) {
		cfg := &Config{
			Enabled: false,
			Rules: []Rule{
				{ID: "bad", Pattern: `[invalid`},
			},
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("sets default redaction string", func(t *testing.T) {
		cfg := &Config{
			Enabled:         true,
			RedactionString: "",
			Rules: []Rule{
				{ID: "test", Pattern: `test`},
			},
		}
		err := cfg.Validate()
		assert.NoError(t, err)
		assert.Equal(t, "[REDACTED]", cfg.RedactionString)
	})

	t.Run("invalid keyword pattern", func(t *testing.T) {
		cfg := &Config{
			Enabled: true,
			Rules: []Rule{
				{
					ID:       "test",
					Pattern:  `test`,
					Keywords: []string{"valid"},
				},
			},
		}
		// Keywords are quoted, so they should always be valid
		err := cfg.Validate()
		assert.NoError(t, err)
	})
}

func TestScrubber_Performance(t *testing.T) {
	s, err := New(nil)
	require.NoError(t, err)

	// Generate 1KB of content
	content := strings.Repeat("This is some test content with api_key=secret123 inside. ", 20)

	result := s.Scrub(content)

	// Should complete in reasonable time (< 100ms for 1KB)
	assert.Less(t, result.Duration.Milliseconds(), int64(100))
}

func TestScrubber_RealWorldSecrets(t *testing.T) {
	s, err := New(nil)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		content string
		expect  bool
	}{
		{
			name:    "AWS key in config",
			content: `aws_access_key_id = "AKIAIOSFODNN7EXAMPLE"`,
			expect:  true,
		},
		{
			name:    "GitHub token in env",
			content: `export GITHUB_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij`,
			expect:  true,
		},
		{
			name:    "Private key file",
			content: `-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAK...\n-----END RSA PRIVATE KEY-----`,
			expect:  true,
		},
		{
			name:    "Database URL",
			content: `postgres://admin:p4ssw0rd@db.example.com:5432/production`,
			expect:  true,
		},
		{
			name:    "JWT in header",
			content: `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIn0.rTCH8cLoGxAm_xw68z-zXVKi9ie6xJn9tnVWjd_9ftE`,
			expect:  true,
		},
		{
			name:    "Stripe live key",
			content: `STRIPE_KEY=sk_live_abcdefghijklmnopqrstuvwx`,
			expect:  true,
		},
		{
			name:    "OpenAI key",
			content: `OPENAI_API_KEY=sk-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRS`,
			expect:  true,
		},
		{
			name:    "Clean code",
			content: `func main() { fmt.Println("Hello, World!") }`,
			expect:  false,
		},
		{
			name:    "API documentation with real key",
			content: `Use the API_KEY header to authenticate. Example: api_key=abc123def456xyz789`,
			expect:  true, // generic-api-key will match (19+ chars)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := s.Scrub(tc.content)
			if tc.expect {
				assert.True(t, result.HasFindings(), "expected findings for: %s", tc.name)
			} else {
				assert.False(t, result.HasFindings(), "expected no findings for: %s", tc.name)
			}
		})
	}
}
