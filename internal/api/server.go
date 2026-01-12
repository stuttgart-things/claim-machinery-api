package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/stuttgart-things/claim-machinery-api/internal/app"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
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
	s.http = &http.Server{
		Addr:         ":8080",
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

	// API endpoints
	s.router.HandleFunc("/api/v1/claim-templates", s.listTemplates).Methods(http.MethodGet)
	s.router.HandleFunc("/api/v1/claim-templates/{name}", s.getTemplate).Methods(http.MethodGet)
	s.router.HandleFunc("/api/v1/claim-templates/{name}/order", s.orderClaim).Methods(http.MethodPost)
}

// applyMiddleware applies middleware to all routes
func (s *Server) applyMiddleware() {
	// Apply middleware in reverse order (last applied = first executed)
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)
	s.router.Use(errorHandlerMiddleware)
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
