// Package metrics — бизнес-метрики commerce-сервиса.
// HTTP-метрики живут в pkg/shared/metrics, здесь только domain-specific.
package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	labelService = "service"
	serviceLabel = "commerce"
)

// Business — набор бизнес-метрик commerce.
type Business struct {
	PromotionPurchases      *prometheus.CounterVec
	PromotionPurchaseErrors *prometheus.CounterVec
	WalletTopups            prometheus.Counter
	WalletTopupsAmount      prometheus.Counter
}

var (
	once     sync.Once
	instance *Business
)

// Get возвращает синглтон. Безопасно для повторного вызова в тестах.
func Get() *Business {
	once.Do(func() {
		instance = &Business{
			PromotionPurchases: promauto.NewCounterVec(prometheus.CounterOpts{
				Name:        "commerce_promotion_purchases_total",
				Help:        "Количество успешных покупок продвижения.",
				ConstLabels: prometheus.Labels{labelService: serviceLabel},
			}, []string{"kind", "plan_code"}),
			PromotionPurchaseErrors: promauto.NewCounterVec(prometheus.CounterOpts{
				Name:        "commerce_promotion_purchase_errors_total",
				Help:        "Количество ошибок при покупке продвижения по типам.",
				ConstLabels: prometheus.Labels{labelService: serviceLabel},
			}, []string{"error_code"}),
			WalletTopups: promauto.NewCounter(prometheus.CounterOpts{
				Name:        "commerce_wallet_topups_total",
				Help:        "Количество успешных пополнений кошелька.",
				ConstLabels: prometheus.Labels{labelService: serviceLabel},
			}),
			WalletTopupsAmount: promauto.NewCounter(prometheus.CounterOpts{
				Name:        "commerce_wallet_topups_amount_rubles_total",
				Help:        "Суммарный объём пополнений кошелька в рублях.",
				ConstLabels: prometheus.Labels{labelService: serviceLabel},
			}),
		}
	})
	return instance
}
