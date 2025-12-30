package workflows

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryConfig_ApplyDefaults(t *testing.T) {
	t.Run("applies all defaults when empty", func(t *testing.T) {
		config := &RetryConfig{}
		config.ApplyDefaults()

		assert.Equal(t, 3, config.MaxRetries)
		assert.Equal(t, time.Second, config.InitialBackoff)
		assert.Equal(t, 30*time.Second, config.MaxBackoff)
		assert.Equal(t, 2.0, config.BackoffMultiplier)
	})

	t.Run("preserves non-zero values", func(t *testing.T) {
		config := &RetryConfig{
			MaxRetries:        5,
			InitialBackoff:    2 * time.Second,
			MaxBackoff:        60 * time.Second,
			BackoffMultiplier: 3.0,
		}
		config.ApplyDefaults()

		assert.Equal(t, 5, config.MaxRetries)
		assert.Equal(t, 2*time.Second, config.InitialBackoff)
		assert.Equal(t, 60*time.Second, config.MaxBackoff)
		assert.Equal(t, 3.0, config.BackoffMultiplier)
	})
}

func TestRetryGitHubOperation_Success(t *testing.T) {
	ctx := context.Background()
	config := &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	callCount := 0
	operation := func() (*github.Response, error) {
		callCount++
		return &github.Response{
			Response: &http.Response{StatusCode: 200},
		}, nil
	}

	resp, err := retryGitHubOperation(ctx, config, operation)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.Response.StatusCode)
	assert.Equal(t, 1, callCount, "should succeed on first attempt")
}

func TestRetryGitHubOperation_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	config := &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	callCount := 0
	operation := func() (*github.Response, error) {
		callCount++
		if callCount < 3 {
			// Fail first 2 attempts with retryable error
			return &github.Response{
				Response: &http.Response{StatusCode: 503},
			}, errors.New("service unavailable")
		}
		// Succeed on 3rd attempt
		return &github.Response{
			Response: &http.Response{StatusCode: 200},
		}, nil
	}

	start := time.Now()
	resp, err := retryGitHubOperation(ctx, config, operation)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.Response.StatusCode)
	assert.Equal(t, 3, callCount, "should succeed on 3rd attempt")

	// Should have waited at least 10ms + 20ms = 30ms (backoff times)
	assert.GreaterOrEqual(t, duration, 30*time.Millisecond)
}

func TestRetryGitHubOperation_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	config := &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	callCount := 0
	operation := func() (*github.Response, error) {
		callCount++
		// Return non-retryable error (404)
		return &github.Response{
			Response: &http.Response{StatusCode: 404},
		}, errors.New("not found")
	}

	resp, err := retryGitHubOperation(ctx, config, operation)

	require.Error(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 404, resp.Response.StatusCode)
	assert.Equal(t, 1, callCount, "should not retry non-retryable errors")
}

func TestRetryGitHubOperation_ExhaustsRetries(t *testing.T) {
	ctx := context.Background()
	config := &RetryConfig{
		MaxRetries:        2,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	callCount := 0
	operation := func() (*github.Response, error) {
		callCount++
		// Always return retryable error
		return &github.Response{
			Response: &http.Response{StatusCode: 503},
		}, errors.New("service unavailable")
	}

	resp, err := retryGitHubOperation(ctx, config, operation)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 2 retries")
	assert.NotNil(t, resp)
	assert.Equal(t, 503, resp.Response.StatusCode)
	assert.Equal(t, 3, callCount, "should try once + 2 retries = 3 total")
}

func TestRetryGitHubOperation_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	callCount := 0
	operation := func() (*github.Response, error) {
		callCount++
		if callCount == 1 {
			// Cancel context during first retry
			cancel()
		}
		return &github.Response{
			Response: &http.Response{StatusCode: 503},
		}, errors.New("service unavailable")
	}

	resp, err := retryGitHubOperation(ctx, config, operation)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation canceled")
	assert.Nil(t, resp)
	assert.Equal(t, 1, callCount, "should stop after context canceled")
}

