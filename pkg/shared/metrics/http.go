// Package metrics предоставляет единый набор Prometheus-метрик для всех сервисов
// проекта Clover: HTTP middleware, gRPC unary-interceptor и /metrics handler.
//
// Использование:
//
//	m := metrics.New("auth")
//	mux.Handle("GET /metrics", metrics.Handler())
//	handler := m.Middleware(mux)
//
// Лейблы (service, method, path, status) выбраны так, чтобы избежать
// cardinality explosion: path берётся из http.Request.Pattern (Go 1.22+
// ServeMux), а не из r.URL.Path.
package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// labelService — имя ConstLabel'а, в который попадает имя сервиса.
const labelService = "service"

// HTTPMetrics — набор HTTP-метрик одного сервиса.
type HTTPMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// registry хранит уже созданные HTTPMetrics по service-имени, чтобы при
// повторном вызове New (например, из тестов) не падать с panic'ом
// "duplicate metrics collector registration".
var (
	httpRegistry   = make(map[string]*HTTPMetrics)
	httpRegistryMu sync.Mutex
)

// New создаёт (или возвращает уже существующий) набор HTTP-метрик для service.
// service попадает в ConstLabel — благодаря этому Prometheus агрегирует по
// сервисам без необходимости job-relabel'а.
func New(service string) *HTTPMetrics {
	httpRegistryMu.Lock()
	defer httpRegistryMu.Unlock()
	if m, ok := httpRegistry[service]; ok {
		return m
	}
	m := &HTTPMetrics{
		requests: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "http_requests_total",
			Help:        "Количество обработанных HTTP-запросов.",
			ConstLabels: prometheus.Labels{labelService: service},
		}, []string{"method", "path", "status"}),
		duration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "Длительность обработки HTTP-запросов в секундах.",
			Buckets:     prometheus.DefBuckets,
			ConstLabels: prometheus.Labels{labelService: service},
		}, []string{"method", "path"}),
	}
	httpRegistry[service] = m
	return m
}

// statusRecorder перехватывает status code (и WriteHeader, и неявный 200).
type statusRecorder struct {
	http.ResponseWriter
	code        int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.code = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.code = http.StatusOK
		s.wroteHeader = true
	}
	return s.ResponseWriter.Write(b)
}

// Middleware пишет http_requests_total и http_request_duration_seconds на
// каждый запрос. Использовать в цепочке middleware ПОСЛЕ маршрутизатора,
// чтобы r.Pattern был уже заполнен.
//
// Цепочка должна быть: CORS -> RequestID -> Metrics -> AccessLog -> ... -> mux
// (сам Metrics ставится снаружи mux'а, но r.Pattern заполняется при матчинге
// внутри mux'а — поэтому label'ы читаются в defer'е после ServeHTTP).
func (m *HTTPMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rec, r)

		path := r.Pattern
		if path == "" {
			path = "unmatched"
		}

		m.requests.WithLabelValues(r.Method, path, strconv.Itoa(rec.code)).Inc()
		m.duration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
	})
}

// Handler возвращает http.Handler для /metrics endpoint.
func Handler() http.Handler { return promhttp.Handler() }
