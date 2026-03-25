package middleware

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSRFMiddleware(t *testing.T) {
	// хендлер, который можно обернуть в middleware
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CSRFMiddleware(nextHandler)

	tests := []struct {
		name           string
		method         string
		path           string
		cookieToken    string
		headerToken    string
		expectedStatus int
	}{
		{
			name:           "Allow GET without token",
			method:         http.MethodGet,
			path:           "/api/v1/ads",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Allow Login without token",
			method:         http.MethodPost,
			path:           "/api/v1/auth/login",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Deny POST without cookie",
			method:         http.MethodPost,
			path:           "/api/v1/auth/logout",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Deny POST with mismatching tokens",
			method:         http.MethodPost,
			path:           "/api/v1/auth/logout",
			cookieToken:    "token1",
			headerToken:    "token2",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Allow POST with matching tokens",
			method:         http.MethodPost,
			path:           "/api/v1/auth/logout",
			cookieToken:    "matching_token",
			headerToken:    "matching_token",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)

			// если в тесте есть кука - добавляем
			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{
					Name:  "csrf_token",
					Value: tt.cookieToken,
				})
			}

			// аналогично заголвок
			if tt.headerToken != "" {
				req.Header.Add("X-CSRF-Token", tt.headerToken)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	token1 := GenerateCSRFToken()
	token2 := GenerateCSRFToken()

	assert.NotEmpty(t, token1)
	assert.NotEmpty(t, token2)
	assert.NotEqual(t, token1, token2, "Tokens should be unique")

	_, err := base64.StdEncoding.DecodeString(token1)
	assert.NoError(t, err)
}
