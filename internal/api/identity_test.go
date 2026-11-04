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
				{Name: "owner", Type: "string", Default: "", Identity: claimtemplate.IdentityOwner},
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
