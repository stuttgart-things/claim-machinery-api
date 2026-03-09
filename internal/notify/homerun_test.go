package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTruthy(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"", false},
		{"  true  ", true},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, isTruthy(tt.input), "isTruthy(%q)", tt.input)
	}
}

func TestLoadHomerunConfig(t *testing.T) {
	t.Run("disabled by default", func(t *testing.T) {
		t.Setenv("ENABLE_HOMERUN", "")
		t.Setenv("HOMERUN_URL", "")
		t.Setenv("HOMERUN_AUTH_TOKEN", "")

		cfg := LoadHomerunConfig()
		assert.False(t, cfg.Enabled)
	})

	t.Run("enabled requires both flag and URL", func(t *testing.T) {
		t.Setenv("ENABLE_HOMERUN", "true")
		t.Setenv("HOMERUN_URL", "")

		cfg := LoadHomerunConfig()
		assert.False(t, cfg.Enabled)
	})

	t.Run("enabled with URL", func(t *testing.T) {
		t.Setenv("ENABLE_HOMERUN", "1")
		t.Setenv("HOMERUN_URL", "https://pitcher.example.com")
		t.Setenv("HOMERUN_AUTH_TOKEN", "secret-token")

		cfg := LoadHomerunConfig()
		assert.True(t, cfg.Enabled)
		assert.Equal(t, "https://pitcher.example.com", cfg.URL)
		assert.Equal(t, "secret-token", cfg.AuthToken)
	})
}

func TestSendHomerunMessage(t *testing.T) {
	t.Run("no-op when disabled", func(t *testing.T) {
		cfg := HomerunConfig{Enabled: false}
		// Should not panic or error
		SendHomerunMessage(cfg, HomerunMessage{Title: "test"})
	})

	t.Run("sends POST to pitcher", func(t *testing.T) {
		var received HomerunMessage
		var gotAuth string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/pitch", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			gotAuth = r.Header.Get("Authorization")

			err := json.NewDecoder(r.Body).Decode(&received)
			require.NoError(t, err)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"success"}`))
		}))
		defer srv.Close()

		cfg := HomerunConfig{
			Enabled:   true,
			URL:       srv.URL,
			AuthToken: "test-token",
		}

		msg := HomerunMessage{
			Title:    "Claim Order: postgresql",
			Message:  "Template ordered",
			Severity: "success",
			Author:   "test-user",
			System:   "claim-machinery-api",
			Tags:     "claim-order,database",
		}

		SendHomerunMessage(cfg, msg)

		assert.Equal(t, "Claim Order: postgresql", received.Title)
		assert.Equal(t, "success", received.Severity)
		assert.Equal(t, "test-user", received.Author)
		assert.Equal(t, "Bearer test-token", gotAuth)
		assert.NotEmpty(t, received.Timestamp)
	})

	t.Run("sends without auth token", func(t *testing.T) {
		var gotAuth string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		cfg := HomerunConfig{Enabled: true, URL: srv.URL}
		SendHomerunMessage(cfg, HomerunMessage{Title: "test"})

		assert.Empty(t, gotAuth)
	})
}

func TestBuildOrderMessage(t *testing.T) {
	msg := BuildOrderMessage(
		"postgresql",
		"PostgreSQL Database Claim",
		"database",
		[]string{"crossplane", "postgresql"},
		"oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim",
		"0.1.1",
		map[string]interface{}{
			"instanceClass": "db.t3.medium",
			"namespace":     "production",
		},
		"",
	)

	assert.Equal(t, "Claim Order: postgresql", msg.Title)
	assert.Contains(t, msg.Message, "PostgreSQL Database Claim")
	assert.Contains(t, msg.Message, "Parameters:")
	assert.Equal(t, "success", msg.Severity)
	assert.Equal(t, "claim-machinery-api", msg.Author)
	assert.Equal(t, "claim-machinery-api", msg.System)
	assert.Equal(t, "claim-order,database,crossplane,postgresql", msg.Tags)
	assert.Equal(t, "oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim:0.1.1", msg.Artifacts)
}

func TestBuildOrderMessage_WithAuthor(t *testing.T) {
	msg := BuildOrderMessage("test", "", "", nil, "oci://test", "", nil, "custom-user")

	assert.Equal(t, "custom-user", msg.Author)
	assert.Equal(t, "Claim Order: test", msg.Title)
	assert.Contains(t, msg.Message, "test")
	assert.NotContains(t, msg.Message, "Parameters:")
}
