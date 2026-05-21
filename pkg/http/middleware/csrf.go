package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	api "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/api"
)

// isViewRecordPath проверяет, что путь соответствует POST /ads/{id}/view.
// Эта ручка вызывается анонимными пользователями (у которых нет csrf_token cookie),
// поэтому исключаем её из CSRF-проверки. Дедупликация выполняется по X-Device-ID.
func isViewRecordPath(path string) bool {
	const prefix = api.ApiPrefix + "/ads/"
	const suffix = "/view"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if id == "" || strings.ContainsRune(id, '/') {
		return false
	}
	return true
}

// CSRFMiddleware защищает от CSRF-атак, сверяя токен в заголовке и cookie.
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
			path == api.ApiPrefix+"/auth/logout" ||
			isViewRecordPath(path) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("csrf_token")
		if err != nil {
			http.Error(w, "Missing CSRF cookie", http.StatusBadRequest)
			return
		}

		headerToken := r.Header.Get("X-CSRF-Token")

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
