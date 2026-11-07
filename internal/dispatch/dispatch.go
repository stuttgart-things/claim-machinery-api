// Package dispatch starts the GitOps repository's order workflow
// (workflow_dispatch) with a GitHub App installation token, finds the run it
// started and, later, the pull request that run opened (#131).
//
// It is off unless GITHUB_DISPATCH_APP_ID, GITHUB_DISPATCH_INSTALLATION_ID
// and GITHUB_DISPATCH_PRIVATE_KEY_FILE are set. The App needs actions:write
// on the repository (dispatch, runs); pull_requests:read lets the UI show the
// PR link of a private repository, without it only the run is linked.
package dispatch

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// Environment variables of the dispatch.
const (
	EnvAppID          = "GITHUB_DISPATCH_APP_ID"
	EnvInstallationID = "GITHUB_DISPATCH_INSTALLATION_ID"
	EnvPrivateKeyFile = "GITHUB_DISPATCH_PRIVATE_KEY_FILE" // pragma: allowlist secret
	EnvRepo           = "GITHUB_DISPATCH_REPO"
	EnvWorkflow       = "GITHUB_DISPATCH_WORKFLOW"
	EnvRef            = "GITHUB_DISPATCH_REF"
	EnvAPIURL         = "GITHUB_DISPATCH_API_URL"
)

// Defaults of the optional variables.
const (
	DefaultRepo     = "stuttgart-things/stuttgart-things"
	DefaultWorkflow = "dispatch-order-claim.yaml"
	DefaultRef      = "main"
	DefaultAPIURL   = "https://api.github.com"
)

// Config is the dispatch configuration.
type Config struct {
	AppID          string
	InstallationID string
	PrivateKeyFile string
	// Repo is owner/name of the GitOps repository.
	Repo     string
	Workflow string
	Ref      string
	// APIURL is the GitHub REST API (tests point it to a fake).
	APIURL string
}

// Enabled reports whether any credential variable is set; New rejects a
// partial set.
func (c Config) Enabled() bool {
	return c.AppID != "" || c.InstallationID != "" || c.PrivateKeyFile != ""
}

// LoadConfig reads the GITHUB_DISPATCH_* environment.
func LoadConfig() Config {
	get := func(name, def string) string {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
		return def
	}
	return Config{
		AppID:          get(EnvAppID, ""),
		InstallationID: get(EnvInstallationID, ""),
		PrivateKeyFile: get(EnvPrivateKeyFile, ""),
		Repo:           get(EnvRepo, DefaultRepo),
		Workflow:       get(EnvWorkflow, DefaultWorkflow),
		Ref:            get(EnvRef, DefaultRef),
		APIURL:         get(EnvAPIURL, DefaultAPIURL),
	}
}

// Client dispatches the order workflow. Use New.
type Client struct {
	cfg  Config
	key  *rsa.PrivateKey
	http *http.Client
	// now is the clock of the App JWT and the run search (tests set it).
	now func() time.Time
	// FindRunTimeout bounds the search for the dispatched run when GitHub
	// does not return it; Poll is the pause between two searches.
	FindRunTimeout time.Duration
	Poll           time.Duration

	mu      sync.Mutex
	token   string
	expires time.Time
}

// New checks cfg, reads the App's private key and returns the client, or nil
// when the dispatch is off.
func New(cfg Config) (*Client, error) {
	if !cfg.Enabled() {
		return nil, nil
	}
	var missing []string
	for name, v := range map[string]string{EnvAppID: cfg.AppID, EnvInstallationID: cfg.InstallationID, EnvPrivateKeyFile: cfg.PrivateKeyFile} {
		if v == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		return nil, fmt.Errorf("order as PR: %s required with the other GITHUB_DISPATCH_* variables", strings.Join(missing, ", "))
	}
	if _, err := strconv.ParseInt(cfg.InstallationID, 10, 64); err != nil {
		return nil, fmt.Errorf("%s must be a number, got %q", EnvInstallationID, cfg.InstallationID)
	}
	owner, name, ok := strings.Cut(cfg.Repo, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return nil, fmt.Errorf("%s must be owner/name, got %q", EnvRepo, cfg.Repo)
	}
	if cfg.Workflow == "" {
		cfg.Workflow = DefaultWorkflow
	}
	if cfg.Ref == "" {
		cfg.Ref = DefaultRef
	}
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultAPIURL
	}
	cfg.APIURL = strings.TrimRight(cfg.APIURL, "/")
	raw, err := os.ReadFile(cfg.PrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", EnvPrivateKeyFile, err)
	}
	key, err := parseKey(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", EnvPrivateKeyFile, err)
	}
	return &Client{
		cfg:            cfg,
		key:            key,
		http:           &http.Client{Timeout: 10 * time.Second},
		now:            time.Now,
		FindRunTimeout: 5 * time.Second,
		Poll:           1500 * time.Millisecond,
	}, nil
}

