package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
)

// LoadAllTemplates scans the templates directory and loads all YAML files
func LoadAllTemplates(dir string) ([]*claimtemplate.ClaimTemplate, error) {
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
