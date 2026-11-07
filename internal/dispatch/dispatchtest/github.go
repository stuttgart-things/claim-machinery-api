// Package dispatchtest is a fake GitHub API for tests: the App's
// installation token, workflow_dispatch, workflow runs and pull requests.
package dispatchtest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stuttgart-things/claim-machinery-api/internal/dispatch"
)

const (
	AppID          = "4242"
	InstallationID = "77"
	Token          = "fake-installation-token"
	Repo           = "stuttgart-things/stuttgart-things"
	RunID          = int64(9001)
)

// Dispatch is one recorded workflow_dispatch.
type Dispatch struct {
	Workflow string
	Ref      string
	Inputs   map[string]string
	// RunDetails is whether the caller asked for return_run_details.
	RunDetails bool
}

// GitHub is a running fake.
type GitHub struct {
	Server *httptest.Server
	// KeyFile is the App's PEM private key (PKCS#1).
	KeyFile string
	key     *rsa.PrivateKey

	mu sync.Mutex
	// RunDetails: answer the dispatch with the run (200) instead of 204.
	RunDetails bool
	// RejectRunDetails: answer return_run_details with 422 (old API).
	RejectRunDetails bool
	// DispatchStatus, when set, is the dispatch's (error) answer.
	DispatchStatus int
	// RunStatus / RunConclusion of the run (default queued).
	RunStatus, RunConclusion string
	// PRBody, when set, is the body of the one pull request listed.
	PRBody string
	// PRStatus, when set, is the answer of the pull request list.
	PRStatus   int
	Dispatches []Dispatch
	TokenCalls int
}

// RunURL is the run's html_url.
func (g *GitHub) RunURL() string { return "https://github.com/" + Repo + "/actions/runs/9001" }

// PRURL is the pull request's html_url.
const PRURL = "https://github.com/" + Repo + "/pull/4711"

// New starts the fake, stopped with the test.
func New(t testing.TB) *GitHub {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	g := &GitHub{key: key, RunDetails: true, RunStatus: "queued"}
	g.KeyFile = filepath.Join(t.TempDir(), "app.pem")
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(g.KeyFile, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /app/installations/{id}/access_tokens", g.accessToken)
	mux.HandleFunc("POST /repos/{owner}/{repo}/actions/workflows/{wf}/dispatches", g.authed(g.dispatch))
	mux.HandleFunc("GET /repos/{owner}/{repo}/actions/workflows/{wf}/runs", g.authed(g.listRuns))
	mux.HandleFunc("GET /repos/{owner}/{repo}/actions/runs/{id}", g.authed(g.getRun))
	mux.HandleFunc("GET /repos/{owner}/{repo}/pulls", g.authed(g.listPulls))
	g.Server = httptest.NewServer(mux)
	t.Cleanup(g.Server.Close)
	return g
}

// Config is a dispatch config for the fake.
func (g *GitHub) Config() dispatch.Config {
	return dispatch.Config{
		AppID: AppID, InstallationID: InstallationID, PrivateKeyFile: g.KeyFile,
		Repo: Repo, Workflow: dispatch.DefaultWorkflow, Ref: dispatch.DefaultRef, APIURL: g.Server.URL,
	}
}

// Client is a dispatch client for the fake that searches runs quickly.
func (g *GitHub) Client(t testing.TB) *dispatch.Client {
	t.Helper()
	c, err := dispatch.New(g.Config())
	if err != nil {
		t.Fatal(err)
	}
	c.FindRunTimeout, c.Poll = 300*time.Millisecond, 20*time.Millisecond
	return c
}

// Set changes the fake's behaviour under its lock.
func (g *GitHub) Set(f func(*GitHub)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	f(g)
}

// Recorded returns the dispatches so far.
func (g *GitHub) Recorded() []Dispatch {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]Dispatch(nil), g.Dispatches...)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// accessToken checks the App JWT (RS256, signed with the App's key, iss =
// App ID, short-lived) and hands out the installation token.
func (g *GitHub) accessToken(w http.ResponseWriter, r *http.Request) {
	raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	tok, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.RS256})
	var claims jwt.Claims
	if !ok || err != nil || tok.Claims(&g.key.PublicKey, &claims) != nil ||
		claims.Issuer != AppID || claims.Expiry == nil || time.Until(claims.Expiry.Time()) > 10*time.Minute {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "A JSON web token could not be decoded"})
		return
	}
	if r.PathValue("id") != InstallationID {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
		return
	}
	var body struct {
		Repositories []string `json:"repositories"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if len(body.Repositories) != 1 || body.Repositories[0] != "stuttgart-things" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "token must be limited to the GitOps repository"})
		return
	}
	g.mu.Lock()
	g.TokenCalls++
	g.mu.Unlock()
	writeJSON(w, http.StatusCreated, map[string]any{"token": Token, "expires_at": time.Now().Add(time.Hour)})
}

func (g *GitHub) authed(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+Token {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Bad credentials"})
			return
		}
		if r.PathValue("owner")+"/"+r.PathValue("repo") != Repo {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
			return
		}
		h(w, r)
	}
}

func (g *GitHub) dispatch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Ref        string            `json:"ref"`
		Inputs     map[string]string `json:"inputs"`
		RunDetails *bool             `json:"return_run_details"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Problems parsing JSON"})
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.DispatchStatus != 0 {
		writeJSON(w, g.DispatchStatus, map[string]string{"message": "Workflow does not have 'workflow_dispatch' trigger"})
		return
	}
	asked := body.RunDetails != nil && *body.RunDetails
	if asked && g.RejectRunDetails {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "Invalid request.\n\n\"return_run_details\" is not a permitted key."})
		return
	}
	g.Dispatches = append(g.Dispatches, Dispatch{Workflow: r.PathValue("wf"), Ref: body.Ref, Inputs: body.Inputs, RunDetails: asked})
	if asked && g.RunDetails {
		writeJSON(w, http.StatusOK, map[string]any{
			"workflow_run_id": RunID,
			"run_url":         g.Server.URL + "/repos/" + Repo + "/actions/runs/9001",
			"html_url":        g.RunURL(),
		})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// run is the one run, named after the last dispatch's order id as the
// workflow's run-name does.
func (g *GitHub) run() map[string]any {
	title := "Order claim"
	if n := len(g.Dispatches); n > 0 {
		title = "Order claim " + g.Dispatches[n-1].Inputs["template"] + " (" + g.Dispatches[n-1].Inputs["order_id"] + ")"
	}
	return map[string]any{
		"id": RunID, "html_url": g.RunURL(), "status": g.RunStatus,
		"conclusion": g.RunConclusion, "display_title": title,
	}
}

func (g *GitHub) listRuns(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	runs := []map[string]any{{"id": 1, "display_title": "Order claim other (cma-000000000000)", "status": "completed"}}
	if len(g.Dispatches) > 0 {
		runs = append(runs, g.run())
	}
	writeJSON(w, http.StatusOK, map[string]any{"total_count": len(runs), "workflow_runs": runs})
}

func (g *GitHub) getRun(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if r.PathValue("id") != "9001" {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
		return
	}
	writeJSON(w, http.StatusOK, g.run())
}

func (g *GitHub) listPulls(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.PRStatus != 0 {
		writeJSON(w, g.PRStatus, map[string]string{"message": "Resource not accessible by integration"})
		return
	}
	prs := []map[string]any{{"html_url": "https://github.com/" + Repo + "/pull/1", "body": "unrelated"}}
	if g.PRBody != "" {
		prs = append(prs, map[string]any{"html_url": PRURL, "body": g.PRBody})
	}
	writeJSON(w, http.StatusOK, prs)
}
