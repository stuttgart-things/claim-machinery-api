package claimtemplate

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadClaimTemplate(path string) (*ClaimTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("empty template file: %s", path)
	}

	var tmpl ClaimTemplate
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}

	if tmpl.Metadata.Name == "" {
		return nil, fmt.Errorf("template missing metadata.name in %s", path)
	}

	return &tmpl, nil
}