// parseKey reads the App's PEM private key: PKCS#1 (GitHub's download) or
// PKCS#8.
func parseKey(raw []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("no PEM block")
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("not an RSA private key: %w", err)
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}
	return rk, nil
}

// Config returns the effective configuration.
func (c *Client) Config() Config { return c.cfg }

// WorkflowURL is the workflow's page on GitHub, e.g. for an order whose run
// was not found in time.
func (c *Client) WorkflowURL() string {
	return c.htmlBase() + "/" + c.cfg.Repo + "/actions/workflows/" + url.PathEscape(c.cfg.Workflow)
}

// htmlBase is github.com for the public API, the GHES host otherwise.
func (c *Client) htmlBase() string {
	if c.cfg.APIURL == DefaultAPIURL {
		return "https://github.com"
	}
	return strings.TrimSuffix(c.cfg.APIURL, "/api/v3")
}

// appJWT is the App's RS256 JWT, valid for a few minutes (GitHub allows 10;
// iat is back-dated for clock drift).
func (c *Client) appJWT() (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: c.key},
		(&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return "", err
	}
	now := c.now()
	return jwt.Signed(signer).Claims(jwt.Claims{
		Issuer:   c.cfg.AppID,
		IssuedAt: jwt.NewNumericDate(now.Add(-60 * time.Second)),
		Expiry:   jwt.NewNumericDate(now.Add(9 * time.Minute)),
	}).Serialize()
}

// installationToken returns a cached installation access token, minting a
// new one (limited to the GitOps repository) when it is about to expire.
func (c *Client) installationToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && c.now().Before(c.expires.Add(-5*time.Minute)) {
		return c.token, nil
	}
	appJWT, err := c.appJWT()
	if err != nil {
		return "", fmt.Errorf("signing the App JWT: %w", err)
	}
	_, repo, _ := strings.Cut(c.cfg.Repo, "/")
	body, _ := json.Marshal(map[string]any{"repositories": []string{repo}})
	var out struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	status, err := c.do(ctx, http.MethodPost, "/app/installations/"+c.cfg.InstallationID+"/access_tokens", "Bearer "+appJWT, body, &out)
	if err != nil {
		return "", fmt.Errorf("installation token: %w", err)
	}
	if status != http.StatusCreated || out.Token == "" {
		return "", fmt.Errorf("installation token: GitHub answered %d", status)
	}
	c.token, c.expires = out.Token, out.ExpiresAt
	return c.token, nil
}

// apiError is a non-2xx GitHub answer.
type apiError struct {
	Status  int
	Message string
}

func (e *apiError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("GitHub answered %d", e.Status)
	}
	return fmt.Sprintf("GitHub answered %d: %s", e.Status, e.Message)
}

// do sends one API request. A 2xx answer is decoded into out (if any) and
// its status returned; anything else is an *apiError.
func (c *Client) do(ctx context.Context, method, path, authz string, body []byte, out any) (int, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.APIURL+path, rd)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", authz)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &e)
		return resp.StatusCode, &apiError{Status: resp.StatusCode, Message: e.Message}
	}
	if out != nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decoding %s: %w", path, err)
		}
	}
	return resp.StatusCode, nil
}

// api is do with the installation token.
func (c *Client) api(ctx context.Context, method, path string, body []byte, out any) (int, error) {
	tok, err := c.installationToken(ctx)
	if err != nil {
		return 0, err
	}
	return c.do(ctx, method, path, "Bearer "+tok, body, out)
}

// Order is one order to dispatch.
type Order struct {
	// ID correlates the run (its run-name) and the PR (its body).
	ID          string
	Template    string
	Params      map[string]interface{}
	RequestedBy string
	Reason      string
}

