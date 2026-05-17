package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	// Dummy handler that simply returns 200 OK
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	handler := CORSMiddleware(dummyHandler)

	tests := []struct {
		name           string
		method         string
		origin         string
		expectedStatus int
		expectCORS     bool
	}{
		{
			name:           "Allowed Origin OPTIONS",
			method:         http.MethodOptions,
			origin:         "http://clover-go.ru:80",
			expectedStatus: http.StatusOK,
			expectCORS:     true,
		},
		{
			name:           "Allowed Origin GET",
			method:         http.MethodGet,
			origin:         "http://clover-go.ru",
			expectedStatus: http.StatusOK,
			expectCORS:     true,
		},
		{
			name:           "Disallowed Origin GET",
			method:         http.MethodGet,
			origin:         "http://evil.com",
			expectedStatus: http.StatusOK,
			expectCORS:     false, // Origin header will not be accepted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), tt.method, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			originHeader := rr.Header().Get("Access-Control-Allow-Origin")
			if tt.expectCORS && originHeader != tt.origin {
				t.Errorf("expected Access-Control-Allow-Origin %q, got %q", tt.origin, originHeader)
			} else if !tt.expectCORS && originHeader != "" {
				t.Errorf("did not expect Access-Control-Allow-Origin, got %q", originHeader)
			}

			// Check other mandatory headers
			if rr.Header().Get("Access-Control-Allow-Credentials") != "true" {
				t.Errorf("missing or incorrect Access-Control-Allow-Credentials header")
			}
		})
	}
}
