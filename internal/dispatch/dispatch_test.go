package dispatch_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stuttgart-things/claim-machinery-api/internal/dispatch"
	"github.com/stuttgart-things/claim-machinery-api/internal/dispatch/dispatchtest"
)

func order() dispatch.Order {
	return dispatch.Order{
		ID: "cma-0123456789ab", Template: "vm",
		Params:      map[string]interface{}{"vmName": "web1", "playbooks": []string{"baseos"}},
		RequestedBy: "patrick@sthings.example", Reason: "testing",
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	for _, name := range []string{dispatch.EnvAppID, dispatch.EnvInstallationID, dispatch.EnvPrivateKeyFile,
		dispatch.EnvRepo, dispatch.EnvWorkflow, dispatch.EnvRef, dispatch.EnvAPIURL} {
		t.Setenv(name, "")
	}
	cfg := dispatch.LoadConfig()
	assert.False(t, cfg.Enabled(), "off by default")
	assert.Equal(t, dispatch.DefaultRepo, cfg.Repo)
	assert.Equal(t, "dispatch-order-claim.yaml", cfg.Workflow)
	assert.Equal(t, "main", cfg.Ref)

	c, err := dispatch.New(cfg)
	require.NoError(t, err)
	assert.Nil(t, c)
}

func TestNewRejectsPartialOrBadConfig(t *testing.T) {
	gh := dispatchtest.New(t)

	cfg := gh.Config()
	cfg.InstallationID = ""
	_, err := dispatch.New(cfg)
	assert.ErrorContains(t, err, dispatch.EnvInstallationID)

	cfg = gh.Config()
	cfg.InstallationID = "abc"
	_, err = dispatch.New(cfg)
	assert.ErrorContains(t, err, "must be a number")

	cfg = gh.Config()
	cfg.Repo = "stuttgart-things"
	_, err = dispatch.New(cfg)
	assert.ErrorContains(t, err, "owner/name")

	cfg = gh.Config()
	cfg.PrivateKeyFile = filepath.Join(t.TempDir(), "missing.pem")
	_, err = dispatch.New(cfg)
	assert.ErrorContains(t, err, dispatch.EnvPrivateKeyFile)

	bad := filepath.Join(t.TempDir(), "bad.pem")
	require.NoError(t, os.WriteFile(bad, []byte("not a key"), 0o600))
	cfg = gh.Config()
	cfg.PrivateKeyFile = bad // pragma: allowlist secret
	_, err = dispatch.New(cfg)
	assert.ErrorContains(t, err, "no PEM block")
}

func TestPKCS8Key(t *testing.T) {
	gh := dispatchtest.New(t)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	f := filepath.Join(t.TempDir(), "pkcs8.pem")
	require.NoError(t, os.WriteFile(f, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600))
	cfg := gh.Config()
	cfg.PrivateKeyFile = f
	_, err = dispatch.New(cfg)
	require.NoError(t, err)
}

func TestDispatchWithRunDetails(t *testing.T) {
	gh := dispatchtest.New(t)
	c := gh.Client(t)

	run, err := c.Dispatch(context.Background(), order())
	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, dispatchtest.RunID, run.ID)
	assert.Equal(t, gh.RunURL(), run.URL)

	got := gh.Recorded()
	require.Len(t, got, 1)
	assert.Equal(t, "dispatch-order-claim.yaml", got[0].Workflow)
	assert.Equal(t, "main", got[0].Ref)
	assert.True(t, got[0].RunDetails)
	in := got[0].Inputs
	assert.Equal(t, "vm", in["template"])
	assert.Equal(t, "patrick@sthings.example", in["requested_by"])
	assert.Equal(t, "testing", in["reason"])
	assert.Equal(t, "cma-0123456789ab", in["order_id"])
	var params map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(in["params"]), &params), "params is JSON")
	assert.Equal(t, "web1", params["vmName"])
	assert.Equal(t, []interface{}{"baseos"}, params["playbooks"])

	// The installation token is cached.
	_, err = c.Dispatch(context.Background(), order())
	require.NoError(t, err)
	gh.Set(func(g *dispatchtest.GitHub) { assert.Equal(t, 1, g.TokenCalls) })
}

