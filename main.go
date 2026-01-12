package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/stuttgart-things/claim-machinery-api/internal/api"
	"github.com/stuttgart-things/claim-machinery-api/internal/app"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
)

func main() {
	fmt.Println("🚀 Claim Machinery API starting")

	// Flags (override env)
	templatesDirFlag := flag.String("templates-dir", "", "Path to templates directory")
	profilePathFlag := flag.String("template-profile-path", "", "Path to template profile YAML")
	flag.Parse()

	// Load templates directory (flag > env > default)
	templatesDir := *templatesDirFlag
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
	profilePath := *profilePathFlag
	if profilePath == "" {
		profilePath = os.Getenv("TEMPLATE_PROFILE_PATH")
	}

	var server *api.Server
	var err error
	if profilePath == "" {
		// Load from directory only
		dirTemplates, err1 := app.LoadAllTemplates(templatesDir)
		if err1 != nil {
			log.Fatalf("failed to load templates from dir: %v", err1)
		}
		fmt.Printf("📂 Using templates directory: %s\n", templatesDir)
		fmt.Printf("🧾 Loaded %d templates from directory\n", len(dirTemplates))
		for _, t := range dirTemplates {
			fmt.Printf("   • %s\n", t.Metadata.Name)
		}
		server, err = api.NewServerWithTemplates(dirTemplates)
	} else {
		// Combine directory templates with profile templates
		dirTemplates, err1 := app.LoadAllTemplates(templatesDir)
		if err1 != nil {
			log.Fatalf("failed to load templates from dir: %v", err1)
		}
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

		// Log loaded sources for visibility
		fmt.Printf("📂 Using templates directory: %s\n", templatesDir)
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
}