func TestRetryGitHubOperation_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()
	config := &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	callCount := 0
	callTimes := []time.Time{}

	operation := func() (*github.Response, error) {
		callCount++
		callTimes = append(callTimes, time.Now())
		if callCount < 4 {
			return &github.Response{
				Response: &http.Response{StatusCode: 503},
			}, errors.New("service unavailable")
		}
		return &github.Response{
			Response: &http.Response{StatusCode: 200},
		}, nil
	}

	_, err := retryGitHubOperation(ctx, config, operation)
	require.NoError(t, err)

	// Verify exponential backoff: 10ms, 20ms, 40ms
	assert.Len(t, callTimes, 4)

	// Check time between attempts
	for i := 1; i < len(callTimes); i++ {
		// Calculate expected backoff: InitialBackoff * (2^(i-1))
		multiplier := 1
		for j := 0; j < i-1; j++ {
			multiplier *= 2
		}
		expectedBackoff := time.Duration(int64(config.InitialBackoff) * int64(multiplier))
		actualBackoff := callTimes[i].Sub(callTimes[i-1])

		// Allow some tolerance (±5ms) for timing variations
		assert.InDelta(t, float64(expectedBackoff), float64(actualBackoff), float64(5*time.Millisecond),
			"backoff between attempt %d and %d should be ~%v", i-1, i, expectedBackoff)
	}
}

func TestRetryGitHubOperation_MaxBackoffCap(t *testing.T) {
	ctx := context.Background()
	config := &RetryConfig{
		MaxRetries:        5,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        30 * time.Millisecond, // Cap at 30ms
		BackoffMultiplier: 2.0,
	}

	callCount := 0
	callTimes := []time.Time{}

	operation := func() (*github.Response, error) {
		callCount++
		callTimes = append(callTimes, time.Now())
		if callCount < 6 {
			return &github.Response{
				Response: &http.Response{StatusCode: 503},
			}, errors.New("service unavailable")
		}
		return &github.Response{
			Response: &http.Response{StatusCode: 200},
		}, nil
	}

	_, err := retryGitHubOperation(ctx, config, operation)
	require.NoError(t, err)

	// Backoffs should be: 10ms, 20ms, 30ms (capped), 30ms (capped), 30ms (capped)
	assert.Len(t, callTimes, 6)

	// Check that later backoffs are capped at MaxBackoff
	for i := 3; i < len(callTimes); i++ {
		actualBackoff := callTimes[i].Sub(callTimes[i-1])
		// Should be approximately MaxBackoff
		assert.InDelta(t, float64(config.MaxBackoff), float64(actualBackoff), float64(5*time.Millisecond),
			"backoff should be capped at MaxBackoff for attempt %d", i)
	}
}

