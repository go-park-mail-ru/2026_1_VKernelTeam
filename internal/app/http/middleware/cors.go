package middleware

import (
	"net/http"
)

// CORSMiddleware добавляет заголовки CORS к ответам и обрабатывает preflight запросы.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Разрешаем только определенные домены. Порт 80 считается эквивалентным домену без порта,
		// поэтому обрабатываем его отдельно.
		if origin == "http://clover-go.ru" || origin == "http://clover-go.ru:80" ||
			origin == "http://clover-go.ru:8080" || origin == "http://localhost:8080" ||
			origin == "http://localhost:80" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
		}

		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Origin, Accept")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 часа

		// Если это preflight запрос, то просто возвращаем статус 200 OK.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
