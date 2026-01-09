package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
)

func main() {
	fmt.Println("🚀 Claim Machinery API starting")

	// Path to template (later: directory scan, FS, DB, etc.)
	templatePath := filepath.Join(
		"internal",
		"claimtemplate",
		"testdata",
		"volumeclaim.yaml",
	)

	tmpl, err := claimtemplate.LoadClaimTemplate(templatePath)
	if err != nil {
		log.Fatalf("failed to load claim template: %v", err)
	}

	printTemplateSummary(tmpl)
}

func printTemplateSummary(t *claimtemplate.ClaimTemplate) {
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
