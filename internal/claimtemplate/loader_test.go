package claimtemplate_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
)

func TestLoadClaimTemplate(t *testing.T) {
	tmpl, err := claimtemplate.LoadClaimTemplate("testdata/volumeclaim.yaml")
	require.NoError(t, err)

	require.Equal(t, "ClaimTemplate", tmpl.Kind)
	require.Equal(t, "volumeclaim-simple", tmpl.Metadata.Name)
	require.Equal(t, "oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim", tmpl.Spec.Source)

	require.Len(t, tmpl.Spec.Parameters, 10)

	param := tmpl.Spec.Parameters[0]
	require.Equal(t, "templateName", param.Name)
	require.Equal(t, "string", param.Type)
	require.Contains(t, param.Enum, "simple")
}

func TestLoadClaimTemplate_Multiselect(t *testing.T) {
	tmpl, err := claimtemplate.LoadClaimTemplate("testdata/multiselect.yaml")
	require.NoError(t, err)

	require.Equal(t, "ClaimTemplate", tmpl.Kind)
	require.Equal(t, "multiselect-test", tmpl.Metadata.Name)
	require.Len(t, tmpl.Spec.Parameters, 2)

	// First param should have multiselect: true
	playbooks := tmpl.Spec.Parameters[0]
	require.Equal(t, "playbooks", playbooks.Name)
	require.True(t, playbooks.Multiselect)
	require.Equal(t, "array", playbooks.Type)
	require.Equal(t, []string{"baseos", "deploy-docker", "configure-network"}, playbooks.Enum)

	// Default should be parsed as a list
	defaults, ok := playbooks.Default.([]interface{})
	require.True(t, ok, "default should be []interface{}")
	require.Len(t, defaults, 1)
	require.Equal(t, "baseos", defaults[0])

	// Second param should not have multiselect
	name := tmpl.Spec.Parameters[1]
	require.Equal(t, "name", name.Name)
	require.False(t, name.Multiselect)
}
