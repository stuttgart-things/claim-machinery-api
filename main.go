package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/stuttgart-things/claim-machinery-api/internal/app"
)

func main() {
	fmt.Println("🚀 Claim Machinery API starting")

	// Load all templates from testdata directory
	templatesDir := filepath.Join(
		"internal",
		"claimtemplate",
		"testdata",
	)

	templates, err := app.LoadAllTemplates(templatesDir)
	if err != nil {
		log.Fatalf("failed to load claim templates: %v", err)
	}

	if len(templates) == 0 {
		log.Fatal("no templates found in testdata directory")
	}

	fmt.Printf("\n✓ Loaded %d claim template(s)\n\n", len(templates))

	// Process each template: load, display summary, and render
	for _, tmpl := range templates {
		app.PrintTemplateSummary(tmpl)

		// Render the template
		fmt.Println("\n🔄 Rendering template...")
		rendered, err := app.RenderTemplate(tmpl)
		if err != nil {
			log.Printf("⚠️  failed to render template %s: %v\n", tmpl.Metadata.Name, err)
			continue
		}

		// Display rendered output
		app.PrintRenderedOutput(tmpl.Metadata.Name, rendered)
		fmt.Println()
	}

	fmt.Println("✓ All templates processed")
}
