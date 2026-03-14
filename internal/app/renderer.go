package app

import (
	"fmt"
	"log"

	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/functions"
	"github.com/stuttgart-things/claim-machinery-api/internal/render"
)

// BuildParameterValues creates a map of parameter values from a template.
// Uses default values where available.
func BuildParameterValues(t *claimtemplate.ClaimTemplate) map[string]interface{} {
	return BuildParameterValuesWithFunctions(t, nil)
}

// BuildParameterValuesWithFunctions creates a map of parameter values from a template.
// Parameters with a valueFrom spec are resolved via the function registry.
func BuildParameterValuesWithFunctions(t *claimtemplate.ClaimTemplate, reg *functions.Registry) map[string]interface{} {
	params := make(map[string]interface{})

	for _, p := range t.Spec.Parameters {
		// Try resolving via valueFrom function first
		if p.ValueFrom != nil && reg != nil {
			val, err := reg.Resolve(&functions.ValueFromSpec{
				Function: p.ValueFrom.Function,
				Args:     p.ValueFrom.Args,
			})
			if err != nil {
				log.Printf("⚠️  failed to resolve valueFrom for parameter %q (function %q): %v", p.Name, p.ValueFrom.Function, err)
			} else {
				params[p.Name] = val
				continue
			}
		}

		// Use default value if available, otherwise use a reasonable default
		if p.Default != nil {
			// For multiselect parameters, convert []interface{} defaults to []string
			if p.Multiselect {
				if items, ok := p.Default.([]interface{}); ok {
					strs := make([]string, 0, len(items))
					for _, item := range items {
						strs = append(strs, fmt.Sprintf("%v", item))
					}
					params[p.Name] = strs
				} else {
					params[p.Name] = p.Default
				}
			} else {
				params[p.Name] = p.Default
			}
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

// RenderTemplate renders a claim template using KCL with optional custom parameters
func RenderTemplate(t *claimtemplate.ClaimTemplate, customParams ...map[string]interface{}) (string, error) {
	return RenderTemplateWithFunctions(t, nil, customParams...)
}

// RenderTemplateWithFunctions renders a claim template, resolving valueFrom parameters via the registry.
func RenderTemplateWithFunctions(t *claimtemplate.ClaimTemplate, reg *functions.Registry, customParams ...map[string]interface{}) (string, error) {
	// Build parameter values from template defaults + function calls
	params := BuildParameterValuesWithFunctions(t, reg)

	// Merge custom parameters if provided (user overrides take precedence)
	if len(customParams) > 0 && customParams[0] != nil {
		for key, value := range customParams[0] {
			params[key] = value
		}
	}

	// Render using KCL from OCI source
	result, err := render.RenderKCLFromOCI(t.Spec.Source, t.Spec.Tag, params)
	if err != nil {
		return "", fmt.Errorf("rendering failed for template %s: %w", t.Metadata.Name, err)
	}

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
