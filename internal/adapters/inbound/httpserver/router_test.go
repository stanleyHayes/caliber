package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter(t *testing.T) {
	gw := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(299) })
	r := NewRouter(gw)

	do := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	h := do("/healthz")
	require.Equal(t, http.StatusOK, h.Code)
	assert.JSONEq(t, `{"status":"ok"}`, h.Body.String())

	rz := do("/readyz")
	require.Equal(t, http.StatusOK, rz.Code)
	assert.JSONEq(t, `{"status":"ready"}`, rz.Body.String())

	gw299 := do("/v1/anything")
	assert.Equal(t, 299, gw299.Code, "should route under /v1 to the gateway handler")
}
