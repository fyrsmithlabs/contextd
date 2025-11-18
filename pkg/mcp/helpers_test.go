package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractOwnerID tests owner ID extraction from authenticated context.
func TestExtractOwnerID(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    func(*echo.Context)
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid owner ID from authenticated context",
			setupCtx: func(c *echo.Context) {
				(*c).Set(string(authenticatedOwnerIDKey), "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678")
			},
			want:    "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678",
			wantErr: false,
		},
		{
			name: "missing owner ID - unauthenticated request",
			setupCtx: func(c *echo.Context) {
				// No owner ID set
			},
			want:        "",
			wantErr:     true,
			errContains: "unauthenticated",
		},
		{
			name: "invalid owner ID format - not hex",
			setupCtx: func(c *echo.Context) {
				(*c).Set(string(authenticatedOwnerIDKey), "not-hex-string!-invalid")
			},
			want:        "",
			wantErr:     true,
			errContains: "invalid owner ID format",
		},
		{
			name: "invalid owner ID format - wrong length",
			setupCtx: func(c *echo.Context) {
				(*c).Set(string(authenticatedOwnerIDKey), "abc123") // Too short
			},
			want:        "",
			wantErr:     true,
			errContains: "invalid owner ID format",
		},
		{
			name: "ignores user-controlled X-Owner-ID header",
			setupCtx: func(c *echo.Context) {
				// User tries to inject owner ID via header
				(*c).Request().Header.Set("X-Owner-ID", "malicious-owner-id")
				// But authenticated context has the real owner ID
				(*c).Set(string(authenticatedOwnerIDKey), "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678")
			},
			want:    "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Setup test context
			tt.setupCtx(&c)

			// Test extraction
			got, err := ExtractOwnerID(c)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestJSONRPCSuccess tests successful JSON-RPC response helper.
func TestJSONRPCSuccess(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	result := map[string]string{"operation_id": "op-123"}
	err := JSONRPCSuccess(c, "req-456", result)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"jsonrpc":"2.0"`)
	assert.Contains(t, rec.Body.String(), `"id":"req-456"`)
	assert.Contains(t, rec.Body.String(), `"operation_id":"op-123"`)
}

// TestJSONRPCErrorWithContext tests error response helper with trace context.
func TestJSONRPCErrorWithContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Request-ID", "trace-789")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	testErr := ErrInvalidParams
	err := JSONRPCErrorWithContext(c, "req-456", InvalidParams, testErr)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"jsonrpc":"2.0"`)
	assert.Contains(t, rec.Body.String(), `"id":"req-456"`)
	assert.Contains(t, rec.Body.String(), `"code":-32602`)
	assert.Contains(t, rec.Body.String(), `"trace-789"`)
}

// TestParseCollectionURI tests collection URI parsing.
func TestParseCollectionURI(t *testing.T) {
	tests := []struct {
		name                string
		uri                 string
		wantOwnerID         string
		wantCollectionName  string
		wantErr             bool
		errContains         string
	}{
		{
			name:                "valid collection URI",
			uri:                 "collection://owner_a1b2c3d4/project_def456/main",
			wantOwnerID:         "a1b2c3d4",
			wantCollectionName:  "owner_a1b2c3d4/project_def456/main",
			wantErr:             false,
		},
		{
			name:                "valid collection URI with 64-char owner ID",
			uri:                 "collection://owner_a1b2c3d4e5f67890123456789012345678901234567890123456789012345678/project_abc/feature-branch",
			wantOwnerID:         "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678",
			wantCollectionName:  "owner_a1b2c3d4e5f67890123456789012345678901234567890123456789012345678/project_abc/feature-branch",
			wantErr:             false,
		},
		{
			name:        "invalid scheme - http",
			uri:         "http://owner_abc/project_def/main",
			wantErr:     true,
			errContains: "invalid URI scheme",
		},
		{
			name:        "invalid scheme - missing scheme",
			uri:         "owner_abc/project_def/main",
			wantErr:     true,
			errContains: "invalid URI scheme",
		},
		{
			name:        "empty collection name",
			uri:         "collection://",
			wantErr:     true,
			errContains: "empty collection name",
		},
		{
			name:        "missing owner prefix",
			uri:         "collection://project_def/main",
			wantErr:     true,
			errContains: "invalid owner prefix",
		},
		{
			name:        "empty owner ID",
			uri:         "collection://owner_/project_def/main",
			wantErr:     true,
			errContains: "empty owner ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ownerID, collectionName, err := ParseCollectionURI(tt.uri)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOwnerID, ownerID)
			assert.Equal(t, tt.wantCollectionName, collectionName)
		})
	}
}
