package claimtemplate_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"yourmodule/claimtemplate"
)

func TestLoadClaimTemplate(t *testing.T) {
	tmpl, err := claimtemplate.LoadClaimTemplate("testdata/volumeclaim.yaml")
	require.NoError(t, err)

	require.Equal(t, "ClaimTemplate", tmpl.Kind)
	require.Equal(t, "volumeclaim", tmpl.Metadata.Name)
	require.Equal(t, "oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim", tmpl.Spec.Source)

	require.Len(t, tmpl.Spec.Parameters, 4)

	param := tmpl.Spec.Parameters[0]
	require.Equal(t, "templateName", param.Name)
	require.Equal(t, "string", param.Type)
	require.Contains(t, param.Enum, "simple")
}
