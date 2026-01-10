package api

import (
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
	server    *http.Server
	templates map[string]*claimtemplate.ClaimTemplate
}

// NewServer creates a new API server with templates loaded from directory
func NewServer(templateDir string) (*Server, error) {
	s := &Server{
		router:    mux.NewRouter(),
		templates: make(map[string]*claimtemplate.ClaimTemplate),
	}

	// Load templates from directory
	templates, err := app.LoadAllTemplates(templateDir)
	if err != nil {
		return nil, err
	}

	// Index templates by name
	for _, tmpl := range templates {
		s.templates[tmpl.Metadata.Name] = tmpl
	}

	log.Printf("✅ Loaded %d templates from %s", len(s.templates), templateDir)
	for name := range s.templates {
		log.Printf("  - %s", name)
	}

	// Register routes and middleware
	s.registerRoutes()
	s.applyMiddleware()

	return s, nil
}

// registerRoutes registers all API routes
func (s *Server) registerRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.healthCheck).Methods(http.MethodGet)

	// API v1
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/claim-templates", s.listTemplates).Methods(http.MethodGet)
	api.HandleFunc("/claim-templates/{name}", s.getTemplate).Methods(http.MethodGet)
	api.HandleFunc("/claim-templates/{name}/order", s.orderClaim).Methods(http.MethodPost)
}

// applyMiddleware applies middleware to all routes
func (s *Server) applyMiddleware() {
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)
	s.router.Use(errorHandlerMiddleware)
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	log.Printf("🚀 Starting API server on http://%s", addr)
	log.Printf("  📌 GET  /health")
	log.Printf("  📌 GET  /api/v1/claim-templates")
	log.Printf("  📌 GET  /api/v1/claim-templates/{name}")
	log.Printf("  📌 POST /api/v1/claim-templates/{name}/order")

	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}
	return s.server.Close()
}

// healthCheck returns server health status
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339)); err != nil {
		log.Printf("error writing health response: %v", err)
	}
}
