package api

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"

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
			groups := cfg.Groups(*id)
			if g, ok := bareGroup(groups, value); ok {
				value = g
				out[p.Name] = g
			}
			switch {
			case value == "":
				out[p.Name] = ref
			case value == ref, cfg.IsAdmin(*id), slices.Contains(groups, value):
				// own ref, an admin, or one of the login's groups
			default:
				log.Printf("[auth] order %s refused (403): %s may not order for %s=%q (groups %v)", name, id.Name, p.Name, value, groups)
				return nil, &orderRefused{status: http.StatusForbidden, msg: fmt.Sprintf("forbidden: %s %q is neither the login nor one of its groups; allowed: %s",
					p.Name, value, strings.Join(append([]string{ref}, groups...), ", "))}
			}

		default:
			log.Printf("[auth] template %s: parameter %s has unknown identity %q", name, p.Name, p.Identity)
			return nil, &orderRefused{status: http.StatusInternalServerError, msg: fmt.Sprintf("template %s: parameter %s has unknown identity %q", name, p.Name, p.Identity)}
		}
	}
	return out, nil
}

// bareGroup resolves an owner given as a bare group name (ai-personas) to
// the one group of the login with that name (group:default/ai-personas)
// (#147). ok is false for a full ref, or when none or several match.
func bareGroup(groups []string, value string) (string, bool) {
	if value == "" || strings.Contains(value, ":") || strings.Contains(value, "/") {
		return "", false
	}
	var found string
	for _, g := range groups {
		if strings.HasSuffix(g, "/"+value) {
			if found != "" {
				return "", false
			}
			found = g
		}
	}
	return found, found != ""
}

// withAllowedOwners is tmpl for the caller id (#147): a copy whose
// identity: owner parameters list what id may order for, the list the web
// UI offers. tmpl itself when it has no such parameter or id has no valid
// user ref.
func withAllowedOwners(tmpl *claimtemplate.ClaimTemplate, cfg auth.Config, id auth.Identity) *claimtemplate.ClaimTemplate {
	ref, err := id.UserRef()
	if err != nil {
		return tmpl
	}
	var params []claimtemplate.Parameter
	for i, p := range tmpl.Spec.Parameters {
		if p.Identity != claimtemplate.IdentityOwner {
			continue
		}
		if params == nil {
			params = slices.Clone(tmpl.Spec.Parameters)
		}
		params[i].AllowedOwners = append([]string{ref}, cfg.Groups(id)...)
		params[i].AnyOwner = cfg.IsAdmin(id)
	}
	if params == nil {
		return tmpl
	}
	out := *tmpl
	out.Spec.Parameters = params
	return &out
}
