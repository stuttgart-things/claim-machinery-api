package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/stuttgart-things/claim-machinery-api/internal/app"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	notifypkg "github.com/stuttgart-things/claim-machinery-api/internal/notify"
	"github.com/stuttgart-things/claim-machinery-api/internal/render"
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
	// With order auth on, the bearer token is checked before anything else.
	id, ok := s.authorizeAPIOrder(w, r)
	if !ok {
		return
	}
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

	author := orderAuthor(req.Author, id)
	params, refused := s.bindIdentity(tmpl, req.Parameters, id)
	if refused != nil {
		w.WriteHeader(refused.status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": refused.msg})
		return
	}
	logOrder(name, id, author)
	rendered, err := s.order(tmpl, params, author, true)
	if err != nil {
		// 400 for invalid parameters (with the fields) and failed module
		// asserts (only their message), 500 for anything else (#135).
		status, body := orderErrorResponse(err, http.StatusInternalServerError)
		writeParamError(w, status, body)
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

// order renders a template with the user's parameters and, when notify is
// set, sends the homerun notification. The API's order endpoint and the web
// UI both go through it; the UI's preview renders without notifying.
func (s *Server) order(tmpl *claimtemplate.ClaimTemplate, userInput map[string]interface{}, author string, notify bool) (string, error) {
	// Debug: log received parameters
	debugParams("Received from request", userInput)

	// Check the template's rules before KCL (#135): drop values of
	// parameters whose showWhen does not match, check required, enum
	// (narrowed by enumWhen), pattern and length. The result is a copy, so
	// valueFrom arg references (e.g. {{.networkKey}}) resolve against it.
	userParams, err := claimtemplate.CheckParams(tmpl, userInput)
	if err != nil {
		return "", err
	}

	// Build parameter values (merge request params with defaults + function calls)
	params := app.BuildParameterValuesWithFunctionsAndUserParams(tmpl, s.GetFunctions(), userParams)

	// Debug: log merged parameters
	debugParams("After merge", params)

	// Render template directly with pre-built params (skip BuildParameterValues rebuild)
	rendered, err := s.render(tmpl, params)
	if err != nil {
		var aerr *render.AssertError
		if errors.As(err, &aerr) {
			// A rule of the module failed: the caller gets only its message.
			log.Printf("[order] %s: module assert failed: %s\n%s", tmpl.Metadata.Name, aerr.Message, aerr.Stderr)
			return "", aerr
		}
		return "", err
	}
	if !notify {
		return rendered, nil
	}

	// Send homerun notification asynchronously
	go func() {
		msg := notifypkg.BuildOrderMessage(
			tmpl.Metadata.Name,
			tmpl.Metadata.Title,
			tmpl.Spec.Type,
			tmpl.Metadata.Tags,
			tmpl.Spec.Source,
			tmpl.Spec.Tag,
			params,
			author,
		)
		s.sendNotification(msg)
	}()

	return rendered, nil
}
