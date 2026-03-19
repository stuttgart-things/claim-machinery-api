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
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/stuttgart-things/claim-machinery-api/internal/api"
	"github.com/stuttgart-things/claim-machinery-api/internal/app"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/functions"
	"github.com/stuttgart-things/claim-machinery-api/internal/notify"
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
	fmt.Println(renderVersionBadge(Version, Commit, Date))

	// --- Environment variables ---------------------------------------------------
	printSection("🔧 Environment")
	envVars := []struct {
		name, desc, fallback string
	}{
		{"PORT", "HTTP server port", "8080"},
		{"TEMPLATES_DIR", "Local template directory", "-"},
		{"TEMPLATE_PROFILE_PATH", "Profile YAML path", "-"},
		{"ENABLE_TEMPLATES_DIR", "Enable template dir loading", "false"},
		{"RELOAD_INTERVAL", "Profile auto-reload interval", "2m"},
		{"DEBUG", "Enable debug logging", "off"},
		{"LOG_FORMAT", "Log format (json/text)", "text"},
		{"ENABLE_HOMERUN", "Enable homerun notifications", "off"},
		{"HOMERUN_URL", "Omni-pitcher base URL", "-"},
		{"HOMERUN_AUTH_TOKEN", "Bearer token for pitcher", "-"},
		{"ENABLE_TEST_ROUTES", "Enable test endpoints", "off"},
	}
	printEnvTable(envVars)

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

	// Optionally load additional templates from YAML profile(s).
	// TEMPLATE_PROFILE_PATH supports colon-separated multiple paths.
	if profilePath == "" {
		profilePath = os.Getenv("TEMPLATE_PROFILE_PATH")
	}
	if profilePath == "" {
		profilePath = "tests/profile.yaml"
	}

	// --- Endpoints ---------------------------------------------------------------
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	printSection("🌐 Endpoints")
	printEndpoint("GET ", "/health", "Health check")
	printEndpoint("GET ", "/api/v1/claim-templates", "List templates")
	printEndpoint("GET ", "/api/v1/claim-templates/{name}", "Get template details")
	printEndpoint("POST", "/api/v1/claim-templates/{name}/order", "Render template")

	// --- Configuration -----------------------------------------------------------
	printSection("⚙️  Configuration")
	if enableTemplatesDir {
		printKV("templates-dir", templatesDir)
	}
	profilePaths := app.SplitProfilePaths(profilePath)
	for i, p := range profilePaths {
		printKV(fmt.Sprintf("profile[%d]", i), p)
	}

	// --- Load templates ----------------------------------------------------------
	var dirTemplates []*claimtemplate.ClaimTemplate
	if enableTemplatesDir {
		var err error
		dirTemplates, err = app.LoadAllTemplates(templatesDir)
		if err != nil {
			log.Fatalf("failed to load templates from dir: %v", err)
		}
	}

	var server *api.Server
	var err error
	importSources := make(map[string]string) // template name -> import path

	if len(profilePaths) == 0 {
		server, err = api.NewServerWithTemplates(dirTemplates)
		for _, t := range dirTemplates {
			importSources[t.Metadata.Name] = templatesDir
		}
	} else {
		profileResults, err2 := app.LoadMultipleProfiles(profilePath)
		if err2 != nil {
			log.Fatalf("failed to load templates from profiles: %v", err2)
		}

		// Merge dir templates + all profile results
		merged := make(map[string]*claimtemplate.ClaimTemplate)
		for _, t := range dirTemplates {
			merged[t.Metadata.Name] = t
			importSources[t.Metadata.Name] = templatesDir
		}

		profileTemplates, mergedReg, profileSources, err2 := app.MergeProfileResults(profileResults)
		if err2 != nil {
			log.Fatalf("failed to merge profile results: %v", err2)
		}
		for _, t := range profileTemplates {
			merged[t.Metadata.Name] = t
			if src, ok := profileSources[t.Metadata.Name]; ok {
				importSources[t.Metadata.Name] = src
			}
		}

		final := make([]*claimtemplate.ClaimTemplate, 0, len(merged))
		for _, t := range merged {
			final = append(final, t)
		}

		server, err = api.NewServerWithTemplates(final)
		if err == nil {
			server.SetFunctions(mergedReg)
		}
	}
	if err == nil {
		server.SetImportSources(importSources)
	}
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// --- Templates summary -------------------------------------------------------
	tplSources := server.TemplateSources()
	printSection(fmt.Sprintf("📦 Templates (%d loaded)", len(tplSources)))
	printTemplateTable(tplSources)

	// --- Notifications -----------------------------------------------------------
	homerunCfg := notify.LoadHomerunConfig()
	server.SetHomerunConfig(homerunCfg)
	printSection("🔔 Notifications")
	if homerunCfg.Enabled {
		printKV("homerun", fmt.Sprintf("✅ enabled (%s)", homerunCfg.URL))
	} else {
		printKV("homerun", "⏸️  disabled")
	}

	// --- Auto-reload ------------------------------------------------------------
	pollInterval := resolveReloadInterval()
	printSection("🔄 Auto-Reload")
	if len(profilePaths) > 0 && pollInterval > 0 {
		printKV("profile watch", fmt.Sprintf("✅ enabled (every %s, %d profiles)", pollInterval, len(profilePaths)))
	} else if len(profilePaths) > 0 {
		printKV("profile watch", "⏸️  disabled (interval set to 0)")
	} else {
		printKV("profile watch", "⏸️  disabled (no profile configured)")
	}

	// --- Start server ------------------------------------------------------------
	fmt.Println()
	fmt.Printf("  🚀 Server listening on :%s\n\n", port)

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
	if len(profilePaths) > 0 && pollInterval > 0 {
		go watchProfilesForReload(stopCh, server, profilePath, enableTemplatesDir, templatesDir, pollInterval)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	fmt.Printf("\n[server] received signal: %v, shutting down...\n", sig)

	// Signal watcher and restart loop to stop
	close(stopCh)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Printf("[server] shutdown error: %v", err)
	}

	fmt.Println("[server] stopped")
	return nil
}

