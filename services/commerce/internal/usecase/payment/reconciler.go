package payment

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

// PendingLister — отдаёт зависшие pending-платежи для сверки.
type PendingLister interface {
	ListStalePending(ctx context.Context, provider string, olderThan time.Time, limit int) ([]models.Payment, error)
}

// Applier применяет терминальный статус (см. wallet.Usecase.ApplyProviderUpdate).
type Applier interface {
	ApplyProviderUpdate(ctx context.Context, provider, providerRef, status string, rawPayload []byte) error
}

// Reconciler периодически опрашивает провайдера по pending-платежам и применяет
// прилетевшие за этот промежуток терминальные статусы. Подстраховка от потерянных webhook'ов.
type Reconciler struct {
	log      *slog.Logger
	payments PendingLister
	provider Provider
	applier  Applier
	interval time.Duration
	staleAge time.Duration
	provName string
	batchN   int
}

// ReconcilerConfig — параметры reconciler'а.
type ReconcilerConfig struct {
	Interval     time.Duration // частота прохода, например 1 мин
	StaleAge     time.Duration // платежи старше этого считаем зависшими, например 5 мин
	BatchSize    int           // сколько платежей за проход
	ProviderName string        // models.PaymentProviderYooKassa
}

// NewReconciler создаёт reconciler.
func NewReconciler(
	log *slog.Logger,
	payments PendingLister,
	provider Provider,
	applier Applier,
	cfg ReconcilerConfig,
) *Reconciler {
	if cfg.Interval <= 0 {
		cfg.Interval = time.Minute
	}
	if cfg.StaleAge <= 0 {
		cfg.StaleAge = 5 * time.Minute
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	return &Reconciler{
		log:      log,
		payments: payments,
		provider: provider,
		applier:  applier,
		interval: cfg.Interval,
		staleAge: cfg.StaleAge,
		provName: cfg.ProviderName,
		batchN:   cfg.BatchSize,
	}
}

// Run крутит reconciler до отмены контекста.
func (r *Reconciler) Run(ctx context.Context) {
	t := time.NewTicker(r.interval)
	defer t.Stop()

	r.log.Info("payment reconciler started",
		slog.String("provider", r.provName),
		slog.Duration("interval", r.interval),
		slog.Duration("stale_age", r.staleAge),
	)

	for {
		select {
		case <-ctx.Done():
			r.log.Info("payment reconciler stopped")
			return
		case <-t.C:
			r.tick(ctx)
		}
	}
}

func (r *Reconciler) tick(ctx context.Context) {
	stale, err := r.payments.ListStalePending(ctx, r.provName, time.Now().Add(-r.staleAge), r.batchN)
	if err != nil {
		r.log.ErrorContext(ctx, "reconciler list failed", slog.String("error", err.Error()))
		return
	}
	if len(stale) == 0 {
		return
	}

	r.log.DebugContext(ctx, "reconciler batch", slog.Int("count", len(stale)))
	for _, p := range stale {
		if p.ProviderRef == nil {
			continue
		}
		res, err := r.provider.GetPayment(ctx, *p.ProviderRef)
		if err != nil {
			r.log.WarnContext(ctx, "reconciler get payment failed",
				slog.Int64("payment_id", p.ID),
				slog.String("error", err.Error()),
			)
			continue
		}
		if res.Status == models.PaymentStatusPending {
			continue
		}
		if err := r.applier.ApplyProviderUpdate(ctx, p.Provider, *p.ProviderRef, res.Status, res.RawPayload); err != nil {
			r.log.ErrorContext(ctx, "reconciler apply failed",
				slog.Int64("payment_id", p.ID),
				slog.String("error", err.Error()),
			)
		}
	}
}
