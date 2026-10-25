package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/notify"
)

func intPtr(i int) *int { return &i }

// uiTestTemplate covers every widget: select with random, multiselect,
// boolean, validated text, a hidden parameter and a profile function.
func uiTestTemplate() *claimtemplate.ClaimTemplate {
	return &claimtemplate.ClaimTemplate{
		Metadata: claimtemplate.ClaimTemplateMetadata{
			Name: "vm", Title: "Virtual Machine", Description: "A test VM", Tags: []string{"vsphere", "vm"},
		},
		Spec: claimtemplate.ClaimTemplateSpec{
			Source: "oci://example/vm", Tag: "1.0.0",
			Parameters: []claimtemplate.Parameter{
				{Name: "size", Title: "Size", Type: "string", Enum: []string{"small", "large"}, Default: "small", AllowRandom: true},
				{Name: "playbooks", Type: "array", Multiselect: true, Enum: []string{"baseos", "docker"}, Default: []interface{}{"baseos"}},
				{Name: "backup", Type: "boolean", Default: true},
				{Name: "vmName", Title: "VM Name", Type: "string", Required: true, Pattern: "^[a-z][a-z0-9-]*$", MaxLength: intPtr(15)},
				{Name: "ip", Type: "string", ValueFrom: &claimtemplate.ValueFromSpec{Function: "clusterbookIP"}},
				{Name: "cluster", Type: "string", Hidden: true, Default: "labda"},
			},
		},
	}
}

// newUITestServer returns a server whose render records the parameters
// instead of running KCL.
func newUITestServer(t *testing.T, basePath string) (*Server, *map[string]interface{}) {
	t.Helper()
	s, err := NewServerWithTemplates([]*claimtemplate.ClaimTemplate{uiTestTemplate()})
	require.NoError(t, err)
	if basePath != "" {
		s.SetBasePath(basePath)
	}
	var got map[string]interface{}
	s.render = func(_ *claimtemplate.ClaimTemplate, params map[string]interface{}) (string, error) {
		got = params
		return "kind: VM\nname: <" + params["vmName"].(string) + ">\n", nil
	}
	return s, &got
}

func serve(s *Server, method, path string, form url.Values, header ...string) *httptest.ResponseRecorder {
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

func TestUIIndex(t *testing.T) {
	s, err := NewServer(testdataDir(t))
	require.NoError(t, err)

	w := serve(s, http.MethodGet, "/ui/", nil)
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	for _, want := range []string{
		`href="/ui/templates/postgresql"`,
		`href="/ui/templates/multiselect-test"`,
		"PostgreSQL Database Claim",
		`data-tag="database"`,
		`id="search"`,
		"4 templates",
	} {
		assert.Contains(t, body, want)
	}

	for _, path := range []string{"/ui/static/htmx.min.js", "/ui/static/ui.css"} {
		assert.Equal(t, http.StatusOK, serve(s, http.MethodGet, path, nil).Code, path)
	}
	assert.Equal(t, http.StatusMovedPermanently, serve(s, http.MethodGet, "/ui", nil).Code)
}

func TestUITemplateForm(t *testing.T) {
	s, _ := newUITestServer(t, "")

	w := serve(s, http.MethodGet, "/ui/templates/vm", nil)
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	for _, want := range []string{
		`hx-post="/ui/templates/vm/render"`,
		`<option value="small" selected>small</option>`,
		`<option value="__random__">`,
		`name="playbooks" value="baseos" checked`,
		`name="playbooks" value="docker">`,
		`<option value="true" selected>true</option>`,
		`pattern="^[a-z][a-z0-9-]*$"`,
		`maxlength="15"`,
		`placeholder="from clusterbookIP"`,
		"1 platform parameter uses its default",
		`<button type="submit" name="_action" value="preview" class="secondary">Preview</button>`,
		`<button type="submit" name="_action" value="order">Order</button>`,
	} {
		assert.Contains(t, body, want)
	}
	assert.NotContains(t, body, `name="cluster"`, "hidden parameter must not be in the form")

	assert.Equal(t, http.StatusNotFound, serve(s, http.MethodGet, "/ui/templates/nope", nil).Code)
}

func TestUIRender(t *testing.T) {
	s, got := newUITestServer(t, "")

	w := serve(s, http.MethodPost, "/ui/templates/vm/render", url.Values{
		"size":      {"__random__"},
		"playbooks": {"baseos", "docker", "evil"},
		"backup":    {"false"},
		"vmName":    {"web-1"},
		"ip":        {""},
		"cluster":   {"overridden"},
		"_author":   {"patrick"},
	})
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "name: &lt;web-1&gt;", "rendered YAML is shown escaped")
	assert.Contains(t, body, `data-download="vm"`)

	params := *got
	assert.Contains(t, []interface{}{"small", "large"}, params["size"], "random picks an enum value")
	assert.Equal(t, []string{"baseos", "docker"}, params["playbooks"], "unknown multiselect values are dropped")
	assert.Equal(t, "false", params["backup"])
	assert.Equal(t, "web-1", params["vmName"])
	assert.Equal(t, "labda", params["cluster"], "hidden parameter keeps its default")
	assert.Equal(t, "", params["ip"], "empty field falls back to the default")
}

