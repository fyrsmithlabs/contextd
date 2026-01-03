# GitHub API Retry Logic

This document describes the retry backoff configuration for GitHub API calls in Temporal workflows.

## Overview

All GitHub API calls in workflow activities now include automatic retry logic with exponential backoff to handle:
- Rate limiting (429, 403 with rate info)
- Transient errors (500, 502, 503, 504)
- Network failures

## Configuration

### RetryConfig

```go
type RetryConfig struct {
    MaxRetries        int           // Maximum number of retry attempts (default: 3)
    InitialBackoff    time.Duration // Initial backoff duration (default: 1 second)
    MaxBackoff        time.Duration // Maximum backoff duration (default: 30 seconds)
    BackoffMultiplier float64       // Multiplier for exponential backoff (default: 2.0)
}
```

### Default Configuration

```go
config := DefaultRetryConfig()
// MaxRetries:        3
// InitialBackoff:    1 second
// MaxBackoff:        30 seconds
// BackoffMultiplier: 2.0
```

## Retry Behavior

### Retryable Errors

| Status Code | Description | Retryable |
|-------------|-------------|-----------|
| 429 | Too Many Requests | Yes |
| 500 | Internal Server Error | Yes |
| 502 | Bad Gateway | Yes |
| 503 | Service Unavailable | Yes |
| 504 | Gateway Timeout | Yes |
| 403 | Forbidden (with rate info) | Yes |
| 400 | Bad Request | No |
| 401 | Unauthorized | No |
| 403 | Forbidden (without rate info) | No |
| 404 | Not Found | No |
| 422 | Unprocessable Entity | No |

### Exponential Backoff

The retry logic uses exponential backoff with the following progression:

```
Attempt 1: Initial request
Attempt 2: Wait InitialBackoff (1s), then retry
Attempt 3: Wait InitialBackoff * 2 (2s), then retry
Attempt 4: Wait InitialBackoff * 4 (4s), then retry (up to MaxBackoff)
```

### Rate Limit Handling

When a rate limit error is detected (429 or 403 with rate info):

1. Extract the `X-RateLimit-Reset` timestamp from the response
2. Calculate time until reset + 1 second buffer
3. Use this as the backoff duration (capped at `MaxBackoff`)
4. Log rate limit information for monitoring

## Usage in Activities

All activities in `version_validation_activities.go` use retry logic:

```go
func FetchFileContentActivity(ctx context.Context, input FetchFileContentInput) (string, error) {
    client, err := NewGitHubClient(ctx, input.GitHubToken)
    if err != nil {
        return "", err
    }

    // Fetch file content with retry logic
    var fileContent *github.RepositoryContent
    retryConfig := DefaultRetryConfig()
    _, err = retryGitHubOperation(ctx, retryConfig, func() (*github.Response, error) {
        var ghResp *github.Response
        var ghErr error
        fileContent, _, ghResp, ghErr = client.Repositories.GetContents(
            ctx, input.Owner, input.Repo, input.Path,
            &github.RepositoryContentGetOptions{Ref: input.Ref},
        )
        return ghResp, ghErr
    })
    if err != nil {
        return "", fmt.Errorf("failed to get file content: %w", err)
    }

    content, err := fileContent.GetContent()
    return content, err
}
```

## Logging

The retry logic provides detailed logging for monitoring and debugging:

### Success After Retries
```
INFO GitHub API operation recovered after retries
  attempts=2
  total_time=3.5s
```

### Rate Limit Hit
```
INFO GitHub API rate limit hit, adjusting backoff
  attempt=2
  max_attempts=4
  backoff=15s
```

### Retrying Transient Error
```
INFO Retrying GitHub API operation after transient error
  attempt=2
  max_attempts=4
  error="service unavailable"
  status_code=503
  backoff=2s
```

### Final Failure
```
WARN GitHub API operation failed after all retries exhausted
  total_attempts=4
  total_time=7.5s
  error="service unavailable"
  status_code=503
```

## Testing

Comprehensive tests cover:
- Configuration defaults and overrides
- Success scenarios (first attempt, after retries)
- Non-retryable error handling
- Retry exhaustion
- Context cancellation
- Exponential backoff timing
- Max backoff capping
- Rate limit detection and backoff calculation

Run tests:
```bash
go test -v ./internal/workflows/... -run TestRetry
```

## Files Modified

| File | Changes |
|------|---------|
| `github_retry.go` | New file with retry configuration and logic |
| `github_retry_test.go` | Comprehensive test coverage for retry logic |
| `version_validation_activities.go` | Updated all GitHub API calls to use retry wrapper |
| `types.go` | Added `Validate()` method to `VersionValidationConfig` |

## Benefits

1. **Resilience**: Automatic recovery from transient GitHub API failures
2. **Rate Limit Compliance**: Intelligent backoff based on GitHub's rate limit headers
3. **Observability**: Detailed logging for monitoring and debugging
4. **Configurability**: Easy to adjust retry parameters for different scenarios
5. **Consistency**: Centralized retry logic across all GitHub API operations

## Future Improvements

Consider these enhancements:
1. Make retry configuration configurable via environment variables or config file
2. Add metrics for retry attempts and success rates
3. Implement jitter in backoff to reduce thundering herd effect
4. Add circuit breaker pattern for prolonged failures
