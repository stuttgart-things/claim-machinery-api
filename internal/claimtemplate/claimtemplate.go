package claimtemplate

// ClaimTemplateList represents GET /claim-templates
type ClaimTemplateList struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Items      []ClaimTemplate `json:"items"`
}

// ClaimTemplate represents a single claim template
type ClaimTemplate struct {
	APIVersion string                `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                `yaml:"kind" json:"kind"`
	Metadata   ClaimTemplateMetadata `yaml:"metadata" json:"metadata"`
	Spec       ClaimTemplateSpec     `yaml:"spec" json:"spec"`
}

type ClaimTemplateMetadata struct {
	Name        string   `yaml:"name" json:"name"`
	Title       string   `yaml:"title,omitempty" json:"title,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

type ClaimTemplateSpec struct {
	Type       string      `yaml:"type" json:"type"`
	Source     string      `yaml:"source" json:"source"`
	Tag        string      `yaml:"tag,omitempty" json:"tag,omitempty"`
	Parameters []Parameter `yaml:"parameters" json:"parameters"`
}

type Parameter struct {
	Name        string      `yaml:"name" json:"name"`
	Title       string      `yaml:"title" json:"title"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	Type        string      `yaml:"type" json:"type"` // string | boolean | array | number
	Default     interface{} `yaml:"default,omitempty" json:"default,omitempty"`
	Required    bool        `yaml:"required,omitempty" json:"required,omitempty"`
	Enum        []string    `yaml:"enum,omitempty" json:"enum,omitempty"`

	// Validation
	Pattern   string `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	MinLength *int   `yaml:"minLength,omitempty" json:"minLength,omitempty"`
	MaxLength *int   `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
}