// resolveReloadInterval returns the reload interval from flag, env, or default (2m).
// Returns 0 to disable auto-reload.
func resolveReloadInterval() time.Duration {
	raw := reloadInterval
	if raw == "" {
		raw = os.Getenv("RELOAD_INTERVAL")
	}
	if raw == "" {
		return 2 * time.Minute
	}
	if raw == "0" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("[auto-reload] invalid interval %q, using default 2m", raw)
		return 2 * time.Minute
	}
	return d
}

// watchProfilesForReload periodically checks all profiles for changes and
// reloads templates into the running server when a change is detected.
// Supports both local file paths and remote HTTP/HTTPS URLs.
func watchProfilesForReload(stopCh <-chan struct{}, server *api.Server, profilePath string, enableTemplatesDir bool, templatesDir string, pollInterval time.Duration) {
	paths := app.SplitProfilePaths(profilePath)

	// Build initial hash map for all profiles
	lastHashes := make(map[string]string, len(paths))
	for _, p := range paths {
		isRemote := strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://")
		lastHashes[p] = profileHash(p, isRemote)
	}
	previousNames := server.TemplateNames()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			changed := false
			for _, p := range paths {
				isRemote := strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://")
				currentHash := profileHash(p, isRemote)
				if currentHash != lastHashes[p] {
					lastHashes[p] = currentHash
					changed = true
				}
			}
			if !changed {
				continue
			}
			log.Printf("[auto-reload] profile change detected, reloading...")

			templates, reg, sources, err := loadMergedTemplates(enableTemplatesDir, templatesDir, profilePath)
			if err != nil {
				log.Printf("[auto-reload] reload failed: %v (keeping previous)", err)
				continue
			}

			server.UpdateTemplatesAndFunctions(templates, reg)
			server.SetImportSources(sources)

			// Build new name set for diff
			newNames := make([]string, 0, len(templates))
			for _, t := range templates {
				newNames = append(newNames, t.Metadata.Name)
			}
			sort.Strings(newNames)

			// Compute added / removed
			oldSet := make(map[string]bool, len(previousNames))
			for _, n := range previousNames {
				oldSet[n] = true
			}
			newSet := make(map[string]bool, len(newNames))
			for _, n := range newNames {
				newSet[n] = true
			}

			var added, removed []string
			for _, n := range newNames {
				if !oldSet[n] {
					added = append(added, n)
				}
			}
			for _, n := range previousNames {
				if !newSet[n] {
					removed = append(removed, n)
				}
			}

			// Styled reload summary
			printSection(fmt.Sprintf("🔄 Auto-Reload (%d templates)", len(templates)))
			for _, n := range added {
				fmt.Printf("  │ %s %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Render("+"), n)
			}
			for _, n := range removed {
				fmt.Printf("  │ %s %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render("−"), n)
			}
			if len(added) == 0 && len(removed) == 0 {
				fmt.Println(dimStyle.Render("  │ templates unchanged (profile content updated)"))
			}
			printTemplateTable(server.TemplateSources())

			previousNames = newNames
		}
	}
}

// loadMergedTemplates loads and merges templates from directory and multiple profiles,
// replicating the startup merge logic. Also returns the function registry and import sources.
func loadMergedTemplates(enableTemplatesDir bool, templatesDir, profilePath string) ([]*claimtemplate.ClaimTemplate, *functions.Registry, map[string]string, error) {
	var dirTemplates []*claimtemplate.ClaimTemplate
	if enableTemplatesDir {
		var err error
		dirTemplates, err = app.LoadAllTemplates(templatesDir)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("load templates from dir: %w", err)
		}
	}

	profileResults, err := app.LoadMultipleProfiles(profilePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load templates from profiles: %w", err)
	}

	profileTemplates, mergedReg, profileSources, err := app.MergeProfileResults(profileResults)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("merge profiles: %w", err)
	}

	merged := make(map[string]*claimtemplate.ClaimTemplate)
	importSources := make(map[string]string)
	for _, t := range dirTemplates {
		merged[t.Metadata.Name] = t
		importSources[t.Metadata.Name] = templatesDir
	}
	for _, t := range profileTemplates {
		merged[t.Metadata.Name] = t
		if src, ok := profileSources[t.Metadata.Name]; ok {
			importSources[t.Metadata.Name] = src
		}
	}

	result := make([]*claimtemplate.ClaimTemplate, 0, len(merged))
	for _, t := range merged {
		result = append(result, t)
	}
	return result, mergedReg, importSources, nil
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