func TestDispatchFindsRunByOrderID(t *testing.T) {
	gh := dispatchtest.New(t)
	gh.Set(func(g *dispatchtest.GitHub) { g.RunDetails = false })
	run, err := gh.Client(t).Dispatch(context.Background(), order())
	require.NoError(t, err)
	require.NotNil(t, run, "found through the run-name")
	assert.Equal(t, dispatchtest.RunID, run.ID)
}

func TestDispatchWithoutRunDetailsSupport(t *testing.T) {
	gh := dispatchtest.New(t)
	gh.Set(func(g *dispatchtest.GitHub) { g.RejectRunDetails = true })
	run, err := gh.Client(t).Dispatch(context.Background(), order())
	require.NoError(t, err)
	require.NotNil(t, run)
	got := gh.Recorded()
	require.Len(t, got, 1, "retried without return_run_details")
	assert.False(t, got[0].RunDetails)
}

func TestDispatchFailure(t *testing.T) {
	gh := dispatchtest.New(t)
	gh.Set(func(g *dispatchtest.GitHub) { g.DispatchStatus = http.StatusNotFound })
	_, err := gh.Client(t).Dispatch(context.Background(), order())
	assert.ErrorContains(t, err, "404")
}

func TestBadAppKey(t *testing.T) {
	gh := dispatchtest.New(t)
	other := dispatchtest.New(t)
	cfg := gh.Config()
	cfg.PrivateKeyFile = other.KeyFile // pragma: allowlist secret (not the App key)
	c, err := dispatch.New(cfg)
	require.NoError(t, err)
	_, err = c.Dispatch(context.Background(), order())
	assert.ErrorContains(t, err, "installation token")
	assert.Empty(t, gh.Recorded())
}

func TestRunAndPR(t *testing.T) {
	gh := dispatchtest.New(t)
	c := gh.Client(t)
	ctx := context.Background()
	_, err := c.Dispatch(ctx, order())
	require.NoError(t, err)

	run, err := c.FindRun(ctx, "cma-0123456789ab")
	require.NoError(t, err)
	require.NotNil(t, run)
	assert.False(t, run.Done())

	run, err = c.FindRun(ctx, "cma-ffffffffffff")
	require.NoError(t, err)
	assert.Nil(t, run)

	gh.Set(func(g *dispatchtest.GitHub) { g.RunStatus, g.RunConclusion = "completed", "success" })
	run, err = c.GetRun(ctx, dispatchtest.RunID)
	require.NoError(t, err)
	assert.True(t, run.Done())
	assert.Equal(t, "success", run.Conclusion)

	pr, err := c.FindPR(ctx, "cma-0123456789ab")
	require.NoError(t, err)
	assert.Empty(t, pr, "no PR names the order yet")

	gh.Set(func(g *dispatchtest.GitHub) { g.PRBody = "Order ID: `cma-0123456789ab`" })
	pr, err = c.FindPR(ctx, "cma-0123456789ab")
	require.NoError(t, err)
	assert.Equal(t, dispatchtest.PRURL, pr)

	gh.Set(func(g *dispatchtest.GitHub) { g.PRStatus = http.StatusForbidden })
	_, err = c.FindPR(ctx, "cma-0123456789ab")
	assert.ErrorContains(t, err, "403")
}

func TestWorkflowURL(t *testing.T) {
	gh := dispatchtest.New(t)
	cfg := gh.Config()
	cfg.APIURL = dispatch.DefaultAPIURL
	c, err := dispatch.New(cfg)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/stuttgart-things/stuttgart-things/actions/workflows/dispatch-order-claim.yaml", c.WorkflowURL())
}
