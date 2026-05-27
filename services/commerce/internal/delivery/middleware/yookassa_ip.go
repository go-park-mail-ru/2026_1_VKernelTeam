package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

// yookassaIPRanges — официальные CIDR'ы, с которых ЮКасса отправляет webhook'и.
// Источник: https://yookassa.ru/developers/using-api/webhooks#ip
var yookassaIPRanges = mustParseCIDRs([]string{
	"185.71.76.0/27",
	"185.71.77.0/27",
	"77.75.153.0/25",
	"77.75.156.11/32",
	"77.75.156.35/32",
	"77.75.154.128/25",
	"2a02:5180::/32",
})

// YooKassaIPWhitelist пропускает только запросы с IP, принадлежащих ЮКассе.
// Реальный IP берётся из X-Forwarded-For (первый элемент) или RemoteAddr.
func YooKassaIPWhitelist(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			if ip == nil || !ipAllowed(ip) {
				log.WarnContext(r.Context(), "yookassa webhook rejected: ip not in whitelist",
					slog.String("ip", ipString(ip)),
					slog.String("xff", r.Header.Get("X-Forwarded-For")),
				)
				responser.RespondWithError(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) net.IP {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Берём первый IP — клиентский, остальные добавлены прокси.
		parts := strings.Split(xff, ",")
		if ip := net.ParseIP(strings.TrimSpace(parts[0])); ip != nil {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return net.ParseIP(host)
}

func ipAllowed(ip net.IP) bool {
	for _, n := range yookassaIPRanges {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func ipString(ip net.IP) string {
	if ip == nil {
		return ""
	}
	return ip.String()
}

func mustParseCIDRs(cidrs []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic("yookassa_ip: invalid CIDR " + c + ": " + err.Error())
		}
		out = append(out, n)
	}
	return out
}
