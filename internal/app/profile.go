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
	"gopkg.in/yaml.v3"
)

// profileYAML represents the structure of the profile file.
// Support both "templates" and the common misspelling "tenplates".
type profileYAML struct {
	Templates []string `yaml:"templates"`
	Tenplates []string `yaml:"tenplates"`
}

// LoadTemplatesFromProfile loads claim templates from a YAML profile file.
// Entries can be local file paths or HTTP/HTTPS URLs. URLs are validated and
// downloaded to a temporary file before parsing.
func LoadTemplatesFromProfile(profilePath string) ([]*claimtemplate.ClaimTemplate, []string, error) {
	f, err := os.Open(profilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open profile: %w", err)
	}
	defer f.Close()

	var p profileYAML
	if err := yaml.NewDecoder(f).Decode(&p); err != nil {
		return nil, nil, fmt.Errorf("parse profile yaml: %w", err)
	}

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

	return out, sources, nil
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
