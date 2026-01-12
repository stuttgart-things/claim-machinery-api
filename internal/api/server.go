package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/stuttgart-things/claim-machinery-api/internal/app"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/version"
)

// Server represents the HTTP API server
type Server struct {
	router    *mux.Router
	http      *http.Server
	templates map[string]*claimtemplate.ClaimTemplate
}

// NewServer creates and initializes a new HTTP server
func NewServer(templatesDir string) (*Server, error) {
	// Load templates on server startup
	templates, err := app.LoadAllTemplates(templatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Build template map for quick lookup
	templateMap := make(map[string]*claimtemplate.ClaimTemplate)
	for i, t := range templates {
		templateMap[t.Metadata.Name] = templates[i]
	}

	s := &Server{
		router:    mux.NewRouter(),
		templates: templateMap,
	}

	// Register routes
	s.registerRoutes()

	// Apply middleware
	s.applyMiddleware()

	// Setup HTTP server
	s.http = &http.Server{
		Addr:         ":8080",
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// NewServerWithTemplates creates a server from an explicit list of templates.
// This is useful when combining multiple sources (e.g., directory + profile file).
func NewServerWithTemplates(templates []*claimtemplate.ClaimTemplate) (*Server, error) {
	// Build template map for quick lookup
	templateMap := make(map[string]*claimtemplate.ClaimTemplate)
	for i, t := range templates {
		templateMap[t.Metadata.Name] = templates[i]
	}

	s := &Server{
		router:    mux.NewRouter(),
		templates: templateMap,
	}

	// Register routes
	s.registerRoutes()

	// Apply middleware
	s.applyMiddleware()

	// Setup HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	s.http = &http.Server{
		Addr:         ":" + port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// registerRoutes sets up all API routes
func (s *Server) registerRoutes() {
	// Health check endpoint
	s.router.HandleFunc("/health", s.healthCheck).Methods(http.MethodGet)

	// Service metadata and docs
	s.router.HandleFunc("/", s.rootInfo).Methods(http.MethodGet)
	s.router.HandleFunc("/version", s.versionInfo).Methods(http.MethodGet)
	s.router.HandleFunc("/openapi", s.serveOpenAPI).Methods(http.MethodGet)
	s.router.HandleFunc("/openapi.yaml", s.serveOpenAPI).Methods(http.MethodGet)
	s.router.HandleFunc("/docs", s.serveDocs).Methods(http.MethodGet)

	// API endpoints
	s.router.HandleFunc("/api/v1/claim-templates", s.listTemplates).Methods(http.MethodGet)
	s.router.HandleFunc("/api/v1/claim-templates/{name}", s.getTemplate).Methods(http.MethodGet)
	s.router.HandleFunc("/api/v1/claim-templates/{name}/order", s.orderClaim).Methods(http.MethodPost)

	// Optional test-only routes (enable with ENABLE_TEST_ROUTES=1)
	if os.Getenv("ENABLE_TEST_ROUTES") == "1" || os.Getenv("ENABLE_TEST_ROUTES") == "true" {
		s.router.HandleFunc("/__test/panic", s.panicTest).Methods(http.MethodGet)
	}
}

// applyMiddleware applies middleware to all routes
func (s *Server) applyMiddleware() {
	// Middleware werden in Registrierungsreihenfolge ausgeführt.
	// Reihenfolge: errorHandler -> cors -> requestID -> logging
	s.router.Use(errorHandlerMiddleware)
	s.router.Use(corsMiddleware)
	s.router.Use(requestIDMiddleware)
	s.router.Use(loggingMiddleware)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("🚀 HTTP API server starting on %s", s.http.Addr)
	return s.http.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("⏹️  Shutting down HTTP server...")
	return s.http.Shutdown(ctx)
}

// healthCheck returns server health status
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// rootInfo returns a minimal service index with useful links
func (s *Server) rootInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
			"service": "claim-machinery-api",
			"version": "%s",
			"endpoints": [
				"/health",
				"/version",
				"/api/v1/claim-templates",
				"/api/v1/claim-templates/{name}",
				"/api/v1/claim-templates/{name}/order",
				"/openapi.yaml",
				"/docs"
			]
		}`, version.Version)
}

// versionInfo returns build-time version metadata
func (s *Server) versionInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
		"version": "%s",
		"commit": "%s",
		"buildDate": "%s"
	}`, version.Version, version.Commit, version.BuildDate)
}

// serveOpenAPI serves the OpenAPI specification from docs/openapi.yaml if present
func (s *Server) serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(filepath.Join("docs", "openapi.yaml"))
	if _, err := os.Stat(path); err == nil {
		w.Header().Set("Content-Type", "application/yaml")
		http.ServeFile(w, r, path)
		return
	}
	// Fallback: minimal inline doc
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "openapi: 3.0.3\ninfo:\n  title: Claim Machinery API\n  version: 0.0.0\npaths:\n  /health:\n    get:\n      summary: Health check\n      responses:\n        '200':\n          description: OK\n  /api/v1/claim-templates:\n    get:\n      summary: List claim templates\n      responses:\n        '200':\n          description: OK\n  /api/v1/claim-templates/{name}:\n    get:\n      summary: Get template\n      parameters:\n        - in: path\n          name: name\n          required: true\n          schema:\n            type: string\n      responses:\n        '200':\n          description: OK\n  /api/v1/claim-templates/{name}/order:\n    post:\n      summary: Render template\n      parameters:\n        - in: path\n          name: name\n          required: true\n          schema:\n            type: string\n      requestBody:\n        content:\n          application/json:\n            schema:\n              type: object\n      responses:\n        '200':\n          description: OK\n")
}

// serveDocs serves a simple Redoc viewer for the OpenAPI spec
func (s *Server) serveDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!doctype html>
<html>
	<head>
		<meta charset="utf-8"/>
		<title>Claim Machinery API Docs</title>
		<script src="https://cdn.jsdelivr.net/npm/redoc@next/bundles/redoc.standalone.js"></script>
	</head>
	<body>
		<redoc spec-url="/openapi.yaml"></redoc>
	</body>
</html>`)
}

// panicTest intentionally panics to test error handler and logging
func (s *Server) panicTest(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Query().Get("msg")
	if msg == "" {
		msg = "test panic"
	}
	panic(msg)
}
