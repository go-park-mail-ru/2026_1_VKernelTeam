package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// RequestIDKey — ключ контекста для хранения request-id.
const RequestIDKey contextKey = "requestID"

// RequestIDMiddleware генерирует уникальный request-id для каждого запроса,
// сохраняет его в контексте и устанавливает заголовок X-Request-ID в ответе.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
