package middleware

import (
	"net/http"
	"strings"
)

// CORSMiddleware добавляет CORS-заголовки и обрабатывает preflight-запросы.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Список точных совпадений для продакшена
		allowedOrigins := map[string]bool{
			"http://clover-go.ru":      true,
			"http://clover-go.ru:80":   true,
			"http://clover-go.ru:8080": true,
			"http://localhost:8080":    true,
		}

		// Разрешаем известные домены ИЛИ любой локалхост (для удобства разработки)
		if allowedOrigins[origin] || origin == "http://localhost" || strings.HasPrefix(origin, "http://localhost:") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Origin, Accept, X-CSRF-Token, X-Device-ID")
		w.Header().Set("Access-Control-Expose-Headers", "X-CSRF-Token")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
