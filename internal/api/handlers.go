package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/stuttgart-things/claim-machinery-api/internal/app"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/notify"
)

// OrderRequest represents a claim order request
type OrderRequest struct {
	Parameters map[string]interface{} `json:"parameters"`
	Author     string                 `json:"author,omitempty"`
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
	for _, tmpl := range s.getTemplateMap() {
		response.Items = append(response.Items, *tmpl)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// getTemplate returns a specific claim template by name
func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract template name from URL
	vars := mux.Vars(r)
	name := vars["name"]

	// Look up template
	tmpl, exists := s.getTemplateMap()[name]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "template not found",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tmpl)
}

// orderClaim renders a claim template with provided parameters
func (s *Server) orderClaim(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract template name from URL
	vars := mux.Vars(r)
	name := vars["name"]

	// Look up template
	tmpl, exists := s.getTemplateMap()[name]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "template not found",
		})
		return
	}

	// Parse request body
	var req OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid request body",
		})
		return
	}

	rendered, err := s.order(tmpl, req.Parameters, req.Author)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
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
	_ = json.NewEncoder(w).Encode(response)
}

// order renders a template with the user's parameters and sends the homerun
// notification. The API's order endpoint and the web UI both go through it.
func (s *Server) order(tmpl *claimtemplate.ClaimTemplate, userInput map[string]interface{}, author string) (string, error) {
	// Debug: log received parameters
	debugParams("Received from request", userInput)

	// Pre-seed user-provided params so valueFrom arg references (e.g. {{.networkKey}}) resolve correctly
	userParams := make(map[string]interface{}, len(userInput))
	for k, v := range userInput {
		userParams[k] = v
	}

	// Build parameter values (merge request params with defaults + function calls)
	params := app.BuildParameterValuesWithFunctionsAndUserParams(tmpl, s.GetFunctions(), userParams)

	// Debug: log merged parameters
	debugParams("After merge", params)

	// Render template directly with pre-built params (skip BuildParameterValues rebuild)
	rendered, err := s.render(tmpl, params)
	if err != nil {
		return "", err
	}

	// Send homerun notification asynchronously
	go func() {
		msg := notify.BuildOrderMessage(
			tmpl.Metadata.Name,
			tmpl.Metadata.Title,
			tmpl.Spec.Type,
			tmpl.Metadata.Tags,
			tmpl.Spec.Source,
			tmpl.Spec.Tag,
			params,
			author,
		)
		notify.SendHomerunMessage(s.homerun, msg)
	}()

	return rendered, nil
}
