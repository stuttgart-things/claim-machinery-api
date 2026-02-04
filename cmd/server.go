package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
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

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			if err.Error() != "http: Server closed" {
				log.Printf("❌ Server error: %v", err)
			}
		}
	}()

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

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Printf("error during shutdown: %v", err)
	}

	fmt.Println("✓ Server stopped gracefully")
	return nil
}
