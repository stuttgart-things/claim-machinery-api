package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/stuttgart-things/claim-machinery-api/internal/api"
)

func main() {
	fmt.Println("🚀 Claim Machinery API starting")

	// Load templates directory
	templatesDir := filepath.Join(
		"internal",
		"claimtemplate",
		"testdata",
	)

	// Create and start HTTP server
	server, err := api.NewServer(templatesDir)
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
