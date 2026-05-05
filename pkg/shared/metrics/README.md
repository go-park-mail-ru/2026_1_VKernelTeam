# pkg/shared/metrics

Единый набор Prometheus-метрик для всех сервисов Clover.

Метрики:

| Метрика | Тип | Лейблы |
|---------|-----|--------|
| `http_requests_total` | counter | `service`, `method`, `path`, `status` |
| `http_request_duration_seconds` | histogram | `service`, `method`, `path` |
| `grpc_server_handled_total` | counter | `service`, `grpc_method`, `grpc_code` |
| `grpc_server_handling_seconds` | histogram | `service`, `grpc_method` |

`path` берётся из `http.Request.Pattern` (Go 1.22+ ServeMux) — это template
вида `GET /api/v1/users/{id}`, без подстановки конкретных id. Защита от
cardinality explosion в Prometheus.

## HTTP

```go
import "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/metrics"

func buildHTTPServer(...) *http.Server {
    mux := http.NewServeMux()
    // ... ваши роуты ...

    // Prometheus scrape endpoint (CSRF пропустит GET, AccessLog skip'нет /metrics).
    mux.Handle("GET /metrics", metrics.Handler())

    httpMetrics := metrics.New("auth") // service-name = job_name в prometheus.yml

    handler := middleware.CSRFMiddleware(mux)
    handler = middleware.AccessLogMiddleware(log)(handler)
    handler = httpMetrics.Middleware(handler) // ставим до AccessLog, но после RequestID
    handler = middleware.RequestIDMiddleware(handler)
    handler = middleware.CORSMiddleware(handler)
    // ...
}
```

## gRPC

```go
func buildGRPCServer(...) *grpclib.Server {
    grpcMetrics := metrics.NewGRPC("auth")
    srv := grpclib.NewServer(grpclib.UnaryInterceptor(grpcMetrics.UnaryServerInterceptor()))
    // ... RegisterXxxServer ...
    return srv
}
```

Если нужно несколько интерсепторов — оборачивайте через `grpclib.ChainUnaryInterceptor(...)`.

## Идемпотентность

`metrics.New(name)` и `metrics.NewGRPC(name)` можно звать многократно с одним
и тем же `name` — вернётся уже зарегистрированный набор. Это удобно в тестах
и при множественных вызовах из подсистем сервиса.

## prometheus.yml

Каждый сервис должен быть в `deployments/prometheus/prometheus.yml`:

```yaml
- job_name: "auth"
  static_configs:
    - targets: ["auth:8001"]
  metrics_path: /metrics
```
