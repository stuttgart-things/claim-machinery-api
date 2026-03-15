package app

import (
	"fmt"
	"log"
	"strings"

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
	return BuildParameterValuesWithFunctionsAndUserParams(t, reg, nil)
}

// BuildParameterValuesWithFunctionsAndUserParams creates a map of parameter values from a template.
// User-provided params are pre-seeded so that valueFrom arg references (e.g. {{.networkKey}}) resolve correctly.
func BuildParameterValuesWithFunctionsAndUserParams(t *claimtemplate.ClaimTemplate, reg *functions.Registry, userParams map[string]interface{}) map[string]interface{} {
	params := make(map[string]interface{})

	// Pre-seed with user-provided values so valueFrom references can resolve them
	for k, v := range userParams {
		params[k] = v
	}

	for _, p := range t.Spec.Parameters {
		// Try resolving via valueFrom function first
		if p.ValueFrom != nil && reg != nil {
			// Check condition: skip if the referenced parameter is not "true"
			if p.ValueFrom.Condition != "" {
				condVal := fmt.Sprintf("%v", params[p.ValueFrom.Condition])
				if condVal != "true" {
					log.Printf("ℹ️  skipping valueFrom for %q: condition %q is %q", p.Name, p.ValueFrom.Condition, condVal)
					goto fallback
				}
			}

			// Resolve {{.paramName}} references in args from already-resolved params
			resolvedArgs := make(map[string]string, len(p.ValueFrom.Args))
			for k, v := range p.ValueFrom.Args {
				resolved := v
				for paramName, paramVal := range params {
					resolved = strings.ReplaceAll(resolved, "{{."+paramName+"}}", fmt.Sprintf("%v", paramVal))
				}
				resolvedArgs[k] = resolved
			}

			val, err := reg.Resolve(&functions.ValueFromSpec{
				Function: p.ValueFrom.Function,
				Args:     resolvedArgs,
			})
			if err != nil {
				log.Printf("⚠️  failed to resolve valueFrom for parameter %q (function %q): %v", p.Name, p.ValueFrom.Function, err)
			} else {
				params[p.Name] = val
				continue
			}
		}
	fallback:

		// Skip if user already provided a value for this parameter
		if _, hasUser := userParams[p.Name]; hasUser {
			continue
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

// RenderTemplateFromParams renders a claim template using pre-built parameter values.
// Unlike RenderTemplate, this does not rebuild parameters from the template definition.
func RenderTemplateFromParams(t *claimtemplate.ClaimTemplate, params map[string]interface{}) (string, error) {
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
