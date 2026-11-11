package claimtemplate

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// vmTemplateYAML follows xr-native-vm's native-vm-* templates (kcl#396).
const vmTemplateYAML = `
apiVersion: resources.stuttgart-things.com/v1alpha1
kind: ClaimTemplate
metadata:
  name: native-vm
spec:
  type: xr-native-vm
  source: oci://example/xr-native-vm
  parameters:
    - name: name
      type: string
      required: true
      pattern: "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
    - name: use_case
      type: string
      default: baseos
      enum: [baseos, kind, dev, rke2, github-runner]
    - name: size
      type: string
      default: medium
      enum: [small, medium, large, xlarge]
      enumWhen:
        use_case:
          dev: [large, xlarge]
          github-runner: [large, xlarge]
          rke2: [medium, large, xlarge]
    - name: kind_workers
      type: string
      default: "1"
      showWhen:
        use_case: [kind]
    - name: dev_features
      type: array
      multiselect: true
      enum: [docker, go, kind]
      default: [docker]
      showWhen:
        use_case: [dev]
    - name: dev_kind_workers
      type: string
      default: "3"
      showWhen:
        dev_features: [kind]
    - name: runner_toolchain
      type: boolean
      default: true
      showWhen:
        use_case: [github-runner]
    - name: runner_extra
      type: string
      showWhen:
        runner_toolchain: [true]
`

func vmTemplate(t *testing.T) *ClaimTemplate {
	t.Helper()
	var tmpl ClaimTemplate
	require.NoError(t, yaml.Unmarshal([]byte(vmTemplateYAML), &tmpl))
	return &tmpl
}

func fieldsOf(t *testing.T, err error) FieldErrors {
	t.Helper()
	var verr *ValidationError
	require.True(t, errors.As(err, &verr), "want a ValidationError, got %v", err)
	return verr.Fields
}

func TestConditionsDecode(t *testing.T) {
	tmpl := vmTemplate(t)
	size := tmpl.Spec.Parameters[2]
	assert.Equal(t, []string{"large", "xlarge"}, size.EnumWhen["use_case"]["dev"])
	assert.Equal(t, map[string][]string{"use_case": {"kind"}}, tmpl.Spec.Parameters[3].ShowWhen)
	// A YAML boolean in showWhen decodes as the string "true".
	assert.Equal(t, []string{"true"}, tmpl.Spec.Parameters[7].ShowWhen["runner_toolchain"])
	assert.Empty(t, tmpl.CheckConditions())
}

func TestActiveParams(t *testing.T) {
	tmpl := vmTemplate(t)

	active, values := ActiveParams(tmpl, map[string]interface{}{"name": "vm1"})
	assert.False(t, active["kind_workers"])
	assert.False(t, active["dev_features"])
	assert.False(t, active["dev_kind_workers"], "chain: dev_features inactive, so no value")
	assert.False(t, active["runner_toolchain"])
	assert.False(t, active["runner_extra"])
	assert.Equal(t, "baseos", values["use_case"], "default fills in")
	assert.NotContains(t, values, "kind_workers")

	active, _ = ActiveParams(tmpl, map[string]interface{}{"use_case": "dev", "dev_features": []interface{}{"go", "kind"}})
	assert.True(t, active["dev_features"])
	assert.True(t, active["dev_kind_workers"], "multiselect matches one of its values")

	active, _ = ActiveParams(tmpl, map[string]interface{}{"use_case": "dev"})
	assert.False(t, active["dev_kind_workers"], "default [docker] has no kind")

	active, _ = ActiveParams(tmpl, map[string]interface{}{"use_case": "github-runner"})
	assert.True(t, active["runner_toolchain"])
	assert.True(t, active["runner_extra"], "boolean default true matches \"true\"")
	active, _ = ActiveParams(tmpl, map[string]interface{}{"use_case": "github-runner", "runner_toolchain": false})
	assert.False(t, active["runner_extra"])
}

func TestAllowedValues(t *testing.T) {
	size := vmTemplate(t).Spec.Parameters[2]
	assert.Equal(t, []string{"large", "xlarge"}, size.AllowedValues(map[string]interface{}{"use_case": "dev"}))
	assert.Equal(t, []string{"small", "medium", "large", "xlarge"}, size.AllowedValues(map[string]interface{}{"use_case": "kind"}), "unlisted value: enum")
	assert.Nil(t, Parameter{Name: "free"}.AllowedValues(nil))

	// Two controlling parameters intersect; no enum takes the list as is.
	p := Parameter{EnumWhen: map[string]map[string][]string{
		"a": {"x": {"1", "2", "3"}},
		"b": {"y": {"2", "3", "4"}},
	}}
	assert.Equal(t, []string{"2", "3"}, p.AllowedValues(map[string]interface{}{"a": "x", "b": "y"}))
	assert.Equal(t, []string{"2", "3", "4"}, p.AllowedValues(map[string]interface{}{"b": "y"}))
}

