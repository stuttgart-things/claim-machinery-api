package main

import (
"fmt"
"log"
"os"
"os/signal"
"path/filepath"
"syscall"

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
if err := server.Start(":8080"); err != nil {
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

// Graceful shutdown
if err := server.Stop(); err != nil {
log.Printf("error during shutdown: %v", err)
}

fmt.Println("✓ Server stopped gracefully")
}
