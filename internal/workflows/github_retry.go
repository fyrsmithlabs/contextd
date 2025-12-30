package workflows

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v57/github"
	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

// RetryConfig configures retry behavior for GitHub API calls.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts.
	// Default: 3
	MaxRetries int

	// InitialBackoff is the initial backoff duration.
	// Default: 1 second
	InitialBackoff time.Duration

	// MaxBackoff is the maximum backoff duration.
	// Default: 30 seconds
	MaxBackoff time.Duration

	// BackoffMultiplier is the multiplier for exponential backoff.
	// Default: 2
	BackoffMultiplier float64
}

// DefaultRetryConfig returns the default retry configuration for GitHub API calls.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// ApplyDefaults sets default values for unset fields.
func (c *RetryConfig) ApplyDefaults() {
	defaults := DefaultRetryConfig()

	if c.MaxRetries == 0 {
		c.MaxRetries = defaults.MaxRetries
	}
	if c.InitialBackoff == 0 {
		c.InitialBackoff = defaults.InitialBackoff
	}
	if c.MaxBackoff == 0 {
		c.MaxBackoff = defaults.MaxBackoff
	}
	if c.BackoffMultiplier == 0 {
		c.BackoffMultiplier = defaults.BackoffMultiplier
	}
}

// retryGitHubOperation retries a GitHub API operation with exponential backoff.
// It handles rate limiting and transient errors automatically.
func retryGitHubOperation(ctx context.Context, config *RetryConfig, operation func() (*github.Response, error)) (*github.Response, error) {
	if config == nil {
		config = DefaultRetryConfig()
	}
	config.ApplyDefaults()

	var lastErr error
	var lastResp *github.Response
	backoff := config.InitialBackoff
	startTime := time.Now()

	// Try to get logger from activity context (may not be available in tests)
	// We'll check if we have a logger before using it
	type logger interface {
		Info(string, ...interface{})
		Debug(string, ...interface{})
		Warn(string, ...interface{})
	}

	var log logger
	func() {
		defer func() {
			_ = recover() // Ignore panic if not an activity context
		}()
		log = activity.GetLogger(ctx)
	}()

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		resp, err := operation()
		if err == nil {
			// Log successful recovery after retries
			if attempt > 0 && log != nil {
				log.Info("GitHub API operation recovered after retries",
					zap.Int("attempts", attempt),
					zap.Duration("total_time", time.Since(startTime)),
				)
			}
			return resp, nil
		}

		lastErr = err
		lastResp = resp

		// Check if error is retryable
		if !isGitHubRetryableError(err, resp) {
			if log != nil {
				log.Debug("GitHub API error is not retryable",
					zap.Error(err),
					zap.Int("status_code", getStatusCode(resp)),
				)
			}
			return resp, err
		}

		// Last attempt, return error
		if attempt == config.MaxRetries {
			break
		}

		// Check for rate limit and adjust backoff accordingly
		if isRateLimitError(resp) {
			backoff = getRateLimitBackoff(resp, config.MaxBackoff)
			if log != nil {
				log.Info("GitHub API rate limit hit, adjusting backoff",
					zap.Int("attempt", attempt+1),
					zap.Int("max_attempts", config.MaxRetries+1),
					zap.Duration("backoff", backoff),
				)
			}
		} else {
			// Log retry attempt for other transient errors
			if log != nil {
				log.Info("Retrying GitHub API operation after transient error",
					zap.Int("attempt", attempt+1),
					zap.Int("max_attempts", config.MaxRetries+1),
					zap.Error(err),
					zap.Int("status_code", getStatusCode(resp)),
					zap.Duration("backoff", backoff),
				)
			}
		}

		// Wait before retry (exponential backoff with max cap)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("operation canceled: %w", ctx.Err())
		case <-time.After(backoff):
			// Calculate next backoff
			nextBackoff := time.Duration(float64(backoff) * config.BackoffMultiplier)
			if nextBackoff > config.MaxBackoff {
				nextBackoff = config.MaxBackoff
			}
			backoff = nextBackoff
		}
	}

	// Log final failure
	if log != nil {
		log.Warn("GitHub API operation failed after all retries exhausted",
			zap.Int("total_attempts", config.MaxRetries+1),
			zap.Duration("total_time", time.Since(startTime)),
			zap.Error(lastErr),
			zap.Int("status_code", getStatusCode(lastResp)),
		)
	}

	return lastResp, fmt.Errorf("GitHub API operation failed after %d retries: %w", config.MaxRetries, lastErr)
}

// isGitHubRetryableError checks if a GitHub API error is retryable.
func isGitHubRetryableError(err error, resp *github.Response) bool {
	if err == nil {
		return false
	}

	// Check HTTP status code for retryable errors
	if resp != nil && resp.Response != nil {
		statusCode := resp.Response.StatusCode

		switch statusCode {
		// Rate limiting
		case http.StatusTooManyRequests: // 429
			return true

		// Server errors (retryable)
		case http.StatusInternalServerError: // 500
			return true
		case http.StatusBadGateway: // 502
			return true
		case http.StatusServiceUnavailable: // 503
			return true
		case http.StatusGatewayTimeout: // 504
			return true

		// Client errors (not retryable)
		case http.StatusBadRequest: // 400
			return false
		case http.StatusUnauthorized: // 401
			return false
		case http.StatusForbidden: // 403
			// Forbidden can be rate limit (secondary rate limit)
			// Check if it has rate limit headers (Limit > 0 means we got rate info)
			if resp.Rate.Limit > 0 {
				return true
			}
			return false
		case http.StatusNotFound: // 404
			return false
		case http.StatusUnprocessableEntity: // 422
			return false

		default:
			// For other status codes, check if it's in the 5xx range
			return statusCode >= 500 && statusCode < 600
		}
	}

	// If we can't determine from status code, check error type
	// Network errors, timeouts, etc. are typically retryable
	return true
}

// isRateLimitError checks if the response indicates a rate limit error.
func isRateLimitError(resp *github.Response) bool {
	if resp == nil || resp.Response == nil {
		return false
	}

	// Check for 429 status code
	if resp.Response.StatusCode == http.StatusTooManyRequests {
		return true
	}

	// Check for 403 with rate limit info
	if resp.Response.StatusCode == http.StatusForbidden && resp.Rate.Limit > 0 {
		return true
	}

	return false
}

// getRateLimitBackoff calculates the backoff duration for rate limit errors.
// It respects the GitHub API rate limit reset time if available.
func getRateLimitBackoff(resp *github.Response, maxBackoff time.Duration) time.Duration {
	if resp == nil || (resp.Rate.Limit == 0 && resp.Rate.Remaining == 0) {
		return time.Minute // Default 1 minute if no rate limit info
	}

	// Calculate time until rate limit reset
	now := time.Now()
	resetTime := resp.Rate.Reset.Time
	backoff := resetTime.Sub(now)

	// Add a small buffer (1 second) to ensure reset has happened
	backoff += time.Second

	// Ensure backoff is positive
	if backoff < 0 {
		backoff = time.Second
	}

	// Cap at max backoff
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}

// getStatusCode safely extracts the HTTP status code from a GitHub response.
func getStatusCode(resp *github.Response) int {
	if resp != nil && resp.Response != nil {
		return resp.Response.StatusCode
	}
	return 0
}
