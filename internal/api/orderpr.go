package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gorilla/mux"
	"github.com/stuttgart-things/claim-machinery-api/internal/auth"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/dispatch"
	notifypkg "github.com/stuttgart-things/claim-machinery-api/internal/notify"
)

// Order as PR (#131): instead of returning the rendered YAML, the order
// starts the GitOps repository's order workflow, which runs the claims CLI
// (`claims render --layout source-path --create-pr`) and opens the pull
// request. Off unless order auth (AUTH_OIDC_ISSUER) AND the dispatch
// credentials (GITHUB_DISPATCH_*) are configured: requested_by is always the
// verified login.

// maxReason bounds the free-text reason passed to the workflow.
const maxReason = 500

// orderPRMaxPolls stops the UI's polling after about ten minutes.
const orderPRMaxPolls = 120

var orderIDPattern = regexp.MustCompile(`^cma-[0-9a-f]{12}$`)

// SetDispatch turns on order as PR, together with SetAuth; nil, the
// default, leaves it off.
func (s *Server) SetDispatch(d *dispatch.Client) {
	s.dispatch = d
}

// orderPREnabled reports whether order as PR is on: it needs a verified
// identity for requested_by and the App credentials.
func (s *Server) orderPREnabled() bool {
	return s.auth != nil && s.dispatch != nil
}

// orderPRBlocked says why a template cannot be ordered as PR, or "".
//   - secrets: the UI does not handle them (client-side SOPS), and the
//     claims CLI's source-path layout refuses templates that define some.
//   - identity: user: the workflow renders through this API with its own
//     machine login, which would replace (and so contradict) the orderer.
func orderPRBlocked(tmpl *claimtemplate.ClaimTemplate) string {
	if len(tmpl.Spec.Secrets) > 0 {
		return "this template creates secrets, which an order as PR cannot carry; order it with the claims CLI"
	}
	for _, p := range tmpl.Spec.Parameters {
		if p.Identity == claimtemplate.IdentityUser {
			return fmt.Sprintf("parameter %s is bound to the login, which the order workflow cannot keep; order it with the claims CLI", p.Name)
		}
	}
	return ""
}

// orderPRResult is what an order as PR started.
type orderPRResult struct {
	OrderID     string
	Template    string
	RequestedBy string
	// Run is nil when the dispatch went through but the run did not show
	// up in time; WorkflowURL is linked instead.
	Run         *dispatch.Run
	WorkflowURL string
}

func newOrderID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "cma-" + hex.EncodeToString(b)
}

// cleanReason trims the reason, turns control characters into spaces and
// cuts it to maxReason characters.
func cleanReason(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, strings.TrimSpace(s))
	if r := []rune(s); len(r) > maxReason {
		s = string(r[:maxReason])
	}
	return s
}

// orderPR validates the order as POST .../order does (template identity
// rules, then a render without notification) and dispatches the workflow
// with the user's parameters: the workflow renders again through the claims
// CLI, so defaults and profile functions resolve there. id is never nil
// (orderPREnabled needs auth).
func (s *Server) orderPR(ctx context.Context, tmpl *claimtemplate.ClaimTemplate, params map[string]interface{}, id *auth.Identity, reason string) (*orderPRResult, *orderRefused) {
	name := tmpl.Metadata.Name
	if msg := orderPRBlocked(tmpl); msg != "" {
		return nil, &orderRefused{status: http.StatusUnprocessableEntity, msg: "order as PR not possible: " + msg}
	}
	params, refused := s.bindIdentity(tmpl, params, id)
	if refused != nil {
		return nil, refused
	}
	if params == nil {
		params = map[string]interface{}{}
	}
	// The workflow gets the checked parameters: values of parameters whose
	// showWhen does not match are dropped (#135).
	checked, err := claimtemplate.CheckParams(tmpl, params)
	if err != nil {
		status, body := orderErrorResponse(err, http.StatusUnprocessableEntity)
		return nil, &orderRefused{status: status, msg: body.Error, fields: body.Fields}
	}
	params = checked
	logOrder(name, id, id.Name)
	if _, err := s.order(tmpl, params, id.Name, false); err != nil {
		// Invalid parameters and module asserts are 400, a render that
		// fails otherwise 422 as before.
		status, body := orderErrorResponse(err, http.StatusUnprocessableEntity)
		return nil, &orderRefused{status: status, msg: body.Error, fields: body.Fields}
	}

	o := dispatch.Order{ID: newOrderID(), Template: name, Params: params, RequestedBy: id.Name, Reason: cleanReason(reason)}
	run, err := s.dispatch.Dispatch(ctx, o)
	if err != nil {
		log.Printf("[order-pr] %s (%s) by %q: %v", name, o.ID, id.Name, err)
		return nil, &orderRefused{status: http.StatusBadGateway, msg: "starting the order workflow failed"}
	}
	res := &orderPRResult{OrderID: o.ID, Template: name, RequestedBy: id.Name, Run: run, WorkflowURL: s.dispatch.WorkflowURL()}
	link := res.WorkflowURL
	if run != nil {
		link = run.URL
	}
	log.Printf("[order-pr] %s (%s) by %q dispatched: %s", name, o.ID, id.Name, link)

	go func() {
		s.sendNotification(notifypkg.BuildOrderPRMessage(name, tmpl.Metadata.Title, tmpl.Spec.Type,
			tmpl.Metadata.Tags, params, id.Name, o.ID, link))
	}()
	return res, nil
}

