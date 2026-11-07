package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stuttgart-things/claim-machinery-api/internal/auth"
	"github.com/stuttgart-things/claim-machinery-api/internal/auth/authtest"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/dispatch/dispatchtest"
	"github.com/stuttgart-things/claim-machinery-api/internal/notify"
)

const prPath = "/api/v1/claim-templates/vm/order-pr"

// newOrderPRServer is the UI test server with order auth and the dispatch
// against a fake GitHub, plus templates with secrets, a login-bound
// parameter and template roles.
func newOrderPRServer(t *testing.T, mod func(*auth.Config)) (*Server, *authtest.Issuer, *dispatchtest.GitHub, chan notify.HomerunMessage) {
	t.Helper()
	s, iss, _, sent := newAuthTestServer(t, "", mod)
	secrets := uiTestTemplate()
	secrets.Metadata.Name = "vm-secrets"
	secrets.Spec.Secrets = []claimtemplate.SecretTemplate{{Name: "creds"}}
	owned := identityTemplate()
	owned.Metadata.Name = "vm-owned"
	owned.Spec.Parameters = owned.Spec.Parameters[:2] // name, owner (no identity: user)
	s.UpdateTemplates([]*claimtemplate.ClaimTemplate{uiTestTemplate(), secrets, identityTemplate(), owned})
	gh := dispatchtest.New(t)
	s.SetDispatch(gh.Client(t))
	return s, iss, gh, sent
}

func bearer(t *testing.T, iss *authtest.Issuer, roles ...string) string {
	return token(t, iss, roles...)
}

func TestOrderPROffByDefault(t *testing.T) {
	// Neither auth nor dispatch: no endpoint, no button.
	s, _ := newUITestServer(t, "")
	w := apiOrder(s, prPath, orderBody)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NotContains(t, serve(s, http.MethodGet, "/ui/templates/vm", nil).Body.String(), "Order as PR")

	// The dispatch alone (no AUTH_OIDC_ISSUER): still off.
	gh := dispatchtest.New(t)
	s.SetDispatch(gh.Client(t))
	w = apiOrder(s, prPath, orderBody)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NotContains(t, serve(s, http.MethodGet, "/ui/templates/vm", nil).Body.String(), "Order as PR")
	w = serve(s, http.MethodPost, "/ui/templates/vm/render", url.Values{"vmName": {"web1"}, "_action": {"order-pr"}}, "HX-Request", "true")
	assert.Contains(t, w.Body.String(), "order as PR is not enabled")
	assert.Equal(t, http.StatusNotFound, serve(s, http.MethodGet, "/ui/order-pr/cma-0123456789ab", nil).Code)
	assert.Empty(t, gh.Recorded())

	// Auth alone: off too.
	s2, iss, _, _ := newAuthTestServer(t, "", nil)
	w = apiOrder(s2, prPath, orderBody, "Authorization", bearer(t, iss))
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NotContains(t, serve(s2, http.MethodGet, "/ui/templates/vm", nil).Body.String(), "Order as PR")
}

func TestOrderPRAPI(t *testing.T) {
	s, iss, gh, sent := newOrderPRServer(t, nil)

	// No token: 401, nothing dispatched.
	w := apiOrder(s, prPath, orderBody)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	// The portal's identity never authorizes the API.
	w = apiOrder(s, prPath, orderBody, auth.HeaderForwardedUser, "mallory")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, gh.Recorded())

	// requested_by is the token's user; a requested_by in the body changes nothing.
	body := `{"parameters":{"vmName":"web1"},"reason":"new web server","requested_by":"mallory","author":"mallory"}`
	w = apiOrder(s, prPath, body, "Authorization", bearer(t, iss))
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	var resp OrderPRResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "OrderPRResponse", resp.Kind)
	assert.Regexp(t, `^cma-[0-9a-f]{12}$`, resp.OrderID)
	assert.Equal(t, "patrick@sthings.example", resp.RequestedBy)
	require.NotNil(t, resp.Run)
	assert.Equal(t, gh.RunURL(), resp.Run.URL)
	assert.Equal(t, dispatchtest.RunID, resp.Run.ID)

	got := gh.Recorded()
	require.Len(t, got, 1)
	in := got[0].Inputs
	assert.Equal(t, "vm", in["template"])
	assert.Equal(t, "patrick@sthings.example", in["requested_by"])
	assert.Equal(t, "new web server", in["reason"])
	assert.Equal(t, resp.OrderID, in["order_id"])
	assert.JSONEq(t, `{"vmName":"web1"}`, in["params"], "the user's parameters, not the merged ones")

	msg := expectNotification(t, sent)
	assert.Equal(t, "Claim Order (PR): vm", msg.Title)
	assert.Equal(t, "patrick@sthings.example", msg.Author)
	assert.Equal(t, gh.RunURL(), msg.Url)
}

