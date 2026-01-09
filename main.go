package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
)

func main() {
	fmt.Println("🚀 Claim Machinery API starting")

	// Load all templates from testdata directory
	templatesDir := filepath.Join(
		"internal",
		"claimtemplate",
		"testdata",
	)

	templates, err := loadAllTemplates(templatesDir)
	if err != nil {
		log.Fatalf("failed to load claim templates: %v", err)
	}

	if len(templates) == 0 {
		log.Fatal("no templates found in testdata directory")
	}

	fmt.Printf("\n✓ Loaded %d claim template(s)\n\n", len(templates))
	for _, tmpl := range templates {
		printTemplateSummary(tmpl)
		fmt.Println()
	}
}

// loadAllTemplates scans the templates directory and loads all YAML files
func loadAllTemplates(dir string) ([]*claimtemplate.ClaimTemplate, error) {
	var templates []*claimtemplate.ClaimTemplate

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only load YAML files
		if !isYAMLFile(entry.Name()) {
			continue
		}

		templatePath := filepath.Join(dir, entry.Name())
		tmpl, err := claimtemplate.LoadClaimTemplate(templatePath)
		if err != nil {
			log.Printf("⚠️  failed to load template %s: %v", entry.Name(), err)
			continue
		}

		templates = append(templates, tmpl)
	}

	return templates, nil
}

// isYAMLFile checks if a file is a YAML file
func isYAMLFile(filename string) bool {
	ext := filepath.Ext(filename)
	return ext == ".yaml" || ext == ".yml"
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
