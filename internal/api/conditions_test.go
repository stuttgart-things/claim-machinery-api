package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/render"
)

// condTemplate is a VM template with parameter conditions (#135), like
// xr-native-vm's.
func condTemplate() *claimtemplate.ClaimTemplate {
	return &claimtemplate.ClaimTemplate{
		Metadata: claimtemplate.ClaimTemplateMetadata{Name: "cvm"},
		Spec: claimtemplate.ClaimTemplateSpec{
			Source: "oci://example/vm", Tag: "1.0.0",
			Parameters: []claimtemplate.Parameter{
				{Name: "vmName", Type: "string", Required: true, Pattern: "^[a-z][a-z0-9-]*$"},
				{Name: "use_case", Type: "string", Default: "baseos", Enum: []string{"baseos", "kind", "dev"}},
				{Name: "size", Type: "string", Default: "medium", Enum: []string{"small", "medium", "large"},
					EnumWhen: map[string]map[string][]string{"use_case": {"dev": {"large"}}}},
				{Name: "kind_workers", Type: "string", Default: "1", ShowWhen: map[string][]string{"use_case": {"kind"}}},
			},
		},
	}
}

func newCondServer(t *testing.T) (*Server, *map[string]interface{}) {
	t.Helper()
	s, got := newUITestServer(t, "")
	s.UpdateTemplates([]*claimtemplate.ClaimTemplate{condTemplate(), uiTestTemplate()})
	return s, got
}

const condPath = "/api/v1/claim-templates/cvm/order"

func decodeParamError(t *testing.T, body []byte) ParamErrorResponse {
	t.Helper()
	var resp ParamErrorResponse
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func TestGetTemplateExposesConditions(t *testing.T) {
	s, _ := newCondServer(t)
	w := serve(s, http.MethodGet, "/api/v1/claim-templates/cvm", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var tmpl claimtemplate.ClaimTemplate
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tmpl))
	assert.Equal(t, []string{"large"}, tmpl.Spec.Parameters[2].EnumWhen["use_case"]["dev"])
	assert.Equal(t, []string{"kind"}, tmpl.Spec.Parameters[3].ShowWhen["use_case"])
	assert.NotContains(t, w.Body.String(), `"showWhen":null`)

	// Parameters without conditions do not carry the fields.
	w = serve(s, http.MethodGet, "/api/v1/claim-templates/vm", nil)
	assert.NotContains(t, w.Body.String(), "showWhen")
	assert.NotContains(t, w.Body.String(), "enumWhen")
}

func TestOrderInvalidParameters(t *testing.T) {
	s, got := newCondServer(t)

	w := apiOrder(s, condPath, `{"parameters":{"vmName":"Bad","use_case":"dev","size":"small"}}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := decodeParamError(t, w.Body.Bytes())
	assert.Equal(t, "invalid parameters", resp.Error)
	assert.Equal(t, claimtemplate.FieldErrors{
		"vmName": "does not match ^[a-z][a-z0-9-]*$",
		"size":   "use_case dev allows only large, got small",
	}, resp.Fields)
	assert.Nil(t, *got, "KCL is not called")

	w = apiOrder(s, condPath, `{"parameters":{}}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "is required", decodeParamError(t, w.Body.Bytes()).Fields["vmName"])

	// A value for an inactive parameter is dropped; its default still goes
	// to KCL, as when a client never sent it.
	w = apiOrder(s, condPath, `{"parameters":{"vmName":"web1","kind_workers":"7"}}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "1", (*got)["kind_workers"])

	w = apiOrder(s, condPath, `{"parameters":{"vmName":"web1","use_case":"kind","kind_workers":"7"}}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "7", (*got)["kind_workers"])
}

func TestOrderKCLAssertIs400(t *testing.T) {
	s, _ := newCondServer(t)

	// What app.RenderTemplateFromParams returns for a failed module assert.
	s.render = func(*claimtemplate.ClaimTemplate, map[string]interface{}) (string, error) {
		return "", fmt.Errorf("rendering failed for template cvm: %w",
			&render.AssertError{Message: "use_case dev needs size large or xlarge, got small", Stderr: "EvaluationError ..."})
	}
	w := apiOrder(s, condPath, `{"parameters":{"vmName":"web1"}}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"use_case dev needs size large or xlarge, got small"}`, w.Body.String())

	// Any other render failure stays 500.
	s.render = func(*claimtemplate.ClaimTemplate, map[string]interface{}) (string, error) {
		return "", errors.New("KCL execution from OCI source failed: pull failed")
	}
	w = apiOrder(s, condPath, `{"parameters":{"vmName":"web1"}}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, decodeError(t, w), "pull failed")

	// The UI shows only the assert message as well.
	s.render = func(*claimtemplate.ClaimTemplate, map[string]interface{}) (string, error) {
		return "", fmt.Errorf("wrapped: %w", &render.AssertError{Message: "use_case rke2 needs size medium or larger"})
	}
	body := serve(s, http.MethodPost, "/ui/templates/cvm/render", url.Values{"vmName": {"web1"}}).Body.String()
	assert.Contains(t, body, "use_case rke2 needs size medium or larger")
	assert.NotContains(t, body, "wrapped")
}

func TestUIConditions(t *testing.T) {
	s, got := newCondServer(t)

	body := serve(s, http.MethodGet, "/ui/templates/cvm", nil).Body.String()
	assert.Contains(t, body, `data-conditions="`)
	assert.Contains(t, body, `data-param="kind_workers"`)
	assert.Contains(t, body, "&#34;showWhen&#34;:{&#34;use_case&#34;:[&#34;kind&#34;]}")
	// Templates without conditions get no condition data.
	assert.NotContains(t, serve(s, http.MethodGet, "/ui/templates/vm", nil).Body.String(), "data-conditions")

	// The form's value for an inactive parameter is not taken.
	serve(s, http.MethodPost, "/ui/templates/cvm/render", url.Values{"vmName": {"web1"}, "kind_workers": {"9"}, "size": {"small"}})
	require.NotNil(t, *got)
	assert.Equal(t, "1", (*got)["kind_workers"])
	assert.Equal(t, "small", (*got)["size"])

	// enumWhen narrows the form's enum check.
	*got = nil
	body = serve(s, http.MethodPost, "/ui/templates/cvm/render", url.Values{"vmName": {"web1"}, "use_case": {"dev"}, "size": {"small"}}).Body.String()
	assert.Contains(t, body, "size use_case dev allows only large, got small")
	assert.Nil(t, *got)

	// The default checked against enumWhen (size medium, use_case dev).
	body = serve(s, http.MethodPost, "/ui/templates/cvm/render", url.Values{"vmName": {"web1"}, "use_case": {"dev"}}).Body.String()
	assert.Contains(t, body, "invalid parameters: size: use_case dev allows only large, got medium (default)")
}

func TestOrderPRInvalidParameters(t *testing.T) {
	s, iss, gh, _ := newOrderPRServer(t, nil)
	s.UpdateTemplates([]*claimtemplate.ClaimTemplate{condTemplate()})
	w := apiOrder(s, "/api/v1/claim-templates/cvm/order-pr", `{"parameters":{"vmName":"web1","use_case":"dev","size":"small"}}`,
		"Authorization", bearer(t, iss))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := decodeParamError(t, w.Body.Bytes())
	assert.Equal(t, "invalid parameters", resp.Error)
	assert.Contains(t, resp.Fields, "size")
	assert.Empty(t, gh.Recorded())
}