func TestOrderPRRefusals(t *testing.T) {
	s, iss, gh, sent := newOrderPRServer(t, nil)
	tok := bearer(t, iss)

	w := apiOrder(s, "/api/v1/claim-templates/nope/order-pr", orderBody, "Authorization", tok)
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = apiOrder(s, prPath, "{", "Authorization", tok)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = apiOrder(s, "/api/v1/claim-templates/vm-secrets/order-pr", orderBody, "Authorization", tok)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, decodeError(t, w), "creates secrets")

	w = apiOrder(s, "/api/v1/claim-templates/vm-id/order-pr", `{"parameters":{}}`, "Authorization", bearer(t, iss, "backstage-developer"))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, decodeError(t, w), "requested_by is bound to the login")

	// The template's orderRoles apply as for Order.
	w = apiOrder(s, "/api/v1/claim-templates/vm-owned/order-pr", `{"parameters":{}}`, "Authorization", tok)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// A parameter the render rejects is not dispatched.
	s.render = func(*claimtemplate.ClaimTemplate, map[string]interface{}) (string, error) {
		return "", errors.New("vmName: does not match")
	}
	w = apiOrder(s, prPath, orderBody, "Authorization", tok)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, decodeError(t, w), "does not match")

	assert.Empty(t, gh.Recorded())
	expectNoNotification(t, sent)

	// GitHub refusing the dispatch is a 502.
	s2, iss2, gh2, _ := newOrderPRServer(t, nil)
	gh2.Set(func(g *dispatchtest.GitHub) { g.DispatchStatus = http.StatusNotFound })
	w = apiOrder(s2, prPath, orderBody, "Authorization", bearer(t, iss2))
	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, "starting the order workflow failed", decodeError(t, w))
}

func TestOrderPROwnerBound(t *testing.T) {
	s, iss, gh, _ := newOrderPRServer(t, nil)
	s.render = func(*claimtemplate.ClaimTemplate, map[string]interface{}) (string, error) { return "kind: VM\n", nil }
	w := apiOrder(s, "/api/v1/claim-templates/vm-owned/order-pr", `{"parameters":{"name":"db1"}}`, "Authorization", bearer(t, iss, "backstage-developer"))
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	got := gh.Recorded()
	require.Len(t, got, 1)
	assert.JSONEq(t, `{"name":"db1","owner":"user:default/patrick"}`, got[0].Inputs["params"], "owner filled from the login")
}

