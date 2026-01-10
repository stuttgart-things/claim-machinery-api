package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestDataPath returns the correct path to testdata directory
func getTestDataPath() string {
	// Try multiple possible locations
	paths := []string{
		"internal/claimtemplate/testdata",
		"./internal/claimtemplate/testdata",
		"../claimtemplate/testdata",
	}

	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}

	// Fallback - try to find it relative to source file
	_, file, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Dir(file)
		testdataPath := filepath.Join(filepath.Dir(dir), "claimtemplate", "testdata")
		if info, err := os.Stat(testdataPath); err == nil && info.IsDir() {
			return testdataPath
		}
	}

	// Last resort - return a relative path that works from repo root
	return "internal/claimtemplate/testdata"
}

func TestHealthCheck(t *testing.T) {
	// Create server
	testdata := getTestDataPath()
	server, err := NewServer(testdata)
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
	testdata := getTestDataPath()
	server, err := NewServer(testdata)
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
	assert.Greater(t, len(resp.Items), 0)
}

func TestGetTemplate(t *testing.T) {
	// Create server
	testdata := getTestDataPath()
	server, err := NewServer(testdata)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/claim-templates/volumeclaim", nil)
	w := httptest.NewRecorder()

	// Handle request
	server.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var resp interface{}
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
}

func TestGetTemplate_NotFound(t *testing.T) {
	// Create server
	testdata := getTestDataPath()
	server, err := NewServer(testdata)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/claim-templates/nonexistent", nil)
	w := httptest.NewRecorder()

	// Handle request
	server.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOrderClaim(t *testing.T) {
	// Create server
	testdata := getTestDataPath()
	server, err := NewServer(testdata)
	require.NoError(t, err)

	// Create request body
	body := bytes.NewBufferString(`{"parameters": {"namespace": "test", "storage": "10Gi"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/claim-templates/volumeclaim/order", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Handle request
	server.router.ServeHTTP(w, req)

	// Should get 200 OK if KCL is available, or 500 if KCL CLI is not found
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

func TestOrderClaim_NotFound(t *testing.T) {
	// Create server
	testdata := getTestDataPath()
	server, err := NewServer(testdata)
	require.NoError(t, err)

	// Create request
	body := bytes.NewBufferString(`{"parameters": {"namespace": "test"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/claim-templates/nonexistent/order", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Handle request
	server.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOrderClaim_InvalidBody(t *testing.T) {
	// Create server
	testdata := getTestDataPath()
	server, err := NewServer(testdata)
	require.NoError(t, err)

	// Create request with invalid JSON
	body := bytes.NewBufferString(`{invalid json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/claim-templates/volumeclaim/order", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Handle request
	server.router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
