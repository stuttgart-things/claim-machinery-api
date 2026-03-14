package functions

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubstituteVars(t *testing.T) {
	args := map[string]string{
		"networkKey": "10.31.103",
		"cluster":    "infra",
	}

	result := substituteVars("http://clusterbook:8080/api/v1/networks/${networkKey}/assign", args)
	require.Equal(t, "http://clusterbook:8080/api/v1/networks/10.31.103/assign", result)
}

func TestSubstituteVars_EnvFallback(t *testing.T) {
	t.Setenv("CLUSTERBOOK_URL", "http://localhost:9090")
	args := map[string]string{"networkKey": "10.31.103"}

	result := substituteVars("${CLUSTERBOOK_URL}/api/v1/networks/${networkKey}/ips", args)
	require.Equal(t, "http://localhost:9090/api/v1/networks/10.31.103/ips", result)
}

func TestSubstituteBodyVars(t *testing.T) {
	body := map[string]any{
		"cluster":    "${cluster}",
		"status":     "${status}",
		"create_dns": "${createDns}",
	}
	args := map[string]string{
		"cluster":   "my-cluster",
		"status":    "ASSIGNED",
		"createDns": "true",
	}

	result := substituteBodyVars(body, args)
	require.Equal(t, "my-cluster", result["cluster"])
	require.Equal(t, "ASSIGNED", result["status"])
	require.Equal(t, true, result["create_dns"])
}

func TestExtractValue_Simple(t *testing.T) {
	data := []byte(`{"ip":"10.31.103.6","status":"ASSIGNED"}`)
	val, err := extractValue(data, "ip")
	require.NoError(t, err)
	require.Equal(t, "10.31.103.6", val)
}

func TestExtractValue_Nested(t *testing.T) {
	data := []byte(`{"data":{"address":"10.31.103.6"}}`)
	val, err := extractValue(data, "data.address")
	require.NoError(t, err)
	require.Equal(t, "10.31.103.6", val)
}

func TestExtractValue_Empty(t *testing.T) {
	data := []byte(`"10.31.103.6"`)
	val, err := extractValue(data, "")
	require.NoError(t, err)
	require.Equal(t, "10.31.103.6", val)
}

func TestExtractValue_KeyNotFound(t *testing.T) {
	data := []byte(`{"ip":"10.31.103.6"}`)
	_, err := extractValue(data, "address")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestRegistry_Resolve_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/networks/10.31.103/ips", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{
			{"ip": "10.31.103.6", "digit": "6", "status": "", "cluster": ""},
		})
	}))
	defer srv.Close()

	reg := NewRegistry([]FunctionDef{
		{
			Name:   "get-available-ip",
			Type:   "http",
			Method: "GET",
			URL:    srv.URL + "/api/v1/networks/${networkKey}/ips",
			Response: ResponseExtract{
				Extract: "",
			},
		},
	})

	val, err := reg.Resolve(&ValueFromSpec{
		Function: "get-available-ip",
		Args:     map[string]string{"networkKey": "10.31.103"},
	})
	require.NoError(t, err)
	require.Contains(t, val, "10.31.103.6")
}

func TestRegistry_Resolve_POST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/networks/192.168.10/assign", r.URL.Path)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		require.Equal(t, "infra", body["cluster"])
		require.Equal(t, "ASSIGNED", body["status"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"ip":      "192.168.10.150",
			"cluster": "infra",
			"status":  "ASSIGNED",
		})
	}))
	defer srv.Close()

	reg := NewRegistry([]FunctionDef{
		{
			Name:   "assign-ip",
			Type:   "http",
			Method: "POST",
			URL:    srv.URL + "/api/v1/networks/${networkKey}/assign",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: map[string]any{
				"cluster":    "${cluster}",
				"status":     "${status}",
				"create_dns": "${createDns}",
			},
			Response: ResponseExtract{Extract: "ip"},
		},
	})

	val, err := reg.Resolve(&ValueFromSpec{
		Function: "assign-ip",
		Args: map[string]string{
			"networkKey": "192.168.10",
			"cluster":    "infra",
			"status":     "ASSIGNED",
			"createDns":  "false",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "192.168.10.150", val)
}

func TestRegistry_Resolve_FunctionNotFound(t *testing.T) {
	reg := NewRegistry(nil)
	_, err := reg.Resolve(&ValueFromSpec{Function: "nonexistent"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestRegistry_Has(t *testing.T) {
	reg := NewRegistry([]FunctionDef{{Name: "test-fn", Type: "http"}})
	require.True(t, reg.Has("test-fn"))
	require.False(t, reg.Has("missing"))
}

func TestRegistry_Nil(t *testing.T) {
	var reg *Registry
	require.False(t, reg.Has("anything"))
	_, err := reg.Resolve(&ValueFromSpec{Function: "test"})
	require.Error(t, err)
}
