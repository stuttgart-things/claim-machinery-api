package app

import (
	"fmt"

	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
)

// PrintTemplateSummary displays a formatted summary of a claim template
func PrintTemplateSummary(t *claimtemplate.ClaimTemplate) {
	fmt.Println("---- Loaded ClaimTemplate ----")
	fmt.Printf("Name:        %s\n", t.Metadata.Name)
	fmt.Printf("Title:       %s\n", t.Metadata.Title)
	fmt.Printf("Source:      %s\n", t.Spec.Source)
	fmt.Printf("Tag:         %s\n", t.Spec.Tag)
	fmt.Printf("Parameters:  %d\n", len(t.Spec.Parameters))

	for _, p := range t.Spec.Parameters {
		fmt.Printf("  - %s (%s) required=%v default=%v\n",
			p.Name,
			p.Type,
			p.Required,
			p.Default,
		)
	}
}

// PrintRenderedOutput displays the rendered YAML output
func PrintRenderedOutput(templateName string, yaml string) {
	fmt.Printf("\n---- Rendered YAML for %s ----\n", templateName)
	fmt.Println(yaml)
}