func TestUIRenderValidation(t *testing.T) {
	s, got := newUITestServer(t, "")

	w := serve(s, http.MethodPost, "/ui/templates/vm/render", url.Values{
		"vmName": {"Not Valid"},
		"size":   {"huge"},
	})
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Render failed")
	assert.Contains(t, body, "VM Name does not match")
	assert.Contains(t, body, "Size must be one of small, large")
	assert.Nil(t, *got, "render must not run on invalid input")

	w = serve(s, http.MethodPost, "/ui/templates/vm/render", url.Values{"vmName": {""}})
	assert.Contains(t, w.Body.String(), "VM Name is required")
}

func TestUIBasePath(t *testing.T) {
	s, _ := newUITestServer(t, "/claims")

	for _, path := range []string{
		"/claims/ui/", "/claims/ui/templates/vm", "/claims/ui/static/htmx.min.js",
		"/claims/docs", "/claims/health", "/claims/api/v1/claim-templates",
		"/health", "/api/v1/claim-templates",
	} {
		assert.Equal(t, http.StatusOK, serve(s, http.MethodGet, path, nil).Code, path)
	}
	for _, path := range []string{"/ui/", "/ui/templates/vm", "/docs"} {
		assert.Equal(t, http.StatusNotFound, serve(s, http.MethodGet, path, nil).Code, path)
	}
	w := serve(s, http.MethodGet, "/claims", nil)
	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Equal(t, "/claims/", w.Header().Get("Location"))

	w = serve(s, http.MethodGet, "/claims/", nil, "Accept", "text/html")
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/claims/ui/", w.Header().Get("Location"))
	assert.Contains(t, serve(s, http.MethodGet, "/claims/", nil).Body.String(), `"service": "claim-machinery-api"`, "API clients still get JSON")

	assert.Contains(t, serve(s, http.MethodGet, "/claims/docs", nil).Body.String(), `spec-url="/claims/openapi.yaml"`)

	// Every URL handed to the browser carries the prefix.
	linkRE := regexp.MustCompile(`(?:href|src|hx-post|hx-get)="(/[^"]*)"`)
	for _, path := range []string{"/claims/ui/", "/claims/ui/templates/vm"} {
		matches := linkRE.FindAllStringSubmatch(serve(s, http.MethodGet, path, nil).Body.String(), -1)
		require.NotEmpty(t, matches, path)
		for _, m := range matches {
			assert.True(t, strings.HasPrefix(m[1], "/claims/"), "%s: %s lacks the prefix", path, m[1])
		}
	}

	w = serve(s, http.MethodPost, "/claims/ui/templates/vm/render", url.Values{"vmName": {"web-1"}})
	assert.Contains(t, w.Body.String(), "name: &lt;web-1&gt;")
}

func TestParseBasePath(t *testing.T) {
	for in, want := range map[string]string{
		"": "", "/": "", "/claims": "/claims", "/claims/": "/claims", " /lab/claims": "/lab/claims",
	} {
		got, err := ParseBasePath(in)
		assert.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"claims", "/a//b", "/a/../b", "/a b", "/a?x", "/a#x", "/a%2f"} {
		_, err := ParseBasePath(in)
		assert.Error(t, err, in)
	}
}

// captureNotifications replaces the homerun sender; order sends from a
// goroutine, so messages arrive on the channel.
func captureNotifications(s *Server) chan notify.HomerunMessage {
	sent := make(chan notify.HomerunMessage, 4)
	s.homerun.Enabled = true
	s.sendNotification = func(msg notify.HomerunMessage) { sent <- msg }
	return sent
}

func TestUIPreviewDoesNotNotify(t *testing.T) {
	for name, form := range map[string]url.Values{
		"preview button": {"vmName": {"web-1"}, "_action": {"preview"}},
		"no action":      {"vmName": {"web-1"}},
	} {
		t.Run(name, func(t *testing.T) {
			s, got := newUITestServer(t, "")
			sent := captureNotifications(s)

			body := serve(s, http.MethodPost, "/ui/templates/vm/render", form).Body.String()
			assert.Contains(t, body, "name: &lt;web-1&gt;")
			assert.Contains(t, body, "preview · no notification")
			assert.NotNil(t, *got, "a preview renders")
			// order returns before it would start the notification goroutine,
			// so nothing can still be on its way here.
			assert.Empty(t, sent)
		})
	}
}

func TestUIOrderNotifies(t *testing.T) {
	s, _ := newUITestServer(t, "")
	sent := captureNotifications(s)

	body := serve(s, http.MethodPost, "/ui/templates/vm/render", url.Values{
		"vmName": {"web-1"}, "_action": {"order"}, "_author": {"patrick"},
	}).Body.String()
	assert.Contains(t, body, "ordered · notification sent")
	select {
	case msg := <-sent:
		assert.Contains(t, msg.Author, "patrick")
	case <-time.After(2 * time.Second):
		t.Fatal("order sent no notification")
	}

	s.homerun.Enabled = false
	body = serve(s, http.MethodPost, "/ui/templates/vm/render", url.Values{"vmName": {"web-1"}, "_action": {"order"}}).Body.String()
	assert.Contains(t, body, "ordered · notifications are off")
}

func TestAPIOrderNotifies(t *testing.T) {
	s, _ := newUITestServer(t, "")
	sent := captureNotifications(s)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/claim-templates/vm/order",
		strings.NewReader(`{"parameters":{"vmName":"web-1"},"author":"patrick"}`))
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	select {
	case msg := <-sent:
		assert.Contains(t, msg.Author, "patrick")
	case <-time.After(2 * time.Second):
		t.Fatal("the order endpoint sent no notification")
	}
}
