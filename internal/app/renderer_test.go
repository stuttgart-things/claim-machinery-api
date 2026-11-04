package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/render"
)

// Only string values of `type: string` parameters are quoted (#127): the web
// UI sends booleans as "true"/"false", which kcl must still parse as bools.
func TestTypedParams(t *testing.T) {
	tmpl := &claimtemplate.ClaimTemplate{}
	tmpl.Spec.Parameters = []claimtemplate.Parameter{
		{Name: "version", Type: "string"},
		{Name: "enabled", Type: "boolean"},
		{Name: "count", Type: "integer"},
		{Name: "features", Type: "array", Multiselect: true},
		{Name: "untyped"},
	}
	in := map[string]interface{}{
		"version":  "1.30",
		"enabled":  "true",
		"count":    "3",
		"features": []string{"go"},
		"untyped":  "007",
		"extra":    "1.5", // not declared in the template
	}
	out := TypedParams(tmpl, in)

	assert.Equal(t, render.QuotedString("1.30"), out["version"])
	assert.Equal(t, "true", out["enabled"])
	assert.Equal(t, "3", out["count"])
	assert.Equal(t, []string{"go"}, out["features"])
	assert.Equal(t, "007", out["untyped"])
	assert.Equal(t, "1.5", out["extra"])
	// the input map is not modified
	assert.Equal(t, "1.30", in["version"])
}

// A non-string value of a string parameter (e.g. a JSON number) is left alone.
func TestTypedParams_NonStringValue(t *testing.T) {
	tmpl := &claimtemplate.ClaimTemplate{}
	tmpl.Spec.Parameters = []claimtemplate.Parameter{{Name: "n", Type: "string"}}
	out := TypedParams(tmpl, map[string]interface{}{"n": 4.0})
	assert.Equal(t, 4.0, out["n"])
}
