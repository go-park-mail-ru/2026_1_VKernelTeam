package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	api "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/api"
)

// CSRF защищает от атак, проверяя наличие токена в заголовке и куках
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// пропускаем проверку для безопасных запросов и для ручек авторизации, а также logout/refresh
		if r.Method == http.MethodGet ||
			r.Method == http.MethodOptions ||
			r.Method == http.MethodHead ||
			path == api.ApiPrefix+"/auth/login" ||
			path == api.ApiPrefix+"/auth/register" ||
			path == api.ApiPrefix+"/auth/refresh" ||
			path == api.ApiPrefix+"/auth/logout" {
			next.ServeHTTP(w, r)
			return
		}

		// достаём куки из куки
		cookie, err := r.Cookie("csrf_token")
		if err != nil {
			http.Error(w, "Missing CSRF cookie", http.StatusBadRequest)
			return
		}

		// достаём токен из заголовка
		headerToken := r.Header.Get("X-CSRF-Token")

		// сравниваем
		if headerToken == "" || headerToken != cookie.Value {
			http.Error(w, "CSRF token mismatch", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GenerateCSRFToken генерирует новый токен для отправки клиенту
func GenerateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
