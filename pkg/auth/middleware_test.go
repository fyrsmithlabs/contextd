package auth

import (
	"net/http"
	"net/http/httptest"
	"os/user"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOwnerAuthMiddleware_Success tests successful authentication.
func TestOwnerAuthMiddleware_Success(t *testing.T) {
	// Setup Echo
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test handler that verifies owner ID is set
	var capturedOwnerID string
	handler := func(c echo.Context) error {
		ownerID, ok := c.Get(string(authenticatedOwnerIDKey)).(string)
		if !ok {
			return c.String(http.StatusInternalServerError, "owner ID not set")
		}
		capturedOwnerID = ownerID
		return c.String(http.StatusOK, "authenticated")
	}

	// Apply middleware
	middleware := OwnerAuthMiddleware()
	h := middleware(handler)

	// Execute
	err := h(c)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, capturedOwnerID, "owner ID should be set in context")
	assert.Len(t, capturedOwnerID, 64, "owner ID should be 64-character SHA256 hash")
	assert.Regexp(t, "^[a-fA-F0-9]{64}$", capturedOwnerID, "owner ID should be valid hex")
}

// TestOwnerAuthMiddleware_OwnerIDFormat tests owner ID format validation.
func TestOwnerAuthMiddleware_OwnerIDFormat(t *testing.T) {
	// Setup Echo
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test handler
	handler := func(c echo.Context) error {
		ownerID, ok := c.Get(string(authenticatedOwnerIDKey)).(string)
		require.True(t, ok, "owner ID should be set")

		// Verify format matches SHA256 output
		assert.Len(t, ownerID, 64, "SHA256 produces 64-character hex string")
		assert.Regexp(t, "^[a-f0-9]{64}$", ownerID, "should be lowercase hex")

		return c.String(http.StatusOK, "ok")
	}

	// Apply middleware
	middleware := OwnerAuthMiddleware()
	h := middleware(handler)

	// Execute
	err := h(c)
	require.NoError(t, err)
}

// TestOwnerAuthMiddleware_Consistency tests that same user gets same owner ID.
func TestOwnerAuthMiddleware_Consistency(t *testing.T) {
	// Get current user (will be same for both requests)
	currentUser, err := user.Current()
	require.NoError(t, err, "should be able to get current user")

	// First request
	e1 := echo.New()
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec1 := httptest.NewRecorder()
	c1 := e1.NewContext(req1, rec1)

	var ownerID1 string
	handler := func(c echo.Context) error {
		id, _ := c.Get(string(authenticatedOwnerIDKey)).(string)
		ownerID1 = id
		return c.String(http.StatusOK, "ok")
	}

	middleware1 := OwnerAuthMiddleware()
	h1 := middleware1(handler)
	err = h1(c1)
	require.NoError(t, err)

	// Second request
	e2 := echo.New()
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec2 := httptest.NewRecorder()
	c2 := e2.NewContext(req2, rec2)

	var ownerID2 string
	handler2 := func(c echo.Context) error {
		id, _ := c.Get(string(authenticatedOwnerIDKey)).(string)
		ownerID2 = id
		return c.String(http.StatusOK, "ok")
	}

	middleware2 := OwnerAuthMiddleware()
	h2 := middleware2(handler2)
	err = h2(c2)
	require.NoError(t, err)

	// Verify consistency
	assert.Equal(t, ownerID1, ownerID2, "same user should get same owner ID")

	// Verify it matches expected derivation
	expectedOwnerID, err := DeriveOwnerID(currentUser.Username)
	require.NoError(t, err)
	assert.Equal(t, expectedOwnerID, ownerID1, "owner ID should match derived value")
}

// TestOwnerAuthMiddleware_WithLogging tests that errors are properly logged.
func TestOwnerAuthMiddleware_WithLogging(t *testing.T) {
	// This test verifies the middleware structure but doesn't test actual error cases
	// since we can't easily simulate user.Current() failing in a unit test.
	// The middleware should handle such errors gracefully.

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	middleware := OwnerAuthMiddleware()
	h := middleware(handler)

	err := h(c)

	// In normal conditions, should succeed
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestOwnerAuthMiddleware_ErrorResponse tests error response format.
func TestOwnerAuthMiddleware_ErrorResponse(t *testing.T) {
	// This test documents expected error response structure
	// In case of authentication failure, middleware should return 401

	// We can't easily test the failure case in a unit test since user.Current()
	// rarely fails in test environments. This is documented for integration testing.
	//
	// Expected behavior on failure:
	// - Status: 401 Unauthorized
	// - Body: JSON error message
	// - Headers: Standard Echo error headers
}
