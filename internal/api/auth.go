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
		status := authStatus(err)
		log.Printf("[auth] order %s refused (%d): %v", r.URL.Path, status, err)
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
		return nil, false
	}
	return &id, true
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
