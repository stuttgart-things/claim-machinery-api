package api

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stuttgart-things/claim-machinery-api/internal/auth"
	"github.com/stuttgart-things/claim-machinery-api/internal/auth/authtest"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/notify"
)

// identityTemplate is a VM template like xr-native-vm's: requested_by bound
// to the login (hidden), owner selectable, limited to two roles.
func identityTemplate() *claimtemplate.ClaimTemplate {
	return &claimtemplate.ClaimTemplate{
		Metadata: claimtemplate.ClaimTemplateMetadata{Name: "vm-id"},
		Spec: claimtemplate.ClaimTemplateSpec{
			Source: "oci://example/vm", Tag: "1.0.0",
			OrderRoles: []string{"backstage-developer", "backstage-team-sre"},
			Parameters: []claimtemplate.Parameter{
				{Name: "name", Type: "string", Default: "demo"},
				{Name: "owner", Type: "string", Default: "", Identity: claimtemplate.IdentityOwner,
					Description: "Backstage entity ref of the owner; empty = the orderer"},
				{Name: "requested_by", Type: "string", Default: "", Hidden: true, Identity: claimtemplate.IdentityUser},
			},
		},
	}
}

func newIdentityServer(t *testing.T, mod func(*auth.Config)) (*Server, *authtest.Issuer, *map[string]interface{}) {
	t.Helper()
	iss := authtest.NewIssuer(t)
	s, err := NewServerWithTemplates([]*claimtemplate.ClaimTemplate{identityTemplate()})
	require.NoError(t, err)
	var got map[string]interface{}
	s.render = func(_ *claimtemplate.ClaimTemplate, params map[string]interface{}) (string, error) {
		got = params
		return "kind: VM\n", nil
	}
	cfg := auth.Config{Issuer: iss.URL(), Audiences: []string{orderAudience}, CAFile: iss.CAFile,
		AdminRoles: []string{"backstage-platform"}}
	if mod != nil {
		mod(&cfg)
	}
	a, err := auth.New(cfg)
	require.NoError(t, err)
	s.SetAuth(a)
	return s, iss, &got
}

// token signs a token for patrick with the given ZITADEL roles.
func token(t *testing.T, iss *authtest.Issuer, roles ...string) string {
	c := iss.Claims(orderAudience)
	r := map[string]any{}
	for _, role := range roles {
		r[role] = map[string]any{"1": "org"}
	}
	c[auth.ZitadelRolesClaim] = r
	return "Bearer " + iss.Sign(t, c)
}

const idPath = "/api/v1/claim-templates/vm-id/order"

func TestIdentityStamped(t *testing.T) {
	s, iss, rendered := newIdentityServer(t, nil)
	tok := token(t, iss, "backstage-developer")

	// Neither sent: requested_by and owner are the login.
	w := apiOrder(s, idPath, `{"parameters":{"name":"vm1"}}`, "Authorization", tok)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "user:default/patrick", (*rendered)["requested_by"])
	assert.Equal(t, "user:default/patrick", (*rendered)["owner"])

	// The login's own value is accepted.
	w = apiOrder(s, idPath, `{"parameters":{"requested_by":"user:default/patrick"}}`, "Authorization", tok)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestIdentityForgedRequesterRefused(t *testing.T) {
	s, iss, rendered := newIdentityServer(t, nil)
	for _, body := range []string{
		`{"parameters":{"requested_by":"user:default/someone-else"}}`,
		`{"parameters":{"requested_by":["user:default/patrick"]}}`,
	} {
		*rendered = nil
		w := apiOrder(s, idPath, body, "Authorization", token(t, iss, "backstage-platform"))
		assert.Equal(t, http.StatusBadRequest, w.Code, body)
		assert.Contains(t, decodeError(t, w), "requested_by", body)
		assert.Nil(t, *rendered, "nothing rendered for %s", body)
	}
}

func TestIdentityOwner(t *testing.T) {
	s, iss, rendered := newIdentityServer(t, nil)
	for _, c := range []struct {
		name   string
		owner  string
		roles  []string
		status int
	}{
		{"own ref", "user:default/patrick", []string{"backstage-developer"}, http.StatusOK},
		{"own team", "group:default/sre", []string{"backstage-team-sre"}, http.StatusOK},
		{"default group", "group:default/guests", []string{"backstage-developer"}, http.StatusOK},
		{"other team", "group:default/platform-team", []string{"backstage-developer"}, http.StatusForbidden},
		{"other user", "user:default/someone-else", []string{"backstage-team-sre"}, http.StatusForbidden},
		{"admin, anyone", "user:default/someone-else", []string{"backstage-platform"}, http.StatusOK},
	} {
		t.Run(c.name, func(t *testing.T) {
			*rendered = nil
			w := apiOrder(s, idPath, `{"parameters":{"owner":"`+c.owner+`"}}`, "Authorization", token(t, iss, c.roles...))
			require.Equal(t, c.status, w.Code, w.Body.String())
			if c.status == http.StatusOK {
				assert.Equal(t, c.owner, (*rendered)["owner"])
				assert.Equal(t, "user:default/patrick", (*rendered)["requested_by"])
			} else {
				assert.Nil(t, *rendered)
			}
		})
	}
}

