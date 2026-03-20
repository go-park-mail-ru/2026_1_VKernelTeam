package logger

import (
	"log/slog"
	"testing"
)

func TestSetupLogger_ValidLogger(t *testing.T) {
	envs := []string{envLocal, envDev, envProd, "unknown", ""}

	for _, env := range envs {
		t.Run(env, func(t *testing.T) {
			logger := SetupLogger(env)
			if logger == nil {
				t.Errorf("SetupLogger(%q) returned nil, expected a valid logger", env)
			}
		})
	}
}

func TestSetupLogger_HandlerTypes(t *testing.T) {
	tests := []struct {
		env    string
		isJSON bool
		isText bool
	}{
		{env: envLocal, isJSON: false, isText: true},
		{env: envDev, isJSON: true, isText: false},
		{env: envProd, isJSON: true, isText: false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			logger := SetupLogger(tt.env)
			handler := logger.Handler()

			if tt.isJSON {
				if _, ok := handler.(*slog.JSONHandler); !ok {
					t.Errorf("SetupLogger(%q) expected JSONHandler, got %T", tt.env, handler)
				}
			}

			if tt.isText {
				if _, ok := handler.(*slog.TextHandler); !ok {
					t.Errorf("SetupLogger(%q) expected TextHandler, got %T", tt.env, handler)
				}
			}
		})
	}
}