// OrderPRRequest is the body of POST .../order-pr.
type OrderPRRequest struct {
	Parameters map[string]interface{} `json:"parameters"`
	Reason     string                 `json:"reason,omitempty"`
}

// OrderPRRun is the dispatched workflow run.
type OrderPRRun struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

// OrderPRResponse answers POST .../order-pr (202: the PR is opened by the
// run, not yet).
type OrderPRResponse struct {
	APIVersion  string                 `json:"apiVersion"`
	Kind        string                 `json:"kind"`
	Metadata    map[string]interface{} `json:"metadata"`
	OrderID     string                 `json:"orderId"`
	Template    string                 `json:"template"`
	RequestedBy string                 `json:"requestedBy"`
	Run         *OrderPRRun            `json:"run,omitempty"`
	WorkflowURL string                 `json:"workflowUrl"`
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// orderClaimPR is POST /api/v1/claim-templates/{name}/order-pr.
func (s *Server) orderClaimPR(w http.ResponseWriter, r *http.Request) {
	if !s.orderPREnabled() {
		http.NotFound(w, r)
		return
	}
	id, ok := s.authorizeAPIOrder(w, r)
	if !ok {
		return
	}
	tmpl, exists := s.getTemplateMap()[mux.Vars(r)["name"]]
	if !exists {
		writeJSONError(w, http.StatusNotFound, "template not found")
		return
	}
	var req OrderPRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	res, refused := s.orderPR(r.Context(), tmpl, req.Parameters, id, req.Reason)
	if refused != nil {
		writeParamError(w, refused.status, ParamErrorResponse{Error: refused.msg, Fields: refused.fields})
		return
	}
	resp := OrderPRResponse{
		APIVersion: "api.claim-machinery.io/v1alpha1",
		Kind:       "OrderPRResponse",
		Metadata: map[string]interface{}{
			"name":      res.OrderID,
			"timestamp": time.Now().Format(time.RFC3339),
		},
		OrderID:     res.OrderID,
		Template:    res.Template,
		RequestedBy: res.RequestedBy,
		WorkflowURL: res.WorkflowURL,
	}
	if res.Run != nil {
		resp.Run = &OrderPRRun{ID: res.Run.ID, URL: res.Run.URL}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}

// uiOrderPRData is the orderpr.html fragment: the dispatched order and,
// while it is not settled, the URL htmx polls.
type uiOrderPRData struct {
	Name        string
	OrderID     string
	RequestedBy string
	RunURL      string
	WorkflowURL string
	PRURL       string
	// State is the badge: dispatched, running, pull request opened, failed.
	State string
	Note  string
	// PollURL, when set, is fetched again in a few seconds.
	PollURL string
}

func (s *Server) pollURL(name, orderID string, runID int64, n int) string {
	q := url.Values{"template": {name}, "n": {strconv.Itoa(n)}}
	if runID != 0 {
		q.Set("run", strconv.FormatInt(runID, 10))
	}
	return s.basePath + "/ui/order-pr/" + orderID + "?" + q.Encode()
}

// uiOrderPR is the form's "Order as PR": the identity is checked as for
// Order, the form parsed as for Order, then orderPR dispatches.
func (s *Server) uiOrderPR(w http.ResponseWriter, r *http.Request, tmpl *claimtemplate.ClaimTemplate) {
	result := uiResultData{Base: s.basePath, Name: tmpl.Metadata.Name, ErrorTitle: "Order as PR not possible"}
	fail := func(status int, msg string) {
		result.Error = msg
		if r.Header.Get("HX-Request") != "true" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(status)
		}
		s.renderUI(w, "result.html", result)
	}
	if !s.orderPREnabled() {
		fail(http.StatusNotFound, "order as PR is not enabled")
		return
	}
	login, err := s.auth.AuthorizeUI(r)
	if err != nil {
		status := authStatus(err)
		log.Printf("[auth] ui order-pr %s refused (%d): %v", tmpl.Metadata.Name, status, err)
		fail(status, publicAuthError(err))
		return
	}
	params, err := formParams(tmpl, r.PostForm)
	if err != nil {
		fail(http.StatusBadRequest, err.Error())
		return
	}
	res, refused := s.orderPR(r.Context(), tmpl, params, &login, r.PostForm.Get("_reason"))
	if refused != nil {
		fail(refused.status, refused.text())
		return
	}
	data := uiOrderPRData{
		Name: res.Template, OrderID: res.OrderID, RequestedBy: res.RequestedBy,
		WorkflowURL: res.WorkflowURL, State: "dispatched",
	}
	var runID int64
	if res.Run != nil {
		data.RunURL, runID = res.Run.URL, res.Run.ID
	}
	data.PollURL = s.pollURL(res.Template, res.OrderID, runID, 1)
	s.renderUI(w, "orderpr.html", data)
}

// uiOrderPRStatus is GET /ui/order-pr/{orderID}: the run's state and, once
// it succeeded, the pull request it opened. htmx polls it until then.
func (s *Server) uiOrderPRStatus(w http.ResponseWriter, r *http.Request) {
	orderID := mux.Vars(r)["orderID"]
	if !s.orderPREnabled() || !orderIDPattern.MatchString(orderID) {
		http.NotFound(w, r)
		return
	}
	if _, err := s.auth.AuthorizeUI(r); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(authStatus(err))
		s.renderUI(w, "result.html", uiResultData{ErrorTitle: "Order status not available", Error: publicAuthError(err)})
		return
	}
	q := r.URL.Query()
	name := q.Get("template")
	n, _ := strconv.Atoi(q.Get("n"))
	runID, _ := strconv.ParseInt(q.Get("run"), 10, 64)
	data := uiOrderPRData{Name: name, OrderID: orderID, WorkflowURL: s.dispatch.WorkflowURL(), State: "dispatched"}

	ctx := r.Context()
	var run *dispatch.Run
	var err error
	if runID != 0 {
		run, err = s.dispatch.GetRun(ctx, runID)
	} else {
		run, err = s.dispatch.FindRun(ctx, orderID)
	}
	switch {
	case err != nil:
		log.Printf("[order-pr] status of %s: %v", orderID, err)
		data.Note = "The run's status could not be read; look at the workflow."
	case run == nil:
		data.Note = "Waiting for the workflow run to start."
	default:
		runID, data.RunURL = run.ID, run.URL
		data.State = "running"
		if run.Done() {
			if run.Conclusion != "success" {
				data.State = "failed"
				data.Note = fmt.Sprintf("The run ended %s; its log says why.", run.Conclusion)
				s.renderUI(w, "orderpr.html", data)
				return
			}
			pr, err := s.dispatch.FindPR(ctx, orderID)
			if err != nil {
				log.Printf("[order-pr] PR of %s: %v", orderID, err)
				data.State = "run succeeded"
				data.Note = "The pull request link could not be read; the run's log names it."
				s.renderUI(w, "orderpr.html", data)
				return
			}
			if pr != "" {
				data.State, data.PRURL = "pull request opened", pr
				s.renderUI(w, "orderpr.html", data)
				return
			}
			data.Note = "The run succeeded; looking for its pull request."
		}
	}
	if n >= orderPRMaxPolls {
		data.Note = "Stopped watching; follow the run on GitHub."
	} else {
		data.PollURL = s.pollURL(name, orderID, runID, n+1)
	}
	s.renderUI(w, "orderpr.html", data)
}
