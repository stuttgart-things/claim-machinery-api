package functions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// FunctionDef defines a reusable function that can be called to resolve parameter values.
type FunctionDef struct {
	Name     string            `yaml:"name" json:"name"`
	Type     string            `yaml:"type" json:"type"`       // "http"
	Method   string            `yaml:"method" json:"method"`   // GET, POST
	URL      string            `yaml:"url" json:"url"`         // supports ${var} placeholders
	Headers  map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body     map[string]any    `yaml:"body,omitempty" json:"body,omitempty"`
	Response ResponseExtract   `yaml:"response" json:"response"`
}

// ResponseExtract defines how to extract a value from the function response.
type ResponseExtract struct {
	// Extract is a dot-separated path into the JSON response (e.g. "ip", "data.address").
	Extract string `yaml:"extract" json:"extract"`
}

// ValueFromSpec references a function and provides arguments for parameter resolution.
type ValueFromSpec struct {
	Function string            `yaml:"function" json:"function"`
	Args     map[string]string `yaml:"args" json:"args"`
}

// Registry holds loaded function definitions for lookup.
type Registry struct {
	funcs map[string]*FunctionDef
}

// NewRegistry creates a Registry from a slice of FunctionDef.
func NewRegistry(defs []FunctionDef) *Registry {
	m := make(map[string]*FunctionDef, len(defs))
	for i := range defs {
		m[defs[i].Name] = &defs[i]
	}
	return &Registry{funcs: m}
}

// Resolve calls the function referenced by spec and returns the extracted value.
func (r *Registry) Resolve(spec *ValueFromSpec) (string, error) {
	if r == nil || spec == nil {
		return "", fmt.Errorf("nil registry or spec")
	}

	fn, ok := r.funcs[spec.Function]
	if !ok {
		return "", fmt.Errorf("function %q not found in registry", spec.Function)
	}

	switch fn.Type {
	case "http":
		return executeHTTP(fn, spec.Args)
	default:
		return "", fmt.Errorf("unsupported function type %q", fn.Type)
	}
}

// Has returns true if the registry contains the named function.
func (r *Registry) Has(name string) bool {
	if r == nil {
		return false
	}
	_, ok := r.funcs[name]
	return ok
}

// Defs returns all function definitions in the registry.
func (r *Registry) Defs() []FunctionDef {
	if r == nil {
		return nil
	}
	defs := make([]FunctionDef, 0, len(r.funcs))
	for _, fn := range r.funcs {
		defs = append(defs, *fn)
	}
	return defs
}

func executeHTTP(fn *FunctionDef, args map[string]string) (string, error) {
	url := substituteVars(fn.URL, args)
	method := strings.ToUpper(fn.Method)
	if method == "" {
		method = http.MethodGet
	}

	log.Printf("function %q: %s %s", fn.Name, method, url)

	var bodyReader io.Reader
	if fn.Body != nil && method == http.MethodPost {
		resolved := substituteBodyVars(fn.Body, args)
		data, err := json.Marshal(resolved)
		if err != nil {
			return "", fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// Set default Content-Type for POST
	if method == http.MethodPost && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Apply configured headers
	for k, v := range fn.Headers {
		req.Header.Set(k, substituteVars(v, args))
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Prevent redirects from changing POST to GET (301/302 behavior)
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute function %q: %w", fn.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("function %q returned status %d: %s", fn.Name, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	return extractValue(body, fn.Response.Extract)
}

// substituteVars replaces ${key} placeholders in s with values from args,
// falling back to environment variables.
func substituteVars(s string, args map[string]string) string {
	result := s
	for k, v := range args {
		result = strings.ReplaceAll(result, "${"+k+"}", v)
	}
	// Resolve remaining ${ENV_VAR} from environment
	for {
		start := strings.Index(result, "${")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		varName := result[start+2 : start+end]
		envVal := os.Getenv(varName)
		result = result[:start] + envVal + result[start+end+1:]
	}
	return result
}

// substituteBodyVars recursively substitutes ${key} in body map values.
func substituteBodyVars(body map[string]any, args map[string]string) map[string]any {
	out := make(map[string]any, len(body))
	for k, v := range body {
		switch val := v.(type) {
		case string:
			resolved := substituteVars(val, args)
			// Convert pure boolean/numeric strings to native types
			if resolved == "true" {
				out[k] = true
			} else if resolved == "false" {
				out[k] = false
			} else {
				out[k] = resolved
			}
		case map[string]any:
			out[k] = substituteBodyVars(val, args)
		default:
			// Check if it's a ${var} placeholder stored as non-string (e.g. boolean template)
			s := fmt.Sprintf("%v", val)
			if strings.Contains(s, "${") {
				resolved := substituteVars(s, args)
				// Try to preserve original type
				if resolved == "true" {
					out[k] = true
				} else if resolved == "false" {
					out[k] = false
				} else {
					out[k] = resolved
				}
			} else {
				out[k] = v
			}
		}
	}
	return out
}

// extractValue extracts a value from JSON using a dot-separated path.
func extractValue(data []byte, path string) (string, error) {
	if path == "" {
		s := strings.TrimSpace(string(data))
		// Strip surrounding quotes from raw JSON string values
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			s = s[1 : len(s)-1]
		}
		return s, nil
	}

	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return "", fmt.Errorf("parse response JSON: %w", err)
	}

	parts := strings.Split(path, ".")
	current := obj

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return "", fmt.Errorf("key %q not found in response at path %q", part, path)
			}
			current = val
		default:
			return "", fmt.Errorf("cannot traverse into %T at key %q", current, part)
		}
	}

	return fmt.Sprintf("%v", current), nil
}