func TestIdentityTemplateRoles(t *testing.T) {
	s, iss, rendered := newIdentityServer(t, nil)
	w := apiOrder(s, idPath, `{}`, "Authorization", token(t, iss, "backstage-beta-vm"))
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "forbidden: missing role for this template", decodeError(t, w))
	assert.Nil(t, *rendered)

	for _, roles := range [][]string{{"backstage-team-sre"}, {"backstage-platform"}} {
		w = apiOrder(s, idPath, `{}`, "Authorization", token(t, iss, roles...))
		assert.Equal(t, http.StatusOK, w.Code, roles)
	}
}

func TestIdentityBadLoginName(t *testing.T) {
	s, iss, _ := newIdentityServer(t, nil)
	c := iss.Claims(orderAudience)
	c["preferred_username"] = "not a name"
	c[auth.ZitadelRolesClaim] = map[string]any{"backstage-developer": map[string]any{"1": "org"}}
	w := apiOrder(s, idPath, `{}`, "Authorization", "Bearer "+iss.Sign(t, c))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Auth off: identity parameters are ordinary parameters, as before.
func TestIdentityAuthOff(t *testing.T) {
	s, err := NewServerWithTemplates([]*claimtemplate.ClaimTemplate{identityTemplate()})
	require.NoError(t, err)
	var got map[string]interface{}
	s.render = func(_ *claimtemplate.ClaimTemplate, params map[string]interface{}) (string, error) {
		got = params
		return "kind: VM\n", nil
	}
	w := apiOrder(s, idPath, `{"parameters":{"requested_by":"user:default/anyone","owner":"group:default/x"}}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "user:default/anyone", got["requested_by"])
	assert.Equal(t, "group:default/x", got["owner"])
}

// The web UI's order through sthings-portal: the forwarded user and groups
// are bound the same way; the hidden requested_by never comes from the form.
func TestIdentityUIForwarded(t *testing.T) {
	s, _, rendered := newIdentityServer(t, func(c *auth.Config) { c.TrustForwardedUser = true })
	order := func(owner string, groups string) int {
		*rendered = nil
		form := url.Values{"_action": {"order"}, "owner": {owner}, "requested_by": {"user:default/forged"}}
		return serve(s, http.MethodPost, "/ui/templates/vm-id/render", form,
			"X-Forwarded-User", "patrick@sthings.example", "X-Forwarded-Groups", groups).Code
	}
	require.Equal(t, http.StatusOK, order("", "backstage-developer"))
	assert.Equal(t, "user:default/patrick", (*rendered)["requested_by"])
	assert.Equal(t, "user:default/patrick", (*rendered)["owner"])

	assert.Equal(t, http.StatusForbidden, order("group:default/platform-team", "backstage-developer"))
	assert.Nil(t, *rendered)

	assert.Equal(t, http.StatusForbidden, order("", "backstage-beta-vm"), "template roles hold for the UI too")
}

// The order form names the login, offers the owners it may pick and does
// not count requested_by as a platform default (#138).
func TestIdentityUIForm(t *testing.T) {
	s, _, _ := newIdentityServer(t, func(c *auth.Config) { c.TrustForwardedUser = true })
	form := func(groups string) string {
		return serve(s, http.MethodGet, "/ui/templates/vm-id", nil,
			"X-Forwarded-User", "patrick@sthings.example", "X-Forwarded-Groups", groups).Body.String()
	}

	body := form("backstage-developer")
	assert.Contains(t, body, "Ordering as <strong>user:default/patrick</strong>")
	assert.Contains(t, body, "requested_by is set from it")
	assert.NotContains(t, body, "platform parameter")
	assert.Contains(t, body, `<option value="">you (user:default/patrick)</option>`)
	assert.Contains(t, body, `<option value="group:default/guests">`)
	assert.NotContains(t, body, `list="owners-owner"`, "no free owner without the admin role")
	assert.Contains(t, body, `hx-disabled-elt="find .actions button"`)

	body = form("backstage-platform")
	assert.Contains(t, body, `list="owners-owner"`, "admins may type any owner")
	assert.Contains(t, body, "empty: you (user:default/patrick)")

	body = serve(s, http.MethodGet, "/ui/templates/vm-id", nil).Body.String()
	assert.Contains(t, body, "Ordering needs a login")
	assert.Contains(t, body, `<input id="p-owner" name="owner" type="text"`, "without a login the owner stays a text field")
}

// A preview with a login renders what the order would; without one it says
// so (#138).
func TestIdentityUIPreview(t *testing.T) {
	s, _, rendered := newIdentityServer(t, func(c *auth.Config) { c.TrustForwardedUser = true })
	preview := func(owner, groups string, header ...string) string {
		*rendered = nil
		form := url.Values{"_action": {"preview"}, "owner": {owner}}
		if groups != "" {
			header = append(header, "X-Forwarded-User", "patrick", "X-Forwarded-Groups", groups)
		}
		return serve(s, http.MethodPost, "/ui/templates/vm-id/render", form, header...).Body.String()
	}

	body := preview("", "backstage-developer")
	assert.Equal(t, "user:default/patrick", (*rendered)["requested_by"])
	assert.Equal(t, "user:default/patrick", (*rendered)["owner"])
	assert.Contains(t, body, "preview · no notification")
	assert.Contains(t, body, "as <code>user:default/patrick</code>, owner <code>user:default/patrick</code>")

	body = preview("group:default/platform-team", "backstage-developer")
	assert.Contains(t, body, "An order would be refused: forbidden:")
	assert.Equal(t, "group:default/platform-team", (*rendered)["owner"], "the refused preview stays unbound")

	body = preview("", "")
	assert.Contains(t, body, "Owner and requested-by are set from your login when you order.")
	assert.Empty(t, (*rendered)["requested_by"], "unbound: the empty default")
}

// An order confirms who ordered what and replaces the buttons, so one click
// is one order (#138).
func TestIdentityUIOrderConfirmation(t *testing.T) {
	s, _, _ := newIdentityServer(t, func(c *auth.Config) { c.TrustForwardedUser = true })
	body := serve(s, http.MethodPost, "/ui/templates/vm-id/render",
		url.Values{"_action": {"order"}, "name": {"web-7"}},
		"X-Forwarded-User", "patrick", "X-Forwarded-Groups", "backstage-developer").Body.String()
	assert.Contains(t, body, `<div class="ordered-box" role="status">`)
	assert.Contains(t, body, "Ordered: vm-id <code>web-7</code>")
	assert.Contains(t, body, "Requested by <code>user:default/patrick</code>, owner <code>user:default/patrick</code>.")
	assert.Contains(t, body, `id="order-actions" hx-swap-oob="true"`)
	assert.Contains(t, body, `<a href="/ui/templates/vm-id">new order</a>`)
	assert.NotContains(t, body, `value="order"`, "the replaced actions carry no order button")
}

// The owner widget shows its own hint, not the template's description, which
// is written for a text field (#140). Without a login the owner is a text
// field and keeps it.
func TestIdentityUIOwnerHint(t *testing.T) {
	s, _, _ := newIdentityServer(t, func(c *auth.Config) { c.TrustForwardedUser = true })
	body := serve(s, http.MethodGet, "/ui/templates/vm-id", nil,
		"X-Forwarded-User", "patrick@sthings.example", "X-Forwarded-Groups", "backstage-developer").Body.String()
	assert.Contains(t, body, "You or one of your groups.")
	assert.NotContains(t, body, "empty = the orderer")

	body = serve(s, http.MethodGet, "/ui/templates/vm-id", nil).Body.String()
	assert.Contains(t, body, "empty = the orderer")
}

// The form says whether Order notifies, and drops the Author field (only
// used in the notification) when notifications are off (#140).
func TestUINotifyHint(t *testing.T) {
	s, _, _ := newIdentityServer(t, nil)
	body := serve(s, http.MethodGet, "/ui/templates/vm-id", nil).Body.String()
	assert.Contains(t, body, "Order renders it; notifications are off.")
	assert.NotContains(t, body, "sends the homerun notification")
	assert.NotContains(t, body, `id="p-_author"`)

	s.SetHomerunConfig(notify.HomerunConfig{Enabled: true, URL: "http://homerun.invalid"})
	body = serve(s, http.MethodGet, "/ui/templates/vm-id", nil).Body.String()
	assert.Contains(t, body, "Order renders and sends the homerun notification.")
	assert.Contains(t, body, `id="p-_author"`)
}
