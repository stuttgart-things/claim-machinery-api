package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/stuttgart-things/claim-machinery-api/internal/auth"
)

// SetAuth turns on the order auth (#119); nil, the default, leaves ordering
// open as before. Read endpoints, the UI's preview and /health never need it.
func (s *Server) SetAuth(a *auth.Authenticator) {
	s.auth = a
}

// authStatus maps an auth error to its HTTP status.
func authStatus(err error) int {
	switch {
	case errors.Is(err, auth.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, auth.ErrUnavailable):
		return http.StatusServiceUnavailable
	default:
		return http.StatusUnauthorized
	}
}

// authorizeAPIOrder checks the bearer token of POST .../order. It returns
// false after writing the error response; the identity is nil when auth is
// off.
func (s *Server) authorizeAPIOrder(w http.ResponseWriter, r *http.Request) (*auth.Identity, bool) {
	if s.auth == nil {
		return nil, true
	}
	id, err := s.auth.Authorize(r)
	if err != nil {
		s.writeAuthError(w, r, err)
		return nil, false
	}
	return &id, true
}

// writeAuthError answers a refused bearer token: the status, a challenge
// for 401, and the public error.
func (s *Server) writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	status := authStatus(err)
	log.Printf("[auth] %s refused (%d): %v", r.URL.Path, status, err)
	if status == http.StatusUnauthorized {
		challenge := `Bearer realm="claim-machinery-api"`
		if !errors.Is(err, auth.ErrNoToken) {
			challenge += `, error="invalid_token"`
		}
		w.Header().Set("WWW-Authenticate", challenge)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": publicAuthError(err)})
}

// publicAuthError is what a refused caller is told; the details (which
// check failed) go to the log only.
func publicAuthError(err error) string {
	switch {
	case errors.Is(err, auth.ErrForbidden):
		return "forbidden: missing order role"
	case errors.Is(err, auth.ErrUnavailable):
		return "identity provider unavailable"
	case errors.Is(err, auth.ErrNoToken):
		return "bearer token required"
	case errors.Is(err, auth.ErrNoIdentity):
		return "login required to order"
	default:
		return "invalid token"
	}
}

// orderAuthor is the author named in the order notification: the one the
// caller gave, else the authenticated user.
func orderAuthor(given string, id *auth.Identity) string {
	if given != "" || id == nil {
		return given
	}
	return id.Name
}

func logOrder(template string, id *auth.Identity, author string) {
	if id == nil {
		return
	}
	log.Printf("[auth] order %s by %q (sub=%q, via %s, author=%q)", template, id.Name, id.Subject, id.Source, author)
}

// WhoamiResponse is the caller as the API sees it (#147): the identity the
// order binds and the groups its roles map to, so clients show them
// without knowing the role mapping.
type WhoamiResponse struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Name       string   `json:"name"`
	Subject    string   `json:"subject,omitempty"`
	UserRef    string   `json:"userRef,omitempty"`
	Roles      []string `json:"roles"`
	Groups     []string `json:"groups"`
	Admin      bool     `json:"admin"`
	// MayOrder is false when AUTH_ORDER_ROLES excludes the caller.
	MayOrder bool `json:"mayOrder"`
}

// whoami returns the bearer token's identity (GET /api/v1/whoami); 404
// while order auth is off.
func (s *Server) whoami(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order auth is off"})
		return
	}
	id, err := s.auth.Authorize(r)
	mayOrder := true
	if errors.Is(err, auth.ErrForbidden) {
		// A valid token without an order role: still say who it is.
		mayOrder, err = false, nil
	}
	if err != nil {
		s.writeAuthError(w, r, err)
		return
	}
	cfg := s.auth.Config()
	ref, _ := id.UserRef()
	resp := WhoamiResponse{
		APIVersion: "api.claim-machinery.io/v1alpha1",
		Kind:       "Identity",
		Name:       id.Name,
		Subject:    id.Subject,
		UserRef:    ref,
		Roles:      append([]string{}, id.Roles...),
		Groups:     cfg.Groups(id),
		Admin:      cfg.IsAdmin(id),
		MayOrder:   mayOrder,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
