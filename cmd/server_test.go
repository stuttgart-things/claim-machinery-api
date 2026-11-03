package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stuttgart-things/claim-machinery-api/internal/api"
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

func TestAnyRemote(t *testing.T) {
	assert.False(t, anyRemote(nil))
	assert.False(t, anyRemote([]string{"a.yaml", "/tmp/b.yaml"}))
	assert.True(t, anyRemote([]string{"a.yaml", "https://example.com/p.yaml"}))
	assert.True(t, anyRemote([]string{"http://example.com/p.yaml"}))
}

// A profile that fails at start is loaded by the background retry, which
// then clears the pending state (#107). The profile appears only after a few
// failed attempts. (A local file stands in for the URL: SplitProfilePaths cuts
// URLs at a port colon, and httptest URLs always carry a port.)
func TestRetryProfileLoadRecoversAfterFailures(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "template.yaml")
	profile := filepath.Join(dir, "profile.yaml")
	writeFile(t, tmpl, "apiVersion: v1\nkind: ClaimTemplate\nmetadata:\n  name: recovered\nspec:\n  type: test\n  source: oci://example/x\n")
	time.AfterFunc(20*time.Millisecond, func() {
		_ = os.WriteFile(profile, []byte("templates:\n  - "+tmpl+"\n"), 0o644)
	})

	server, err := api.NewServerWithTemplates(nil)
	require.NoError(t, err)
	server.SetProfilesPending(true)

	stop := make(chan struct{})
	defer close(stop)
	ok := retryProfileLoad(stop, server, false, "", profile, time.Millisecond, 5*time.Millisecond)

	require.True(t, ok)
	assert.False(t, server.ProfilesPending())
	assert.Equal(t, []string{"recovered"}, server.TemplateNames())
}

func TestRetryProfileLoadStops(t *testing.T) {
	server, err := api.NewServerWithTemplates(nil)
	require.NoError(t, err)
	server.SetProfilesPending(true)

	stop := make(chan struct{})
	time.AfterFunc(20*time.Millisecond, func() { close(stop) })
	ok := retryProfileLoad(stop, server, false, "", filepath.Join(t.TempDir(), "missing.yaml"), time.Millisecond, 5*time.Millisecond)

	assert.False(t, ok)
	assert.True(t, server.ProfilesPending())
}