func TestIsGitHubRetryableError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		hasRate    bool
		want       bool
	}{
		{
			name:       "nil error",
			err:        nil,
			statusCode: 200,
			want:       false,
		},
		{
			name:       "429 rate limit",
			err:        errors.New("rate limit exceeded"),
			statusCode: 429,
			want:       true,
		},
		{
			name:       "500 internal server error",
			err:        errors.New("internal error"),
			statusCode: 500,
			want:       true,
		},
		{
			name:       "502 bad gateway",
			err:        errors.New("bad gateway"),
			statusCode: 502,
			want:       true,
		},
		{
			name:       "503 service unavailable",
			err:        errors.New("service unavailable"),
			statusCode: 503,
			want:       true,
		},
		{
			name:       "504 gateway timeout",
			err:        errors.New("gateway timeout"),
			statusCode: 504,
			want:       true,
		},
		{
			name:       "400 bad request",
			err:        errors.New("bad request"),
			statusCode: 400,
			want:       false,
		},
		{
			name:       "401 unauthorized",
			err:        errors.New("unauthorized"),
			statusCode: 401,
			want:       false,
		},
		{
			name:       "403 forbidden without rate info",
			err:        errors.New("forbidden"),
			statusCode: 403,
			want:       false,
		},
		{
			name:       "403 forbidden with rate info",
			err:        errors.New("forbidden"),
			statusCode: 403,
			hasRate:    true,
			want:       true,
		},
		{
			name:       "404 not found",
			err:        errors.New("not found"),
			statusCode: 404,
			want:       false,
		},
		{
			name:       "422 unprocessable entity",
			err:        errors.New("validation failed"),
			statusCode: 422,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *github.Response
			if tt.statusCode > 0 {
				resp = &github.Response{
					Response: &http.Response{StatusCode: tt.statusCode},
				}
				if tt.hasRate {
					resp.Rate = github.Rate{Limit: 5000, Remaining: 0}
				}
			}

			got := isGitHubRetryableError(tt.err, resp)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		hasRate    bool
		want       bool
	}{
		{
			name:       "nil response",
			statusCode: 0,
			want:       false,
		},
		{
			name:       "429 status",
			statusCode: 429,
			want:       true,
		},
		{
			name:       "403 with rate info",
			statusCode: 403,
			hasRate:    true,
			want:       true,
		},
		{
			name:       "403 without rate info",
			statusCode: 403,
			want:       false,
		},
		{
			name:       "200 success",
			statusCode: 200,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *github.Response
			if tt.statusCode > 0 {
				resp = &github.Response{
					Response: &http.Response{StatusCode: tt.statusCode},
				}
				if tt.hasRate {
					resp.Rate = github.Rate{Limit: 5000, Remaining: 0}
				}
			}

			got := isRateLimitError(resp)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetRateLimitBackoff(t *testing.T) {
	tests := []struct {
		name       string
		resetTime  time.Time
		maxBackoff time.Duration
		wantMin    time.Duration
		wantMax    time.Duration
	}{
		{
			name:       "reset in 5 seconds",
			resetTime:  time.Now().Add(5 * time.Second),
			maxBackoff: 30 * time.Second,
			wantMin:    5 * time.Second,
			wantMax:    7 * time.Second, // 5s + 1s buffer + timing tolerance
		},
		{
			name:       "reset in past (should return 1 second)",
			resetTime:  time.Now().Add(-5 * time.Second),
			maxBackoff: 30 * time.Second,
			wantMin:    time.Second,
			wantMax:    2 * time.Second,
		},
		{
			name:       "reset beyond max backoff",
			resetTime:  time.Now().Add(60 * time.Second),
			maxBackoff: 30 * time.Second,
			wantMin:    30 * time.Second,
			wantMax:    30 * time.Second,
		},
		{
			name:       "nil rate info",
			resetTime:  time.Time{},
			maxBackoff: 30 * time.Second,
			wantMin:    time.Minute,
			wantMax:    time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *github.Response
			if !tt.resetTime.IsZero() {
				resp = &github.Response{
					Rate: github.Rate{
						Reset: github.Timestamp{Time: tt.resetTime},
						Limit: 5000,
					},
				}
			}

			backoff := getRateLimitBackoff(resp, tt.maxBackoff)

			assert.GreaterOrEqual(t, backoff, tt.wantMin)
			assert.LessOrEqual(t, backoff, tt.wantMax)
		})
	}
}

func TestGetStatusCode(t *testing.T) {
	tests := []struct {
		name string
		resp *github.Response
		want int
	}{
		{
			name: "valid response",
			resp: &github.Response{
				Response: &http.Response{StatusCode: 200},
			},
			want: 200,
		},
		{
			name: "nil response",
			resp: nil,
			want: 0,
		},
		{
			name: "response with nil http.Response",
			resp: &github.Response{Response: nil},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStatusCode(tt.resp)
			assert.Equal(t, tt.want, got)
		})
	}
}