func TestOrderPRUI(t *testing.T) {
	s, _, gh, sent := newOrderPRServer(t, func(c *auth.Config) { c.TrustForwardedUser = true })

	page := serve(s, http.MethodGet, "/ui/templates/vm", nil).Body.String()
	assert.Contains(t, page, `value="order-pr">Order as PR</button>`)
	assert.Contains(t, page, `name="_reason"`)
	page = serve(s, http.MethodGet, "/ui/templates/vm-secrets", nil).Body.String()
	assert.NotContains(t, page, `value="order-pr"`)
	assert.Contains(t, page, "No Order as PR: this template creates secrets")

	form := url.Values{"vmName": {"web1"}, "_action": {"order-pr"}, "_reason": {"for\x00 the demo"}}

	// Without a login: refused.
	w := serve(s, http.MethodPost, "/ui/templates/vm/render", form, "HX-Request", "true")
	assert.Contains(t, w.Body.String(), "login required to order")
	assert.Empty(t, gh.Recorded())

	w = serve(s, http.MethodPost, "/ui/templates/vm/render", form, "HX-Request", "true", auth.HeaderForwardedUser, "patrick")
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "order as PR · dispatched")
	assert.Contains(t, body, gh.RunURL())
	assert.Contains(t, body, "requested by <strong>patrick</strong>")
	assert.Contains(t, body, `hx-get="/ui/order-pr/cma-`)
	got := gh.Recorded()
	require.Len(t, got, 1)
	assert.Equal(t, "patrick", got[0].Inputs["requested_by"])
	assert.Equal(t, "for  the demo", got[0].Inputs["reason"])
	assert.JSONEq(t, `{"vmName":"web1"}`, got[0].Inputs["params"], "empty fields are left to the defaults")
	assert.Equal(t, gh.RunURL(), expectNotification(t, sent).Url)

	orderID := got[0].Inputs["order_id"]
	poll := "/ui/order-pr/" + orderID + "?template=vm&run=9001&n=1"

	// Polling needs the login too.
	assert.Equal(t, http.StatusUnauthorized, serve(s, http.MethodGet, poll, nil).Code)

	w = serve(s, http.MethodGet, poll, nil, auth.HeaderForwardedUser, "patrick")
	assert.Contains(t, w.Body.String(), "order as PR · running")
	assert.Contains(t, w.Body.String(), "n=2")

	// Finished, PR not listed yet: keep polling.
	gh.Set(func(g *dispatchtest.GitHub) { g.RunStatus, g.RunConclusion = "completed", "success" })
	w = serve(s, http.MethodGet, poll, nil, auth.HeaderForwardedUser, "patrick")
	assert.Contains(t, w.Body.String(), "looking for its pull request")
	assert.Contains(t, w.Body.String(), "hx-get=")

	gh.Set(func(g *dispatchtest.GitHub) { g.PRBody = "Order ID: `" + orderID + "`" })
	w = serve(s, http.MethodGet, poll, nil, auth.HeaderForwardedUser, "patrick")
	body = w.Body.String()
	assert.Contains(t, body, "order as PR · pull request opened")
	assert.Contains(t, body, dispatchtest.PRURL)
	assert.NotContains(t, body, "hx-get=", "settled: no more polling")

	// Without the run id the run is found by the order id.
	w = serve(s, http.MethodGet, "/ui/order-pr/"+orderID+"?template=vm", nil, auth.HeaderForwardedUser, "patrick")
	assert.Contains(t, w.Body.String(), dispatchtest.PRURL)

	// No pull_requests:read: the run link stays, polling stops.
	gh.Set(func(g *dispatchtest.GitHub) { g.PRStatus = http.StatusForbidden })
	w = serve(s, http.MethodGet, poll, nil, auth.HeaderForwardedUser, "patrick")
	assert.Contains(t, w.Body.String(), "pull request link could not be read")
	assert.NotContains(t, w.Body.String(), "hx-get=")

	// A failed run is shown as such.
	gh.Set(func(g *dispatchtest.GitHub) { g.RunConclusion = "failure" })
	w = serve(s, http.MethodGet, poll, nil, auth.HeaderForwardedUser, "patrick")
	assert.Contains(t, w.Body.String(), "order as PR · failed")

	// The polling stops after a while.
	gh.Set(func(g *dispatchtest.GitHub) { g.RunStatus = "in_progress" })
	w = serve(s, http.MethodGet, "/ui/order-pr/"+orderID+"?template=vm&run=9001&n=120", nil, auth.HeaderForwardedUser, "patrick")
	assert.Contains(t, w.Body.String(), "Stopped watching")
	assert.NotContains(t, w.Body.String(), "hx-get=")

	// Only order ids are accepted.
	assert.Equal(t, http.StatusNotFound, serve(s, http.MethodGet, "/ui/order-pr/cma-XYZ", nil, auth.HeaderForwardedUser, "patrick").Code)
}

func TestCleanReason(t *testing.T) {
	assert.Equal(t, "a b", cleanReason("  a\nb "))
	long := make([]rune, 600)
	for i := range long {
		long[i] = 'x'
	}
	assert.Len(t, []rune(cleanReason(string(long))), maxReason)
}
