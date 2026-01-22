package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/stuttgart-things/claim-machinery-api/internal/app"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
)

// OrderRequest represents a claim order request
type OrderRequest struct {
	Parameters map[string]interface{} `json:"parameters"`
}

// OrderResponse represents a rendered claim response
type OrderResponse struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Rendered   string                 `json:"rendered"`
}

// ClaimTemplateListResponse wraps templates for list endpoint
type ClaimTemplateListResponse struct {
	APIVersion string                        `json:"apiVersion"`
	Kind       string                        `json:"kind"`
	Items      []claimtemplate.ClaimTemplate `json:"items"`
}

// listTemplates returns all available claim templates
func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Build response
	response := ClaimTemplateListResponse{
		APIVersion: "api.claim-machinery.io/v1alpha1",
		Kind:       "ClaimTemplateList",
		Items:      make([]claimtemplate.ClaimTemplate, 0),
	}

	// Add all templates to response
	for _, tmpl := range s.templates {
		response.Items = append(response.Items, *tmpl)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getTemplate returns a specific claim template by name
func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract template name from URL
	vars := mux.Vars(r)
	name := vars["name"]

	// Look up template
	tmpl, exists := s.templates[name]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "template not found",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tmpl)
}

// orderClaim renders a claim template with provided parameters
func (s *Server) orderClaim(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract template name from URL
	vars := mux.Vars(r)
	name := vars["name"]

	// Look up template
	tmpl, exists := s.templates[name]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "template not found",
		})
		return
	}

	// Parse request body
	var req OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid request body",
		})
		return
	}

	// Build parameter values (merge request params with defaults)
	params := app.BuildParameterValues(tmpl)
	for key, value := range req.Parameters {
		params[key] = value
	}

	// Render template with custom parameters
	rendered, err := app.RenderTemplate(tmpl, params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Return success response
	response := OrderResponse{
		APIVersion: "api.claim-machinery.io/v1alpha1",
		Kind:       "OrderResponse",
		Metadata: map[string]interface{}{
			"name":      name + "-order-" + time.Now().Format("20060102150405"),
			"timestamp": time.Now().Format(time.RFC3339),
		},
		Rendered: rendered,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
