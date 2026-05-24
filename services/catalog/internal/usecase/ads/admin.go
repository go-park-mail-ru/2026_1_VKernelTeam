package ads

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/models"
)

// ErrAdminDepsNotConfigured возвращается, когда админ-функционал вызван без
// сконфигурированных зависимостей (см. Ads.WithAdmin).
var ErrAdminDepsNotConfigured = errors.New("admin dependencies are not configured")

// AdminDeleteAd выполняет жёсткое (для всех, кроме админа) удаление объявления
// и отправляет продавцу системное сообщение.
func (a *Ads) AdminDeleteAd(ctx context.Context, adID, adminID int64) error {
	a.log.InfoContext(ctx, "admin deleting ad",
		slog.String("op", opAdminDeleteAd),
		slog.Int64("ad_id", adID),
		slog.Int64("admin_id", adminID),
	)

	sellerID, title, err := a.adsStorage.AdminDeleteAd(ctx, adID, adminID)
	if err != nil {
		return err
	}

	a.notifySeller(ctx, sellerID, adID,
		fmt.Sprintf("Ваш товар «%s» удалён по решению администратора.", title),
		opAdminDeleteAd,
	)
	return nil
}

// ApproveAd одобряет объявление, ожидающее модерации.
func (a *Ads) ApproveAd(ctx context.Context, adID, adminID int64) error {
	a.log.InfoContext(ctx, "admin approving ad",
		slog.String("op", opApproveAd),
		slog.Int64("ad_id", adID),
		slog.Int64("admin_id", adminID),
	)

	sellerID, title, err := a.adsStorage.ModerateAd(ctx, adID, models.AdStatusActive, "")
	if err != nil {
		return err
	}

	a.notifySeller(ctx, sellerID, adID,
		fmt.Sprintf("Ваше объявление «%s» одобрено и опубликовано.", title),
		opApproveAd,
	)
	return nil
}

// RejectAd отклоняет объявление, ожидающее модерации, с указанием причины.
func (a *Ads) RejectAd(ctx context.Context, adID, adminID int64, reason string) error {
	a.log.InfoContext(ctx, "admin rejecting ad",
		slog.String("op", opRejectAd),
		slog.Int64("ad_id", adID),
		slog.Int64("admin_id", adminID),
	)

	sellerID, title, err := a.adsStorage.ModerateAd(ctx, adID, models.AdStatusRejected, reason)
	if err != nil {
		return err
	}

	text := fmt.Sprintf("Ваше объявление «%s» отклонено модератором.", title)
	if reason != "" {
		text += " Причина: " + reason
	}
	a.notifySeller(ctx, sellerID, adID, text, opRejectAd)
	return nil
}

// GetModerationQueue возвращает объявления, ожидающие модерации.
func (a *Ads) GetModerationQueue(ctx context.Context) ([]models.Ad, error) {
	return a.adsStorage.GetModerationQueue(ctx)
}

// GetUserAdsByStatus возвращает объявления продавца с указанным статусом
// (например, для вкладки «На модерации»).
func (a *Ads) GetUserAdsByStatus(ctx context.Context, userID int64, status string) ([]models.Ad, error) {
	return a.adsStorage.GetUserAdsByStatus(ctx, userID, status)
}

// IsModerationEnabled возвращает текущее состояние флага модерации.
func (a *Ads) IsModerationEnabled(ctx context.Context) bool {
	if a.moderationGate == nil {
		return false
	}

	return a.moderationGate.IsEnabled(ctx)
}

// SetModerationEnabled переключает глобальный флаг модерации.
func (a *Ads) SetModerationEnabled(ctx context.Context, enabled bool, adminID int64) error {
	if a.moderationGate == nil {
		return ErrAdminDepsNotConfigured
	}

	a.log.InfoContext(ctx, "switching moderation",
		slog.String("op", opSetModeration),
		slog.Bool("enabled", enabled),
		slog.Int64("admin_id", adminID),
	)
	return a.moderationGate.SetEnabled(ctx, enabled, adminID)
}

// notifySeller отправляет системное сообщение продавцу.
func (a *Ads) notifySeller(ctx context.Context, sellerID, adID int64, text, op string) {
	if a.sysMessenger == nil || a.systemUserID == 0 {
		return
	}

	if err := a.sysMessenger.Send(ctx, a.systemUserID, sellerID, adID, text); err != nil {
		a.log.WarnContext(ctx, "failed to send system message",
			slog.String("op", op),
			slog.Int64("ad_id", adID),
			slog.Int64("seller_id", sellerID),
			slog.String("error", err.Error()),
		)
	}
}
