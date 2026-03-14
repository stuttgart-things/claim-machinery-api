package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadTemplatesFromProfile_LocalFile(t *testing.T) {
	templates, sources, err := LoadTemplatesFromProfile("testdata/profile.yaml")
	require.NoError(t, err)
	require.Len(t, templates, 1)
	require.Len(t, sources, 1)
	require.Equal(t, "test-simple", templates[0].Metadata.Name)
}

func TestLoadTemplatesFromProfile_HTTPProfile(t *testing.T) {
	// Read the template file to serve it
	templateData, err := os.ReadFile("testdata/simple-template.yaml")
	require.NoError(t, err)

	// Start a test HTTP server that serves the template and profile
	mux := http.NewServeMux()
	mux.HandleFunc("/template.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.Write(templateData)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create a profile that references the HTTP-served template
	profileContent := fmt.Sprintf("templates:\n  - %s/template.yaml\n", srv.URL)
	profileFile, err := os.CreateTemp(t.TempDir(), "profile-*.yaml")
	require.NoError(t, err)
	_, err = profileFile.WriteString(profileContent)
	require.NoError(t, err)
	profileFile.Close()

	templates, sources, err := LoadTemplatesFromProfile(profileFile.Name())
	require.NoError(t, err)
	require.Len(t, templates, 1)
	require.Equal(t, "test-simple", templates[0].Metadata.Name)
	require.Len(t, sources, 1)
}

func TestLoadTemplatesFromProfile_HTTPProfilePath(t *testing.T) {
	// Read the template file to serve it
	templateData, err := os.ReadFile("testdata/simple-template.yaml")
	require.NoError(t, err)

	// Start a test HTTP server that serves both the profile and template
	mux := http.NewServeMux()
	mux.HandleFunc("/template.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.Write(templateData)
	})

	var srv *httptest.Server
	mux.HandleFunc("/profile.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		fmt.Fprintf(w, "templates:\n  - %s/template.yaml\n", srv.URL)
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	// Call LoadTemplatesFromProfile with an HTTP URL as the profile path
	templates, sources, err := LoadTemplatesFromProfile(srv.URL + "/profile.yaml")
	require.NoError(t, err)
	require.Len(t, templates, 1)
	require.Equal(t, "test-simple", templates[0].Metadata.Name)
	require.Len(t, sources, 1)
}

func TestLoadTemplatesFromProfile_MissingFile(t *testing.T) {
	_, _, err := LoadTemplatesFromProfile("testdata/nonexistent.yaml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "open profile")
}

func TestLoadProfileWithFunctions_ParsesFunctions(t *testing.T) {
	result, err := LoadProfileWithFunctions("testdata/profile-with-functions.yaml")
	require.NoError(t, err)
	require.Len(t, result.Templates, 1)
	require.Equal(t, "test-simple", result.Templates[0].Metadata.Name)
	require.NotNil(t, result.Functions)
	require.True(t, result.Functions.Has("get-ip"))
	require.False(t, result.Functions.Has("nonexistent"))
}

func TestLoadProfileWithFunctions_NoFunctions(t *testing.T) {
	result, err := LoadProfileWithFunctions("testdata/profile.yaml")
	require.NoError(t, err)
	require.Len(t, result.Templates, 1)
	require.NotNil(t, result.Functions)
	require.False(t, result.Functions.Has("get-ip"))
}
