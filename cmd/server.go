package cmd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/stuttgart-things/claim-machinery-api/internal/api"
	"github.com/stuttgart-things/claim-machinery-api/internal/app"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the API server",
	Long:  `Start the HTTP API server for template rendering.`,
	RunE:  runServer,
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

func runServer(cmd *cobra.Command, args []string) error {
	fmt.Println(logo)
	fmt.Printf("Version:    %s\n", Version)
	fmt.Printf("Commit:     %s\n", Commit)
	fmt.Printf("Build Date: %s\n\n", Date)

	// Resolve enableTemplatesDir from flag or env var
	if !enableTemplatesDir {
		if v := os.Getenv("ENABLE_TEMPLATES_DIR"); v == "true" {
			enableTemplatesDir = true
		}
	}

	// Load templates directory (flag > env > default)
	if templatesDir == "" {
		templatesDir = os.Getenv("TEMPLATES_DIR")
	}
	if templatesDir == "" {
		templatesDir = filepath.Join(
			"internal",
			"claimtemplate",
			"testdata",
		)
	}

	// Optionally load additional templates from YAML profile
	if profilePath == "" {
		profilePath = os.Getenv("TEMPLATE_PROFILE_PATH")
	}
	if profilePath == "" {
		profilePath = "tests/profile.yaml"
	}

	var dirTemplates []*claimtemplate.ClaimTemplate
	if enableTemplatesDir {
		var err error
		dirTemplates, err = app.LoadAllTemplates(templatesDir)
		if err != nil {
			log.Fatalf("failed to load templates from dir: %v", err)
		}
		fmt.Printf("📂 Using templates directory: %s\n", templatesDir)
		fmt.Printf("🧾 Loaded %d templates from directory\n", len(dirTemplates))
		for _, t := range dirTemplates {
			fmt.Printf("   • %s\n", t.Metadata.Name)
		}
	} else {
		fmt.Println("📂 Templates directory loading disabled")
	}

	var server *api.Server
	var err error
	if profilePath == "" {
		server, err = api.NewServerWithTemplates(dirTemplates)
	} else {
		profileTemplates, sources, err2 := app.LoadTemplatesFromProfile(profilePath)
		if err2 != nil {
			log.Fatalf("failed to load templates from profile: %v", err2)
		}

		// Merge, de-duplicate by metadata.name (profile overrides directory on conflict)
		merged := make(map[string]*claimtemplate.ClaimTemplate)
		for _, t := range dirTemplates {
			merged[t.Metadata.Name] = t
		}
		for _, t := range profileTemplates {
			merged[t.Metadata.Name] = t
		}
		final := make([]*claimtemplate.ClaimTemplate, 0, len(merged))
		for _, t := range merged {
			final = append(final, t)
		}

		fmt.Printf("🧾 Loaded %d templates from profile %s\n", len(profileTemplates), profilePath)
		for _, s := range sources {
			fmt.Printf("   • source: %s\n", s)
		}
		fmt.Printf("🧾 Templates in use (%d):\n", len(final))
		for _, t := range final {
			fmt.Printf("   • %s\n", t.Metadata.Name)
		}

		server, err = api.NewServerWithTemplates(final)
	}
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Start server with automatic restart on crash
	stopCh := make(chan struct{})
	go func() {
		backoff := time.Second
		const maxBackoff = 30 * time.Second
		for {
			if err := server.Start(); err != nil {
				if err.Error() == "http: Server closed" {
					return
				}
				log.Printf("Server error: %v (restarting in %s)", err, backoff)
				select {
				case <-stopCh:
					return
				case <-time.After(backoff):
				}
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
			return
		}
	}()

	// Start config auto-reload watcher for profile changes
	if profilePath != "" {
		go watchProfileForReload(stopCh, server, profilePath, enableTemplatesDir, templatesDir)
	}

	fmt.Println("✓ API server listening on http://localhost:8080")
	fmt.Printf("📂 Using templates directory: %s\n", templatesDir)
	fmt.Println("\n📋 Available endpoints:")
	fmt.Println("  GET  /health                                    - Health check")
	fmt.Println("  GET  /api/v1/claim-templates                    - List templates")
	fmt.Println("  GET  /api/v1/claim-templates/{name}             - Get template details")
	fmt.Println("  POST /api/v1/claim-templates/{name}/order       - Render template")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	fmt.Printf("\n📮 Received signal: %v\n", sig)

	// Signal watcher and restart loop to stop
	close(stopCh)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Printf("error during shutdown: %v", err)
	}

	fmt.Println("✓ Server stopped gracefully")
	return nil
}

// watchProfileForReload periodically checks the profile for changes and
// reloads templates into the running server when a change is detected.
// Supports both local file paths and remote HTTP/HTTPS URLs.
func watchProfileForReload(stopCh <-chan struct{}, server *api.Server, profilePath string, enableTemplatesDir bool, templatesDir string) {
	const pollInterval = 10 * time.Second
	isRemote := strings.HasPrefix(profilePath, "http://") || strings.HasPrefix(profilePath, "https://")

	lastHash := profileHash(profilePath, isRemote)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			currentHash := profileHash(profilePath, isRemote)
			if currentHash == lastHash {
				continue
			}
			lastHash = currentHash
			log.Printf("Profile changed, reloading templates... (source: %s)", profilePath)

			templates, err := loadMergedTemplates(enableTemplatesDir, templatesDir, profilePath)
			if err != nil {
				log.Printf("Failed to reload templates: %v (keeping previous)", err)
				continue
			}

			server.UpdateTemplates(templates)
			log.Printf("Templates reloaded successfully (%d templates)", len(templates))
		}
	}
}

// loadMergedTemplates loads and merges templates from directory and profile,
// replicating the startup merge logic.
func loadMergedTemplates(enableTemplatesDir bool, templatesDir, profilePath string) ([]*claimtemplate.ClaimTemplate, error) {
	var dirTemplates []*claimtemplate.ClaimTemplate
	if enableTemplatesDir {
		var err error
		dirTemplates, err = app.LoadAllTemplates(templatesDir)
		if err != nil {
			return nil, fmt.Errorf("load templates from dir: %w", err)
		}
	}

	profileTemplates, _, err := app.LoadTemplatesFromProfile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("load templates from profile: %w", err)
	}

	merged := make(map[string]*claimtemplate.ClaimTemplate)
	for _, t := range dirTemplates {
		merged[t.Metadata.Name] = t
	}
	for _, t := range profileTemplates {
		merged[t.Metadata.Name] = t
	}

	result := make([]*claimtemplate.ClaimTemplate, 0, len(merged))
	for _, t := range merged {
		result = append(result, t)
	}
	return result, nil
}

// profileHash returns the SHA-256 hex digest of a profile source (local file or remote URL).
// Returns empty string on error.
func profileHash(path string, isRemote bool) string {
	if isRemote {
		return remoteHash(path)
	}
	return fileHash(path)
}

// fileHash returns the SHA-256 hex digest of a local file, or empty string on error.
func fileHash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// remoteHash fetches a URL and returns the SHA-256 hex digest of the response body,
// or empty string on error.
func remoteHash(url string) string {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Failed to fetch profile URL for hash check: %v", err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		log.Printf("Profile URL returned status %s during hash check", resp.Status)
		return ""
	}
	h := sha256.New()
	if _, err := io.Copy(h, resp.Body); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
