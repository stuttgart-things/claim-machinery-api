package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stuttgart-things/claim-machinery-api/internal/auth"
	"github.com/stuttgart-things/claim-machinery-api/internal/auth/authtest"
	"github.com/stuttgart-things/claim-machinery-api/internal/notify"
)

const orderAudience = "claim-machinery-api"

// newAuthTestServer is the UI test server with order auth against a fake
// issuer; mod adjusts the auth config. Rendered parameters are recorded in
// *rendered, notifications in the channel.
func newAuthTestServer(t *testing.T, basePath string, mod func(*auth.Config)) (*Server, *authtest.Issuer, *map[string]interface{}, chan notify.HomerunMessage) {
	t.Helper()
	iss := authtest.NewIssuer(t)
	s, rendered := newUITestServer(t, basePath)
	cfg := auth.Config{Issuer: iss.URL(), Audiences: []string{orderAudience}, CAFile: iss.CAFile}
	if mod != nil {
		mod(&cfg)
	}
	a, err := auth.New(cfg)
	require.NoError(t, err)
	s.SetAuth(a)
	return s, iss, rendered, captureNotifications(s)
}

func apiOrder(s *Server, path, body string, header ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

const orderBody = `{"parameters":{"vmName":"web1"}}`

func expectNotification(t *testing.T, sent chan notify.HomerunMessage) notify.HomerunMessage {
	t.Helper()
	select {
	case msg := <-sent:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("no notification sent")
		return notify.HomerunMessage{}
	}
}

func expectNoNotification(t *testing.T, sent chan notify.HomerunMessage) {
	t.Helper()
	select {
	case msg := <-sent:
		t.Fatalf("unexpected notification: %+v", msg)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestOrderAuthOff(t *testing.T) {
	s, _ := newUITestServer(t, "")
	sent := captureNotifications(s)
	w := apiOrder(s, "/api/v1/claim-templates/vm/order", orderBody)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "claim-machinery-api", expectNotification(t, sent).Author, "default author, as before")
}

func TestOrderAuthAPI(t *testing.T) {
	s, iss, rendered, sent := newAuthTestServer(t, "", nil)
	path := "/api/v1/claim-templates/vm/order"

	// No token: 401 with a Bearer challenge, nothing rendered or sent.
	w := apiOrder(s, path, orderBody)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, `Bearer realm="claim-machinery-api"`, w.Header().Get("WWW-Authenticate"))
	assert.Contains(t, w.Body.String(), "bearer token required")
	assert.Nil(t, *rendered)

	// The portal's identity header never authorizes the API.
	w = apiOrder(s, path, orderBody, "X-Forwarded-User", "patrick", "X-Forwarded-Groups", "portal-admin")
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	expired := iss.Claims(orderAudience)
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	for name, token := range map[string]string{
		"bad signature": iss.Sign(t, iss.Claims(orderAudience))[:20] + "x" + iss.Sign(t, iss.Claims(orderAudience))[21:],
		"wrong aud":     iss.Sign(t, iss.Claims("someone-else")),
		"expired":       iss.Sign(t, expired),
	} {
		w = apiOrder(s, path, orderBody, "Authorization", "Bearer "+token)
		assert.Equal(t, http.StatusUnauthorized, w.Code, name)
		assert.Contains(t, w.Header().Get("WWW-Authenticate"), `error="invalid_token"`, name)
		assert.Equal(t, "invalid token", decodeError(t, w), name)
	}
	assert.Nil(t, *rendered)
	expectNoNotification(t, sent)

	// Valid token: ordered, the token's user is the author.
	w = apiOrder(s, path, orderBody, "Authorization", "Bearer "+iss.Sign(t, iss.Claims(orderAudience)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "web1", (*rendered)["vmName"])
	assert.Equal(t, "patrick@sthings.example", expectNotification(t, sent).Author)

	// An author in the body is kept.
	w = apiOrder(s, path, `{"parameters":{"vmName":"web1"},"author":"backstage"}`, "Authorization", "Bearer "+iss.Sign(t, iss.Claims(orderAudience)))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "backstage", expectNotification(t, sent).Author)
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body["error"]
}

func TestOrderAuthRoles(t *testing.T) {
	s, iss, rendered, _ := newAuthTestServer(t, "", func(c *auth.Config) { c.OrderRoles = []string{"portal-admin"} })
	path := "/api/v1/claim-templates/vm/order"

	viewer := iss.Claims(orderAudience)
	viewer[auth.ZitadelRolesClaim] = map[string]any{"portal-viewer": map[string]any{"1": "org"}}
	w := apiOrder(s, path, orderBody, "Authorization", "Bearer "+iss.Sign(t, viewer))
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "forbidden: missing order role", decodeError(t, w))
	assert.Nil(t, *rendered)

	admin := iss.Claims(orderAudience)
	admin[auth.ZitadelRolesClaim] = map[string]any{"portal-admin": map[string]any{"1": "org"}}
	w = apiOrder(s, path, orderBody, "Authorization", "Bearer "+iss.Sign(t, admin))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrderAuthUnderBasePath(t *testing.T) {
	s, iss, _, _ := newAuthTestServer(t, "/claims", nil)
	for _, path := range []string{"/claims/api/v1/claim-templates/vm/order", "/api/v1/claim-templates/vm/order"} {
		assert.Equal(t, http.StatusUnauthorized, apiOrder(s, path, orderBody).Code, path)
		w := apiOrder(s, path, orderBody, "Authorization", "Bearer "+iss.Sign(t, iss.Claims(orderAudience)))
		assert.Equal(t, http.StatusOK, w.Code, path)
	}
}

func TestOrderAuthLeavesReadsOpen(t *testing.T) {
	s, _, _, _ := newAuthTestServer(t, "", nil)
	for _, path := range []string{"/health", "/version", "/api/v1/claim-templates", "/api/v1/claim-templates/vm", "/ui/", "/ui/templates/vm"} {
		assert.Equal(t, http.StatusOK, serve(s, http.MethodGet, path, nil).Code, path)
	}
}

func TestOrderAuthIssuerUnavailable(t *testing.T) {
	s, iss, _, _ := newAuthTestServer(t, "", nil)
	token := iss.Sign(t, iss.Claims(orderAudience))
	iss.Server.Close()
	w := apiOrder(s, "/api/v1/claim-templates/vm/order", orderBody, "Authorization", "Bearer "+token)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func uiOrderForm(action string) url.Values {
	return url.Values{"vmName": {"web1"}, "_action": {action}}
}

func TestOrderAuthUI(t *testing.T) {
	path := "/ui/templates/vm/render"

	t.Run("preview stays open", func(t *testing.T) {
		s, _, rendered, sent := newAuthTestServer(t, "", nil)
		w := serve(s, http.MethodPost, path, uiOrderForm("preview"))
		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "preview · no notification")
		assert.Equal(t, "web1", (*rendered)["vmName"])
		expectNoNotification(t, sent)
	})

	t.Run("order without identity", func(t *testing.T) {
		s, _, rendered, sent := newAuthTestServer(t, "", func(c *auth.Config) { c.TrustForwardedUser = true })
		w := serve(s, http.MethodPost, path, uiOrderForm("order"))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Order not allowed")

		// htmx gets the error fragment with 200 so it is shown.
		w = serve(s, http.MethodPost, path, uiOrderForm("order"), "HX-Request", "true")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "login required to order")
		assert.Nil(t, *rendered)
		expectNoNotification(t, sent)
	})

	t.Run("forwarded user not trusted", func(t *testing.T) {
		s, _, rendered, sent := newAuthTestServer(t, "", nil)
		w := serve(s, http.MethodPost, path, uiOrderForm("order"), "X-Forwarded-User", "patrick")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Nil(t, *rendered)
		expectNoNotification(t, sent)
	})

	t.Run("forwarded user trusted", func(t *testing.T) {
		s, _, rendered, sent := newAuthTestServer(t, "", func(c *auth.Config) {
			c.TrustForwardedUser = true
			c.OrderRoles = []string{"portal-admin"}
		})
		w := serve(s, http.MethodPost, path, uiOrderForm("order"), "X-Forwarded-User", "patrick", "X-Forwarded-Groups", "portal-admin")
		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ordered · notification sent")
		assert.Equal(t, "web1", (*rendered)["vmName"])
		assert.Equal(t, "patrick", expectNotification(t, sent).Author)

		// The form's author wins.
		form := uiOrderForm("order")
		form.Set("_author", "team-a")
		serve(s, http.MethodPost, path, form, "X-Forwarded-User", "patrick", "X-Forwarded-Groups", "portal-admin")
		assert.Equal(t, "team-a", expectNotification(t, sent).Author)
	})

	t.Run("forwarded user without role", func(t *testing.T) {
		s, _, rendered, sent := newAuthTestServer(t, "", func(c *auth.Config) {
			c.TrustForwardedUser = true
			c.OrderRoles = []string{"portal-admin"}
		})
		w := serve(s, http.MethodPost, path, uiOrderForm("order"), "X-Forwarded-User", "patrick", "X-Forwarded-Groups", "portal-viewer")
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Nil(t, *rendered)
		expectNoNotification(t, sent)
	})

	t.Run("bearer token", func(t *testing.T) {
		s, iss, _, sent := newAuthTestServer(t, "", nil)
		w := serve(s, http.MethodPost, path, uiOrderForm("order"), "Authorization", "Bearer "+iss.Sign(t, iss.Claims(orderAudience)))
		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "patrick@sthings.example", expectNotification(t, sent).Author)
	})
}
