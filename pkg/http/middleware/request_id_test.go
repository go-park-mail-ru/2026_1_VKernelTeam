package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestIDMiddleware_GeneratesIDWhenAbsent(t *testing.T) {
	var ctxValue any
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxValue = r.Context().Value(RequestIDKey)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	id, ok := ctxValue.(string)
	assert.True(t, ok)
	assert.NotEmpty(t, id)
	assert.Equal(t, id, rec.Header().Get("X-Request-ID"))
}

func TestRequestIDMiddleware_PreservesIncomingID(t *testing.T) {
	var ctxValue any
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxValue = r.Context().Value(RequestIDKey)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "preset-id-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "preset-id-123", ctxValue)
	assert.Equal(t, "preset-id-123", rec.Header().Get("X-Request-ID"))
}
