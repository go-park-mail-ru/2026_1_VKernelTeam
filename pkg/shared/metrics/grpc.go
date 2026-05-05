package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// GRPCMetrics — набор gRPC-метрик одного сервиса.
type GRPCMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

var (
	grpcRegistry   = make(map[string]*GRPCMetrics)
	grpcRegistryMu sync.Mutex
)

// NewGRPC создаёт (или возвращает уже существующий) набор gRPC-метрик для service.
func NewGRPC(service string) *GRPCMetrics {
	grpcRegistryMu.Lock()
	defer grpcRegistryMu.Unlock()
	if m, ok := grpcRegistry[service]; ok {
		return m
	}
	m := &GRPCMetrics{
		requests: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "grpc_server_handled_total",
			Help:        "Количество обработанных gRPC-запросов.",
			ConstLabels: prometheus.Labels{"service": service},
		}, []string{"grpc_method", "grpc_code"}),
		duration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "grpc_server_handling_seconds",
			Help:        "Длительность обработки gRPC-запросов в секундах.",
			Buckets:     prometheus.DefBuckets,
			ConstLabels: prometheus.Labels{"service": service},
		}, []string{"grpc_method"}),
	}
	grpcRegistry[service] = m
	return m
}

// UnaryServerInterceptor возвращает grpc.UnaryServerInterceptor, который
// инкрементит grpc_server_handled_total и пишет latency в гистограмму.
func (m *GRPCMetrics) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		code := status.Code(err).String()
		m.requests.WithLabelValues(info.FullMethod, code).Inc()
		m.duration.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())

		return resp, err
	}
}
