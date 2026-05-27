package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ipMWLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestYooKassaIPWhitelist(t *testing.T) {
	const internalRemote = "10.0.0.1:1234"

	tests := []struct {
		name     string
		xff      string
		remote   string
		wantCode int
	}{
		{name: "allowed XFF (185.71.76.x)", xff: "185.71.76.5", remote: internalRemote, wantCode: 200},
		{name: "allowed XFF (185.71.77.x)", xff: "185.71.77.30", remote: internalRemote, wantCode: 200},
		{name: "allowed XFF (77.75.156.11)", xff: "77.75.156.11", remote: internalRemote, wantCode: 200},
		{name: "denied XFF (1.2.3.4)", xff: "1.2.3.4", remote: internalRemote, wantCode: http.StatusForbidden},
		{name: "denied remote without XFF", xff: "", remote: "1.2.3.4:1234", wantCode: http.StatusForbidden},
		{name: "allowed remote without XFF", xff: "", remote: "185.71.76.10:443", wantCode: 200},
	}

	mw := YooKassaIPWhitelist(ipMWLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/x", nil)
			req.RemoteAddr = tt.remote
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}
