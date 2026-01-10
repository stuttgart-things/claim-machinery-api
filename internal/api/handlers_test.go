package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	// Create server
	server, err := NewServer("internal/claimtemplate/testdata")
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Handle request
	server.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestListTemplates(t *testing.T) {
	// Create server
	server, err := NewServer("internal/claimtemplate/testdata")
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/claim-templates", nil)
	w := httptest.NewRecorder()

	// Handle request
	server.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var resp ClaimTemplateListResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, "api.claim-machinery.io/v1alpha1", resp.APIVersion)
	assert.Equal(t, "ClaimTemplateList", resp.Kind)
	assert.NotEmpty(t, resp.Items)
}

func TestGetTemplate(t *testing.T) {
	// Create server
	server, err := NewServer("internal/claimtemplate/testdata")
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/claim-templates/volumeclaim", nil)
	w := httptest.NewRecorder()

	// Handle request
	server.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var resp map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Contains(t, resp, "metadata")
	assert.Contains(t, resp, "spec")
}

func TestGetTemplate_NotFound(t *testing.T) {
	// Create server
	server, err := NewServer("internal/claimtemplate/testdata")
	require.NoError(t, err)

	// Create request for non-existent template
	req := httptest.NewRequest(http.MethodGet, "/api/v1/claim-templates/nonexistent", nil)
	w := httptest.NewRecorder()

	// Handle request
	server.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOrderClaim(t *testing.T) {
	// Create server
	server, err := NewServer("internal/claimtemplate/testdata")
	require.NoError(t, err)

	// Create request body
	reqBody := OrderRequest{
		Parameters: map[string]interface{}{
			"namespace": "test-namespace",
		},
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/claim-templates/volumeclaim/order",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Handle request
	server.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var resp OrderResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, "api.claim-machinery.io/v1alpha1", resp.APIVersion)
	assert.Equal(t, "OrderResponse", resp.Kind)
	assert.NotEmpty(t, resp.Rendered)
}

func TestOrderClaim_NotFound(t *testing.T) {
	// Create server
	server, err := NewServer("internal/claimtemplate/testdata")
	require.NoError(t, err)

	// Create request body
	reqBody := OrderRequest{Parameters: map[string]interface{}{}}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	// Create request for non-existent template
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/claim-templates/nonexistent/order",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Handle request
	server.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOrderClaim_InvalidBody(t *testing.T) {
	// Create server
	server, err := NewServer("internal/claimtemplate/testdata")
	require.NoError(t, err)

	// Create request with invalid JSON
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/claim-templates/volumeclaim/order",
		bytes.NewReader([]byte("invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Handle request
	server.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
