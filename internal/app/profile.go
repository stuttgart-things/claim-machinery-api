package app

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/functions"
	"gopkg.in/yaml.v3"
)

// profileYAML represents the structure of the profile file.
// Support both "templates" and the common misspelling "tenplates".
type profileYAML struct {
	Name      string                 `yaml:"name,omitempty"`
	Templates []string               `yaml:"templates"`
	Tenplates []string               `yaml:"tenplates"`
	Functions []functions.FunctionDef `yaml:"functions,omitempty"`
}

// ProfileResult holds everything loaded from a profile file.
type ProfileResult struct {
	Name      string // profile name (from YAML or derived from filename)
	Templates []*claimtemplate.ClaimTemplate
	Sources   []string
	Functions *functions.Registry
}

// LoadProfileWithFunctions loads templates and functions from a profile file.
func LoadProfileWithFunctions(profilePath string) (*ProfileResult, error) {
	p, err := parseProfile(profilePath)
	if err != nil {
		return nil, err
	}

	profileName := resolveProfileName(p, profilePath)
	templates, sources := loadTemplateEntries(p)

	// Tag every template with the profile name
	for _, t := range templates {
		t.Metadata.Profile = profileName
	}

	return &ProfileResult{
		Name:      profileName,
		Templates: templates,
		Sources:   sources,
		Functions: functions.NewRegistry(p.Functions),
	}, nil
}

