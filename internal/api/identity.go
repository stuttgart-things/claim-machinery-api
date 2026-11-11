package api

import (
	"fmt"
	"log"
	"net/http"
	"slices"

	"github.com/stuttgart-things/claim-machinery-api/internal/auth"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
)

// orderRefused is an order the identity rules refuse: the status and what
// the caller is told.
type orderRefused struct {
	status int
	msg    string
	// fields, for invalid parameters (#135), name what is wrong per field.
	fields claimtemplate.FieldErrors
}

func (e *orderRefused) Error() string { return e.msg }

// text is msg with the fields, for the web UI, which shows one line.
func (e *orderRefused) text() string {
	if len(e.fields) == 0 {
		return e.msg
	}
	return (&claimtemplate.ValidationError{Fields: e.fields}).Error()
}

// bindIdentity applies the template's identity rules to an order (#122):
// its orderRoles, and the parameters bound to the orderer (identity: user /
// owner). params is not changed; the result is. Without auth (id nil) the
// order is left as it is.
func (s *Server) bindIdentity(tmpl *claimtemplate.ClaimTemplate, params map[string]interface{}, id *auth.Identity) (map[string]interface{}, *orderRefused) {
	if id == nil || s.auth == nil {
		return params, nil
	}
	cfg := s.auth.Config()
	name := tmpl.Metadata.Name

	if !cfg.MayOrder(*id, tmpl.Spec.OrderRoles) {
		log.Printf("[auth] order %s refused (403): %s has none of the template's roles %v", name, id.Name, tmpl.Spec.OrderRoles)
		return nil, &orderRefused{status: http.StatusForbidden, msg: "forbidden: missing role for this template"}
	}

	out := make(map[string]interface{}, len(params)+2)
	for k, v := range params {
		out[k] = v
	}

	var ref string
	for _, p := range tmpl.Spec.Parameters {
		if p.Identity == "" {
			continue
		}
		if ref == "" {
			r, err := id.UserRef()
			if err != nil {
				log.Printf("[auth] order %s refused (403): %v", name, err)
				return nil, &orderRefused{status: http.StatusForbidden, msg: "forbidden: the login is not a Backstage user name"}
			}
			ref = r
		}
		given, set := params[p.Name]
		value, isString := given.(string)
		if set && !isString {
			return nil, &orderRefused{status: http.StatusBadRequest, msg: fmt.Sprintf("%s: want a string", p.Name)}
		}

		switch p.Identity {
		case claimtemplate.IdentityUser:
			if value != "" && value != ref {
				log.Printf("[auth] order %s refused (400): %s sent %s=%q, the login is %s", name, id.Name, p.Name, value, ref)
				return nil, &orderRefused{status: http.StatusBadRequest, msg: fmt.Sprintf("%s is set from the login (%s); leave it out", p.Name, ref)}
			}
			out[p.Name] = ref

		case claimtemplate.IdentityOwner:
			switch {
			case value == "":
				out[p.Name] = ref
			case value == ref, cfg.IsAdmin(*id), slices.Contains(cfg.Groups(*id), value):
				// own ref, an admin, or one of the login's groups
			default:
				log.Printf("[auth] order %s refused (403): %s may not order for %s=%q (groups %v)", name, id.Name, p.Name, value, cfg.Groups(*id))
				return nil, &orderRefused{status: http.StatusForbidden, msg: fmt.Sprintf("forbidden: %s %q is neither the login nor one of its groups", p.Name, value)}
			}

		default:
			log.Printf("[auth] template %s: parameter %s has unknown identity %q", name, p.Name, p.Identity)
			return nil, &orderRefused{status: http.StatusInternalServerError, msg: fmt.Sprintf("template %s: parameter %s has unknown identity %q", name, p.Name, p.Identity)}
		}
	}
	return out, nil
}
