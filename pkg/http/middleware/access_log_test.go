package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccessLogMiddleware_LogsAtAllSeverities(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	cases := []struct {
		name   string
		status int
	}{
		{"info-2xx", http.StatusOK},
		{"warn-4xx", http.StatusBadRequest},
		{"error-5xx", http.StatusInternalServerError},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			handler := AccessLogMiddleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.status)
				_, _ = w.Write([]byte("ok"))
			}))

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, c.status, rec.Code)
		})
	}
}

func TestAccessLogMiddleware_DefaultStatusIsOK(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := AccessLogMiddleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Не вызываем WriteHeader — статус должен остаться 200 по умолчанию.
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
