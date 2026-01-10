package app

import (
	"fmt"
	"log"

	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/render"
)

// BuildParameterValues creates a map of parameter values from a template
// Uses default values where available
func BuildParameterValues(t *claimtemplate.ClaimTemplate) map[string]interface{} {
	params := make(map[string]interface{})

	for _, p := range t.Spec.Parameters {
		// Use default value if available, otherwise use a reasonable default
		if p.Default != nil {
			params[p.Name] = p.Default
		} else {
			// Provide reasonable defaults based on type
			switch p.Type {
			case "string":
				params[p.Name] = ""
			case "boolean":
				params[p.Name] = false
			case "number":
				params[p.Name] = 0
			case "array":
				params[p.Name] = []interface{}{}
			default:
				params[p.Name] = nil
			}
		}
	}

	return params
}

// RenderTemplate renders a claim template using KCL
func RenderTemplate(t *claimtemplate.ClaimTemplate) (string, error) {
	// Build parameter values from template defaults
	params := BuildParameterValues(t)

	// Render using KCL from OCI source
	result := render.RenderKCLFromOCI(t.Spec.Source, t.Spec.Tag, params)

	if result == "" {
		return "", fmt.Errorf("rendering produced empty result for template %s", t.Metadata.Name)
	}

	return result, nil
}

// RenderTemplateToFile renders a template and saves to file
func RenderTemplateToFile(t *claimtemplate.ClaimTemplate, destination string) (string, error) {
	// Build parameter values from template defaults
	params := BuildParameterValues(t)

	// Render using KCL from OCI source
	result, err := render.RenderKCLFromOCIToFile(t.Spec.Source, t.Spec.Tag, params, destination)
	if err != nil {
		log.Printf("⚠️  failed to render template %s: %v", t.Metadata.Name, err)
		return result, err
	}

	return result, nil
}
