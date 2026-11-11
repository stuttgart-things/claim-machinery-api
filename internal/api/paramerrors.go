package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/render"
)

// ParamErrorResponse is the 400 body of an order whose parameters break the
// template's rules (#135): {"error":"invalid parameters","fields":{...}}.
// A failed assert of the KCL module answers {"error":"<assert message>"}.
type ParamErrorResponse struct {
	Error  string                    `json:"error"`
	Fields claimtemplate.FieldErrors `json:"fields,omitempty"`
}

// errInvalidParameters is the error text of a field validation failure.
const errInvalidParameters = "invalid parameters"

// orderErrorResponse maps an error of Server.order to a status and body:
// invalid parameters and failed module asserts are the caller's (400), any
// other render failure is ours (fallback, e.g. 500).
func orderErrorResponse(err error, fallback int) (int, ParamErrorResponse) {
	var verr *claimtemplate.ValidationError
	if errors.As(err, &verr) {
		return http.StatusBadRequest, ParamErrorResponse{Error: errInvalidParameters, Fields: verr.Fields}
	}
	var aerr *render.AssertError
	if errors.As(err, &aerr) {
		return http.StatusBadRequest, ParamErrorResponse{Error: aerr.Message}
	}
	return fallback, ParamErrorResponse{Error: err.Error()}
}

func writeParamError(w http.ResponseWriter, status int, body ParamErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
