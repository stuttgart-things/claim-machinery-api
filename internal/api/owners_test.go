package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stuttgart-things/claim-machinery-api/internal/auth"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
)

func apiGet(s *Server, path string, header ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

func ownerParam(t *testing.T, w *httptest.ResponseRecorder) claimtemplate.Parameter {
	t.Helper()
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var tmpl claimtemplate.ClaimTemplate
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tmpl))
	for _, p := range tmpl.Spec.Parameters {
		if p.Name == "owner" {
			return p
		}
	}
	t.Fatal("no owner parameter")
	return claimtemplate.Parameter{}
}

// #147: the template detail lists the caller's allowed owners.
func TestTemplateDetailAllowedOwners(t *testing.T) {
	s, iss, _ := newIdentityServer(t, nil)
	const path = "/api/v1/claim-templates/vm-id"

	p := ownerParam(t, apiGet(s, path, "Authorization", token(t, iss, "backstage-developer", "backstage-team-ai-personas")))
	assert.Equal(t, []string{"user:default/patrick", "group:default/ai-personas", "group:default/guests"}, p.AllowedOwners)
	assert.False(t, p.AnyOwner)

	p = ownerParam(t, apiGet(s, path, "Authorization", token(t, iss, "backstage-platform")))
	assert.Equal(t, []string{"user:default/patrick", "group:default/platform-team"}, p.AllowedOwners)
	assert.True(t, p.AnyOwner)

	// No token or a bad one: the plain template, still 200.
	assert.Empty(t, ownerParam(t, apiGet(s, path)).AllowedOwners)
	assert.Empty(t, ownerParam(t, apiGet(s, path, "Authorization", "Bearer nope")).AllowedOwners)

	// The stored template is not changed.
	assert.Empty(t, s.getTemplateMap()["vm-id"].Spec.Parameters[1].AllowedOwners)
}

func TestOwnerForbiddenNamesAllowed(t *testing.T) {
	s, iss, _ := newIdentityServer(t, nil)
	w := apiOrder(s, idPath, `{"parameters":{"owner":"group:default/platform-team"}}`,
		"Authorization", token(t, iss, "backstage-developer", "backstage-team-ai-personas"))
	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, `forbidden: owner "group:default/platform-team" is neither the login nor one of its groups; `+
		`allowed: user:default/patrick, group:default/ai-personas, group:default/guests`, decodeError(t, w))
}

func TestOwnerBareGroupName(t *testing.T) {
	s, iss, rendered := newIdentityServer(t, nil)
	tok := token(t, iss, "backstage-developer", "backstage-team-ai-personas")

	w := apiOrder(s, idPath, `{"parameters":{"owner":"ai-personas"}}`, "Authorization", tok)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "group:default/ai-personas", (*rendered)["owner"])

	*rendered = nil
	w = apiOrder(s, idPath, `{"parameters":{"owner":"platform-team"}}`, "Authorization", tok)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Nil(t, *rendered)
}

func TestBareGroup(t *testing.T) {
	groups := []string{"group:default/ai-personas", "group:default/sre", "group:other/sre"}
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{"ai-personas", "group:default/ai-personas", true},
		{"sre", "", false}, // two match
		{"personas", "", false},
		{"group:default/ai-personas", "", false},
		{"default/ai-personas", "", false},
		{"", "", false},
	} {
		got, ok := bareGroup(groups, c.in)
		assert.Equal(t, c.ok, ok, c.in)
		assert.Equal(t, c.want, got, c.in)
	}
}

func TestWhoami(t *testing.T) {
	s, iss, _ := newIdentityServer(t, func(c *auth.Config) { c.OrderRoles = []string{"backstage-developer"} })

	w := apiGet(s, "/api/v1/whoami", "Authorization", token(t, iss, "backstage-developer", "backstage-team-ai-personas"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var got WhoamiResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "Identity", got.Kind)
	assert.Equal(t, "user:default/patrick", got.UserRef)
	assert.Equal(t, []string{"group:default/ai-personas", "group:default/guests"}, got.Groups)
	assert.True(t, got.MayOrder)
	assert.False(t, got.Admin)

	// Without an order role: still answered, mayOrder false.
	w = apiGet(s, "/api/v1/whoami", "Authorization", token(t, iss, "backstage-team-sre"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.False(t, got.MayOrder)
	assert.Equal(t, []string{"group:default/sre"}, got.Groups)

	w = apiGet(s, "/api/v1/whoami")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NotEmpty(t, w.Header().Get("WWW-Authenticate"))
}

func TestWhoamiAuthOff(t *testing.T) {
	s, err := NewServerWithTemplates([]*claimtemplate.ClaimTemplate{identityTemplate()})
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, apiGet(s, "/api/v1/whoami").Code)
}