// LoadMultipleProfiles loads templates and functions from multiple colon-separated profile paths.
// Templates are merged by metadata.name (later profiles override earlier ones on conflict).
// Functions from all profiles are merged; duplicate function names cause an error.
func LoadMultipleProfiles(profilePaths string) ([]*ProfileResult, error) {
	paths := SplitProfilePaths(profilePaths)
	var results []*ProfileResult

	for _, p := range paths {
		result, err := LoadProfileWithFunctions(p)
		if err != nil {
			return nil, fmt.Errorf("load profile %s: %w", p, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// MergeProfileResults merges multiple ProfileResults into consolidated templates, sources, and functions.
// Duplicate function names across profiles cause an error.
func MergeProfileResults(results []*ProfileResult) ([]*claimtemplate.ClaimTemplate, *functions.Registry, map[string]string, error) {
	merged := make(map[string]*claimtemplate.ClaimTemplate)
	importSources := make(map[string]string)
	var allFuncDefs []functions.FunctionDef
	seenFuncs := make(map[string]string) // func name -> profile name

	for _, r := range results {
		for i, t := range r.Templates {
			merged[t.Metadata.Name] = t
			if i < len(r.Sources) {
				importSources[t.Metadata.Name] = r.Sources[i]
			}
		}

		// Collect functions, checking for duplicates
		for _, fd := range r.Functions.Defs() {
			if existingProfile, exists := seenFuncs[fd.Name]; exists {
				return nil, nil, nil, fmt.Errorf("duplicate function %q in profiles %q and %q", fd.Name, existingProfile, r.Name)
			}
			seenFuncs[fd.Name] = r.Name
			allFuncDefs = append(allFuncDefs, fd)
		}
	}

	templates := make([]*claimtemplate.ClaimTemplate, 0, len(merged))
	for _, t := range merged {
		templates = append(templates, t)
	}

	return templates, functions.NewRegistry(allFuncDefs), importSources, nil
}

// splitProfilePaths splits a colon-separated list of profile paths.
// Supports both local paths and URLs (colons in http:// and https:// are preserved).
func SplitProfilePaths(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var paths []string
	remaining := raw
	for remaining != "" {
		// Check for URL prefix
		if strings.HasPrefix(remaining, "http://") || strings.HasPrefix(remaining, "https://") {
			// Find the next colon that is NOT part of the URL scheme
			scheme := "http://"
			if strings.HasPrefix(remaining, "https://") {
				scheme = "https://"
			}
			rest := remaining[len(scheme):]
			idx := strings.Index(rest, ":")
			if idx == -1 {
				paths = append(paths, remaining)
				break
			}
			paths = append(paths, remaining[:len(scheme)+idx])
			remaining = rest[idx+1:]
		} else {
			idx := strings.Index(remaining, ":")
			if idx == -1 {
				paths = append(paths, remaining)
				break
			}
			paths = append(paths, remaining[:idx])
			remaining = remaining[idx+1:]
		}
	}

	// Trim whitespace from each path
	var cleaned []string
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return cleaned
}

// resolveProfileName returns the profile name from the YAML or derives it from the file path.
func resolveProfileName(p *profileYAML, profilePath string) string {
	if p.Name != "" {
		return p.Name
	}
	// Derive from filename: /path/to/flux-profiles.yaml -> "flux-profiles"
	base := filepath.Base(profilePath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// LoadTemplatesFromProfile loads claim templates from a YAML profile file.
// profilePath can be a local file path or an HTTP/HTTPS URL. Individual
// entries inside the profile can also be local paths or URLs.
func LoadTemplatesFromProfile(profilePath string) ([]*claimtemplate.ClaimTemplate, []string, error) {
	p, err := parseProfile(profilePath)
	if err != nil {
		return nil, nil, err
	}

	templates, sources := loadTemplateEntries(p)
	return templates, sources, nil
}

// parseProfile reads and parses a profile YAML from a local path or URL.
func parseProfile(profilePath string) (*profileYAML, error) {
	var reader io.ReadCloser

	if strings.HasPrefix(profilePath, "http://") || strings.HasPrefix(profilePath, "https://") {
		tmpPath, err := downloadToTemp(profilePath)
		if err != nil {
			return nil, fmt.Errorf("download profile from %s: %w", profilePath, err)
		}
		defer os.Remove(tmpPath)
		f, err := os.Open(tmpPath)
		if err != nil {
			return nil, fmt.Errorf("open downloaded profile: %w", err)
		}
		reader = f
	} else {
		f, err := os.Open(profilePath)
		if err != nil {
			return nil, fmt.Errorf("open profile: %w", err)
		}
		reader = f
	}
	defer reader.Close()

	var p profileYAML
	if err := yaml.NewDecoder(reader).Decode(&p); err != nil {
		return nil, fmt.Errorf("parse profile yaml: %w", err)
	}
	return &p, nil
}

// loadTemplateEntries resolves template entries (local or remote) from a parsed profile.
func loadTemplateEntries(p *profileYAML) ([]*claimtemplate.ClaimTemplate, []string) {
	entries := append([]string{}, p.Templates...)
	if len(p.Tenplates) > 0 {
		entries = append(entries, p.Tenplates...)
	}

	var (
		out     []*claimtemplate.ClaimTemplate
		sources []string
	)

	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}

		localPath := e
		if strings.HasPrefix(e, "http://") || strings.HasPrefix(e, "https://") {
			// Validate URL via HEAD (fallback to GET), then download
			if err := validateURL(e); err != nil {
				log.Printf("⚠️  skip unreachable URL %s: %v", e, err)
				continue
			}
			pth, err := downloadToTemp(e)
			if err != nil {
				log.Printf("⚠️  failed to download %s: %v (skipping)", e, err)
				continue
			}
			localPath = pth
		} else {
			// Ensure local path exists
			if _, err := os.Stat(localPath); err != nil {
				log.Printf("⚠️  missing local template %s: %v (skipping)", localPath, err)
				continue
			}
		}

		tmpl, err := claimtemplate.LoadClaimTemplate(localPath)
		if err != nil {
			log.Printf("⚠️  failed to load template %s: %v (skipping)", e, err)
			continue
		}
		out = append(out, tmpl)
		sources = append(sources, e)
	}

	return out, sources
}

func validateURL(u string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	// Try HEAD first
	req, _ := http.NewRequest(http.MethodHead, u, nil)
	resp, err := client.Do(req)
	if err == nil && resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return nil
		}
	}
	// Fallback to GET
	resp, err = client.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return nil
}

func downloadToTemp(u string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("download status: %s", resp.Status)
	}

	// Make a temp file with a stable extension if possible
	ext := filepath.Ext(u)
	f, err := os.CreateTemp("", "claim-template-*"+ext)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return f.Name(), nil
}
