# Authentication Workflows

**Parent**: [../SPEC.md](../SPEC.md)

## Usage Examples

> **Note**: These examples describe Unix socket transport. MVP uses HTTP transport on port 8080 with no authentication.

### Example 1: Basic Server Setup

```go
package main

import (
    "context"
    "net"
    "net/http"
    "os"

    "github.com/labstack/echo/v4"
    "github.com/axyzlabs/contextd/pkg/auth"
)

func main() {
    // Setup paths
    socketPath := "/tmp/api.sock"
    tokenPath := "/tmp/token"

    // Ensure auth token exists
    if err := auth.LoadOrGenerateToken(tokenPath); err != nil {
        panic(err)
    }

    // Create Echo server
    e := echo.New()

    // Public routes (no auth)
    e.GET("/health", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
    })

    // Protected routes (auth required)
    api := e.Group("/api/v1")
    api.Use(auth.TokenAuth(tokenPath))

    api.GET("/data", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{"data": "secret"})
    })

    // Create Unix socket
    listener, err := net.Listen("unix", socketPath)
    if err != nil {
        panic(err)
    }
    defer os.Remove(socketPath)

    // Set socket permissions
    if err := os.Chmod(socketPath, 0600); err != nil {
        panic(err)
    }

    // Start server
    e.Listener = listener
    e.Start("")
}
```

### Example 2: Client Authentication

```go
package main

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "os"
)

func main() {
    // Read token from file
    token, err := os.ReadFile("/tmp/token")
    if err != nil {
        panic(err)
    }

    // Create HTTP client with Unix socket transport
    client := &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
                return net.Dial("unix", "/tmp/api.sock")
            },
        },
    }

    // Create request with Bearer token
    req, err := http.NewRequest("GET", "http://localhost/api/v1/data", nil)
    if err != nil {
        panic(err)
    }
    req.Header.Set("Authorization", "Bearer "+string(token))

    // Make request
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    // Check response
    if resp.StatusCode != http.StatusOK {
        fmt.Printf("Authentication failed: %d\n", resp.StatusCode)
        return
    }

    fmt.Println("Authentication successful!")
}
```

### Example 3: Token Regeneration

```bash
#!/bin/bash

# Stop server
systemctl --user stop contextd

# Remove old token
rm ~/.config/contextd/token

# Start server (will auto-generate new token)
systemctl --user start contextd

# Read new token
NEW_TOKEN=$(cat ~/.config/contextd/token)
echo "New token: $NEW_TOKEN"
```

### Example 4: ctxd CLI Integration

```go
package main

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "os"
    "path/filepath"
)

func main() {
    // Get token path
    home := os.Getenv("HOME")
    tokenPath := filepath.Join(home, ".config", "contextd", "token")
    socketPath := filepath.Join(home, ".config", "contextd", "api.sock")

    // Read token
    token, err := os.ReadFile(tokenPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error reading token: %v\n", err)
        os.Exit(1)
    }

    // Create client
    client := &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
                return net.Dial("unix", socketPath)
            },
        },
    }

    // Make authenticated request
    req, _ := http.NewRequest("GET", "http://localhost/api/v1/checkpoints", nil)
    req.Header.Set("Authorization", "Bearer "+string(token))

    resp, err := client.Do(req)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
        os.Exit(1)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusUnauthorized {
        fmt.Fprintf(os.Stderr, "Authentication failed. Token may be invalid.\n")
        os.Exit(1)
    }

    fmt.Println("Request successful!")
}
```

### Example 5: Testing Authentication

```go
package main_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/labstack/echo/v4"
    "github.com/axyzlabs/contextd/pkg/auth"
    "github.com/stretchr/testify/assert"
)

func TestAuthentication(t *testing.T) {
    // Generate token
    tokenPath := "/tmp/test-token"
    if err := auth.GenerateToken(tokenPath); err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tokenPath)

    // Read generated token
    token, _ := os.ReadFile(tokenPath)

    // Create Echo server
    e := echo.New()
    api := e.Group("/api")
    api.Use(auth.TokenAuth(tokenPath))

    api.GET("/test", func(c echo.Context) error {
        return c.String(http.StatusOK, "authenticated")
    })

    // Test valid token
    req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
    req.Header.Set("Authorization", "Bearer "+string(token))
    rec := httptest.NewRecorder()
    e.ServeHTTP(rec, req)
    assert.Equal(t, http.StatusOK, rec.Code)

    // Test invalid token
    req = httptest.NewRequest(http.MethodGet, "/api/test", nil)
    req.Header.Set("Authorization", "Bearer invalid")
    rec = httptest.NewRecorder()
    e.ServeHTTP(rec, req)
    assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
```

## Authentication Flow Diagrams

### Successful Authentication Flow

```
┌──────────┐                                    ┌──────────┐
│  Client  │                                    │  Server  │
└────┬─────┘                                    └────┬─────┘
     │                                               │
     │  GET /api/v1/checkpoints                     │
     │  Authorization: Bearer <token>               │
     │──────────────────────────────────────────────▶
     │                                               │
     │                  ┌───────────────────────┐   │
     │                  │ TokenAuth Middleware  │   │
     │                  │                       │   │
     │                  │ 1. Extract token      │   │
     │                  │ 2. Validate format    │   │
     │                  │ 3. Constant-time      │   │
     │                  │    compare            │   │
     │                  │ 4. Accept or reject   │   │
     │                  └───────────────────────┘   │
     │                                               │
     │◀──────────────────────────────────────────────│
     │  200 OK or 401 Unauthorized                   │
     │                                               │
```

### Middleware Behavior

#### Successful Authentication
```
Request → TokenAuth → Validates token → Calls next() → Handler → Response
```

#### Failed Authentication
```
Request → TokenAuth → Rejects token → Returns 401 → No handler called
```

## File Operations

### Read Token
```bash
# Read token from file
TOKEN=$(cat ~/.config/contextd/token)
```

### Regenerate Token
```bash
# Delete old token
rm ~/.config/contextd/token

# Restart contextd (will auto-generate new token)
systemctl --user restart contextd
```

### Verify Permissions
```bash
# Check token file permissions
stat -c "%a %n" ~/.config/contextd/token
# Output: 600 /home/user/.config/contextd/token
```

## Error Handling Patterns

### Initialization

```go
// Load or generate token during startup
if err := auth.LoadOrGenerateToken(tokenPath); err != nil {
    // Critical error - cannot start server
    log.Fatal("Failed to setup auth token:", err)
}
```

### Middleware Setup

```go
// TokenAuth panics on invalid token - use defer/recover if needed
defer func() {
    if r := recover(); r != nil {
        log.Fatal("TokenAuth initialization failed:", r)
    }
}()

api.Use(auth.TokenAuth(tokenPath))
```

### Client-Side Error Handling

```go
resp, err := client.Get(url, headers)
if err != nil {
    return err
}

if resp.StatusCode == 401 {
    // Token invalid - may need to regenerate
    return errors.New("authentication failed: invalid token")
}
```
