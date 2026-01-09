package claimtemplate

import (
	"os"

	"gopkg.in/yaml.v3"
)

func LoadClaimTemplate(path string) (*ClaimTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tmpl ClaimTemplate
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}

	return &tmpl, nil
}