// Run is a workflow run.
type Run struct {
	ID         int64  `json:"id"`
	URL        string `json:"html_url"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Title      string `json:"display_title"`
}

// Done reports whether the run finished.
func (r Run) Done() bool { return r.Status == "completed" }

// Dispatch starts the workflow for o and returns the run it started. A nil
// run with a nil error means the dispatch went through but the run did not
// show up in time; FindRun looks again later.
func (c *Client) Dispatch(ctx context.Context, o Order) (*Run, error) {
	params, err := json.Marshal(o.Params)
	if err != nil {
		return nil, fmt.Errorf("encoding params: %w", err)
	}
	payload := map[string]any{
		"ref": c.cfg.Ref,
		"inputs": map[string]string{
			"template":     o.Template,
			"params":       string(params),
			"requested_by": o.RequestedBy,
			"reason":       o.Reason,
			"order_id":     o.ID,
		},
		// GitHub answers 200 with the run instead of 204 (2026-02).
		"return_run_details": true,
	}
	started := c.now().Add(-time.Minute)
	path := "/repos/" + c.cfg.Repo + "/actions/workflows/" + url.PathEscape(c.cfg.Workflow) + "/dispatches"
	body, _ := json.Marshal(payload)
	var details struct {
		RunID   int64  `json:"workflow_run_id"`
		HTMLURL string `json:"html_url"`
	}
	_, err = c.api(ctx, http.MethodPost, path, body, &details)
	var ae *apiError
	if errors.As(err, &ae) && ae.Status == http.StatusUnprocessableEntity && strings.Contains(ae.Message, "return_run_details") {
		// An API without run details (GHES): dispatch the classic way.
		delete(payload, "return_run_details")
		body, _ = json.Marshal(payload)
		_, err = c.api(ctx, http.MethodPost, path, body, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("dispatching %s: %w", c.cfg.Workflow, err)
	}
	if details.RunID != 0 {
		return &Run{ID: details.RunID, URL: details.HTMLURL, Status: "queued"}, nil
	}

	// No run in the answer: look for the run named after the order, like
	// claims change's findDispatchedRun, but by its order id.
	deadline := c.now().Add(c.FindRunTimeout)
	for {
		run, err := c.findRun(ctx, o.ID, started)
		if err != nil || run != nil {
			return run, err
		}
		if !c.now().Before(deadline) {
			return nil, nil
		}
		select {
		case <-ctx.Done():
			return nil, nil
		case <-time.After(c.Poll):
		}
	}
}

// FindRun looks for the run of an order dispatched in the last hour; nil
// when there is none (yet).
func (c *Client) FindRun(ctx context.Context, orderID string) (*Run, error) {
	return c.findRun(ctx, orderID, c.now().Add(-time.Hour))
}

func (c *Client) findRun(ctx context.Context, orderID string, after time.Time) (*Run, error) {
	q := url.Values{
		"event":    {"workflow_dispatch"},
		"per_page": {"50"},
		"created":  {">=" + after.UTC().Format(time.RFC3339)},
	}
	var out struct {
		Runs []Run `json:"workflow_runs"`
	}
	path := "/repos/" + c.cfg.Repo + "/actions/workflows/" + url.PathEscape(c.cfg.Workflow) + "/runs?" + q.Encode()
	if _, err := c.api(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, fmt.Errorf("listing runs: %w", err)
	}
	for i := range out.Runs {
		if strings.Contains(out.Runs[i].Title, orderID) {
			return &out.Runs[i], nil
		}
	}
	return nil, nil
}

// GetRun returns the run's current status.
func (c *Client) GetRun(ctx context.Context, id int64) (*Run, error) {
	var r Run
	if _, err := c.api(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/actions/runs/%d", c.cfg.Repo, id), nil, &r); err != nil {
		return nil, fmt.Errorf("run %d: %w", id, err)
	}
	return &r, nil
}

// FindPR returns the URL of the pull request whose body names the order, or
// "" when there is none among the latest ones. The workflow writes the order
// id into the body. Needs pull_requests:read on a private repository.
func (c *Client) FindPR(ctx context.Context, orderID string) (string, error) {
	var prs []struct {
		URL  string `json:"html_url"`
		Body string `json:"body"`
	}
	path := "/repos/" + c.cfg.Repo + "/pulls?state=all&sort=created&direction=desc&per_page=30"
	if _, err := c.api(ctx, http.MethodGet, path, nil, &prs); err != nil {
		return "", fmt.Errorf("listing pull requests: %w", err)
	}
	for _, pr := range prs {
		if strings.Contains(pr.Body, orderID) {
			return pr.URL, nil
		}
	}
	return "", nil
}
