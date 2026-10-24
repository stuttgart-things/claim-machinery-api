package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestProfileHashChangesWithReferencedTemplate(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "template.yaml")
	profile := filepath.Join(dir, "profile.yaml")
	writeFile(t, tmpl, "kind: ClaimTemplate\nmetadata:\n  name: a\n")
	writeFile(t, profile, "templates:\n  - "+tmpl+"\n")

	before, ok := profileHash(profile)
	require.True(t, ok)

	again, ok := profileHash(profile)
	require.True(t, ok)
	assert.Equal(t, before, again, "unchanged sources must hash the same")

	// Only the template changes, the profile file stays byte-identical.
	writeFile(t, tmpl, "kind: ClaimTemplate\nmetadata:\n  name: a\n  description: changed\n")
	after, ok := profileHash(profile)
	require.True(t, ok)
	assert.NotEqual(t, before, after, "a changed template must change the profile hash")
}

func TestProfileHashRemoteTemplate(t *testing.T) {
	body := "kind: ClaimTemplate\nmetadata:\n  name: remote\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	profile := filepath.Join(t.TempDir(), "profile.yaml")
	writeFile(t, profile, "templates:\n  - "+srv.URL+"/template.yaml\n")

	before, ok := profileHash(profile)
	require.True(t, ok)

	body = "kind: ClaimTemplate\nmetadata:\n  name: remote\n  description: changed\n"
	after, ok := profileHash(profile)
	require.True(t, ok)
	assert.NotEqual(t, before, after)
}

func TestProfileHashNotOKWhenSourceUnreadable(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.yaml")
	writeFile(t, profile, "templates:\n  - "+filepath.Join(dir, "missing.yaml")+"\n")

	_, ok := profileHash(profile)
	assert.False(t, ok, "an unreadable template must not count as a change")

	_, ok = profileHash(filepath.Join(dir, "no-such-profile.yaml"))
	assert.False(t, ok)
}
