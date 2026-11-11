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
	Profile     string   `yaml:"-" json:"profile,omitempty"` // injected at load time from profile name
}

type ClaimTemplateSpec struct {
	Type       string           `yaml:"type" json:"type"`
	Source     string           `yaml:"source" json:"source"`
	Tag        string           `yaml:"tag,omitempty" json:"tag,omitempty"`
	Parameters []Parameter      `yaml:"parameters" json:"parameters"`
	Secrets    []SecretTemplate `yaml:"secrets,omitempty" json:"secrets,omitempty"`

	// OrderRoles, when set and order auth is on, are the roles of which the
	// orderer needs one to order this template, in addition to the global
	// AUTH_ORDER_ROLES; admins (AUTH_ADMIN_ROLES) always may (#122).
	OrderRoles []string `yaml:"orderRoles,omitempty" json:"orderRoles,omitempty"`
}

// SecretTemplate defines a Kubernetes Secret that accompanies the rendered template.
// Secret values are collected and encrypted client-side (SOPS/age) — they never pass through the API.
type SecretTemplate struct {
	Name       string      `yaml:"name" json:"name"`             // supports Go template expressions e.g. "{{.secretName}}"
	Namespace  string      `yaml:"namespace" json:"namespace"`   // supports Go template expressions
	Parameters []Parameter `yaml:"parameters" json:"parameters"` // secret key definitions
}

// ValueFromSpec references a profile function to dynamically resolve a parameter value.
type ValueFromSpec struct {
	Function  string            `yaml:"function" json:"function"`
	Args      map[string]string `yaml:"args" json:"args"`
	Condition string            `yaml:"condition,omitempty" json:"condition,omitempty"` // parameter name that must be "true" for this function to execute
}

type Parameter struct {
	Name        string      `yaml:"name" json:"name"`
	Title       string      `yaml:"title" json:"title"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	Type        string      `yaml:"type" json:"type"` // string | boolean | array | number
	Default     interface{} `yaml:"default,omitempty" json:"default,omitempty"`
	Required    bool        `yaml:"required,omitempty" json:"required,omitempty"`
	Enum        []string    `yaml:"enum,omitempty" json:"enum,omitempty"`

	// Hidden parameters are not shown in forms but use their default value
	// Useful for platform-defined values that users shouldn't change
	Hidden bool `yaml:"hidden,omitempty" json:"hidden,omitempty"`

	// AllowRandom adds a "Random" option to enum fields that picks a random value
	// Only applies to parameters with enum values
	AllowRandom bool `yaml:"allowRandom,omitempty" json:"allowRandom,omitempty"`

	// Multiselect allows selecting multiple values from enum options
	// Only applies to parameters with enum values
	Multiselect bool `yaml:"multiselect,omitempty" json:"multiselect,omitempty"`

	// ValueFrom resolves this parameter's value by calling a function defined in the profile.
	// The resolved value is used as the default; users can still override it interactively.
	ValueFrom *ValueFromSpec `yaml:"valueFrom,omitempty" json:"valueFrom,omitempty"`

	// Identity binds the parameter to the authenticated orderer when order
	// auth is on (#122); without auth it is an ordinary parameter.
	//   user:  set from the login (Backstage user ref); a different value
	//          from the caller is refused (400)
	//   owner: the login's user ref when empty; otherwise one of the login's
	//          groups or its own ref, any value for an admin (else 403)
	Identity string `yaml:"identity,omitempty" json:"identity,omitempty"`

	// ShowWhen makes the parameter conditional (#135): it applies only when
	// every named parameter has one of the listed values, e.g.
	// {use_case: [dev]}. Clients hide it otherwise and the API ignores a
	// value sent for it. See conditions.go.
	ShowWhen map[string][]string `yaml:"showWhen,omitempty" json:"showWhen,omitempty"`

	// EnumWhen narrows the allowed values by another parameter's value
	// (#135), e.g. {use_case: {dev: [large, xlarge]}}; a value not listed
	// leaves the enum as it is. See conditions.go.
	EnumWhen map[string]map[string][]string `yaml:"enumWhen,omitempty" json:"enumWhen,omitempty"`

	// Validation
	Pattern   string `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	MinLength *int   `yaml:"minLength,omitempty" json:"minLength,omitempty"`
	MaxLength *int   `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
}

// Identity binding values of Parameter.Identity.
const (
	IdentityUser  = "user"
	IdentityOwner = "owner"
)
