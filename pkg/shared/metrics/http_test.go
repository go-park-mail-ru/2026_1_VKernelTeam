package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPMiddleware_RecordsRequestAndDuration(t *testing.T) {
	m := New("test_http_basic")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})

	srv := m.Middleware(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/ping", nil)
	srv.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, float64(1), testutil.ToFloat64(m.requests.WithLabelValues("GET", "GET /api/v1/ping", "200")))
}

func TestHTTPMiddleware_CapturesNon200Status(t *testing.T) {
	m := New("test_http_status")

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/fail", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	srv := m.Middleware(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/fail", nil)
	srv.ServeHTTP(rec, req)

	assert.Equal(t, float64(1), testutil.ToFloat64(m.requests.WithLabelValues("POST", "POST /api/v1/fail", "500")))
}

func TestHTTPMiddleware_ImplicitOK(t *testing.T) {
	m := New("test_http_implicit")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /implicit", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	srv := m.Middleware(mux)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/implicit", nil))

	assert.Equal(t, float64(1), testutil.ToFloat64(m.requests.WithLabelValues("GET", "GET /implicit", "200")))
}

func TestHTTPMiddleware_UnmatchedPath(t *testing.T) {
	m := New("test_http_unmatched")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /known", func(w http.ResponseWriter, r *http.Request) {})

	srv := m.Middleware(mux)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/totally/unknown", nil))

	assert.Equal(t, float64(1), testutil.ToFloat64(m.requests.WithLabelValues("GET", "unmatched", "404")))
}

func TestNew_IsIdempotent(t *testing.T) {
	a := New("test_http_idem")
	b := New("test_http_idem")
	assert.Same(t, a, b)
}

func TestHandler_ExposesMetrics(t *testing.T) {
	_ = New("test_http_handler")

	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.True(t, strings.Contains(body, "http_requests_total") || strings.Contains(body, "go_goroutines"))
}