func TestCheckParams(t *testing.T) {
	tmpl := vmTemplate(t)

	// Values of inactive parameters are dropped, others kept as sent.
	got, err := CheckParams(tmpl, map[string]interface{}{
		"name": "vm1", "use_case": "baseos", "size": "small",
		"kind_workers": "5", "dev_features": []interface{}{"go"}, "runner_toolchain": false, "extra": "x",
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"name": "vm1", "use_case": "baseos", "size": "small", "extra": "x"}, got)

	got, err = CheckParams(tmpl, map[string]interface{}{"name": "vm1", "use_case": "kind", "kind_workers": "2"})
	require.NoError(t, err)
	assert.Equal(t, "2", got["kind_workers"])

	_, err = CheckParams(tmpl, map[string]interface{}{"name": "vm1", "use_case": "dev", "size": "small"})
	assert.Equal(t, FieldErrors{"size": "use_case dev allows large or xlarge, got small"}, fieldsOf(t, err))
	assert.Contains(t, err.Error(), "invalid parameters: size:")

	// The default of an enumWhen parameter is checked too.
	_, err = CheckParams(tmpl, map[string]interface{}{"name": "vm1", "use_case": "github-runner"})
	assert.Equal(t, FieldErrors{"size": "use_case github-runner allows large or xlarge, got medium (default)"}, fieldsOf(t, err))

	_, err = CheckParams(tmpl, map[string]interface{}{"name": "vm1", "use_case": "rke2", "size": "small"})
	assert.Equal(t, "use_case rke2 allows medium, large or xlarge, got small", fieldsOf(t, err)["size"])
	_, err = CheckParams(tmpl, map[string]interface{}{"name": "vm1", "use_case": "rke2", "size": "medium"})
	assert.NoError(t, err)

	// required, enum, pattern, boolean, multiselect items.
	_, err = CheckParams(tmpl, map[string]interface{}{
		"use_case": "nope", "dev_features": []interface{}{"evil"},
	})
	f := fieldsOf(t, err)
	assert.Equal(t, "is required", f["name"])
	assert.Equal(t, "must be one of baseos, kind, dev, rke2, github-runner, got nope", f["use_case"])
	assert.NotContains(t, f, "dev_features", "inactive: use_case nope is not dev")

	_, err = CheckParams(tmpl, map[string]interface{}{
		"name": "Bad Name", "use_case": "dev", "size": "large", "dev_features": []interface{}{"go", "evil"},
	})
	f = fieldsOf(t, err)
	assert.Contains(t, f["name"], "does not match")
	assert.Equal(t, "must be one of docker, go, kind, got evil", f["dev_features"])

	_, err = CheckParams(tmpl, map[string]interface{}{"name": "vm1", "use_case": "github-runner", "size": "large", "runner_toolchain": "yes"})
	assert.Equal(t, "must be true or false", fieldsOf(t, err)["runner_toolchain"])

	// Empty strings are left to the default / KCL, as before.
	_, err = CheckParams(tmpl, map[string]interface{}{"name": "vm1", "size": ""})
	assert.NoError(t, err)
}

// A template without conditions keeps today's behaviour for valid input.
func TestCheckParamsWithoutConditions(t *testing.T) {
	tmpl := &ClaimTemplate{Spec: ClaimTemplateSpec{Parameters: []Parameter{
		{Name: "a", Type: "string", Default: "x"},
		{Name: "b", Type: "string", Required: true, ValueFrom: &ValueFromSpec{Function: "f"}},
	}}}
	in := map[string]interface{}{"a": "y", "other": 1}
	got, err := CheckParams(tmpl, in)
	require.NoError(t, err)
	assert.Equal(t, in, got)
}

func TestCheckConditions(t *testing.T) {
	tmpl := &ClaimTemplate{Spec: ClaimTemplateSpec{Parameters: []Parameter{
		{Name: "use_case", Enum: []string{"dev"}},
		{Name: "size", Enum: []string{"small", "large"},
			ShowWhen: map[string][]string{"usecase": {"dev"}},
			EnumWhen: map[string]map[string][]string{"use_case": {"dev": {"large", "huge"}}}},
	}}}
	assert.Equal(t, []string{
		"parameter size: enumWhen use_case=dev allows huge, which is not in its enum",
		"parameter size: showWhen names unknown parameter usecase",
	}, tmpl.CheckConditions())
}
