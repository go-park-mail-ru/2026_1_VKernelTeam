// Package promotion — usecase покупки и просмотра промо.
package promotion

//go:generate mockgen -source=promotion.go -destination=mocks/mock_promotion.go -package=mocks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	commercemetrics "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/metrics"
	commercepostgres "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/postgres"
	promorepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/promotion"
	walletrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/wallet"
)

const (
	opGetPlans     = "usecase.promotion.GetPlans"
	opPurchase     = "usecase.promotion.Purchase"
	opListByAd     = "usecase.promotion.ListByAd"
	opListByUser   = "usecase.promotion.ListByUser"
	defaultListLim = 20
	maxListLim     = 100
)

// Бизнес-ошибки (текст совпадает с кодом, отдаваемым клиенту).
var (
	ErrPlanNotFound     = errors.New("PLAN_NOT_FOUND")
	ErrPlanInactive     = errors.New("PLAN_INACTIVE")
	ErrAdNotFound       = errors.New("AD_NOT_FOUND")
	ErrNotAdOwner       = errors.New("NOT_AD_OWNER")
	ErrInvalidAdStatus  = errors.New("INVALID_AD_STATUS")
	ErrInsufficientFund = errors.New("INSUFFICIENT_FUNDS")
	ErrInvalidRequest   = errors.New("INVALID_REQUEST")
)

// TxRunner — узкий контракт пула.
type TxRunner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PlanRepo — каталог тарифов с кэшем и БД.
type PlanRepo interface {
	GetActivePlans(ctx context.Context) ([]models.PromotionPlan, error)
	GetByCode(ctx context.Context, code string) (models.PromotionPlan, error)
	GetByID(ctx context.Context, id int64) (models.PromotionPlan, error)
}

// PromotionRepo — купленные промо.
type PromotionRepo interface {
	InsertTx(
		ctx context.Context, tx pgx.Tx,
		productID, userID, planID int64, kind string,
		expiresAt time.Time, pricePaid int64,
	) (models.Promotion, error)
	MaxActiveExpiresTx(ctx context.Context, tx pgx.Tx, productID int64, kind string) (time.Time, bool, error)
	GetActiveByAd(ctx context.Context, productID int64) ([]models.Promotion, error)
	ListByUser(ctx context.Context, userID int64, cursor int64, limit int) ([]models.Promotion, error)
	GetByID(ctx context.Context, id int64) (models.Promotion, error)
}

// WalletRepo — нужны tx-методы, idempotency-чек и обычное чтение баланса.
type WalletRepo interface {
	Get(ctx context.Context, userID int64) (models.Wallet, error)
	GetForUpdateTx(ctx context.Context, tx pgx.Tx, userID int64) (models.Wallet, error)
	DecrementBalanceTx(ctx context.Context, tx pgx.Tx, userID, amount int64) (int64, error)
	InsertTransactionTx(
		ctx context.Context, tx pgx.Tx,
		userID, amount int64, txType string, referenceID *int64, idempotencyKey *string,
	) (models.WalletTransaction, error)
	GetTransactionByIdempotencyKey(ctx context.Context, key string) (models.WalletTransaction, error)
}

// PlanCache — Redis-кэш тарифов (опционален).
type PlanCache interface {
	Get(ctx context.Context) ([]models.PromotionPlan, error)
	Set(ctx context.Context, plans []models.PromotionPlan) error
}

// AdsProvider — внешний catalog-клиент для проверки владельца и статуса объявления.
type AdsProvider interface {
	GetAdByID(ctx context.Context, id int64) (models.Ad, error)
}

// Usecase — основной сервис продвижения.
type Usecase struct {
	log        *slog.Logger
	tx         TxRunner
	plans      PlanRepo
	promotions PromotionRepo
	wallets    WalletRepo
	cache      PlanCache
	ads        AdsProvider
}

// New создаёт usecase продвижения с зависимостями репозиториев, кэша и каталога.
func New(
	log *slog.Logger,
	tx TxRunner,
	plans PlanRepo,
	promotions PromotionRepo,
	wallets WalletRepo,
	cache PlanCache,
	ads AdsProvider,
) *Usecase {
	return &Usecase{
		log:        log,
		tx:         tx,
		plans:      plans,
		promotions: promotions,
		wallets:    wallets,
		cache:      cache,
		ads:        ads,
	}
}

// GetPlans возвращает активные тарифы. Кэшируется в Redis на 5 минут.
func (u *Usecase) GetPlans(ctx context.Context) ([]dto.PromotionPlanResponse, error) {
	u.log.DebugContext(ctx, "getting plans", slog.String("op", opGetPlans))

	if u.cache != nil {
		if cached, err := u.cache.Get(ctx); err == nil {
			return planModelsToDTO(cached), nil
		}
	}

	plans, err := u.plans.GetActivePlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetPlans: %w", err)
	}
	if u.cache != nil {
		if err := u.cache.Set(ctx, plans); err != nil {
			u.log.WarnContext(ctx, "failed to set plan cache",
				slog.String("error", err.Error()),
			)
		}
	}
	return planModelsToDTO(plans), nil
}

func planModelsToDTO(plans []models.PromotionPlan) []dto.PromotionPlanResponse {
	out := make([]dto.PromotionPlanResponse, 0, len(plans))
	for _, p := range plans {
		out = append(out, dto.PromotionPlanResponse{
			ID:           p.ID,
			Code:         p.Code,
			Kind:         p.Kind,
			DurationDays: p.DurationDays,
			Price:        p.Price,
		})
	}
	return out
}

// Purchase — главный сценарий покупки промо.
func (u *Usecase) Purchase(
	ctx context.Context, userID, productID int64, planCode, idempotencyKey string,
) (dto.PurchasePromotionResponse, error) {
	if planCode == "" || idempotencyKey == "" {
		return dto.PurchasePromotionResponse{}, ErrInvalidRequest
	}

	u.log.InfoContext(ctx, "purchase requested",
		slog.String("op", opPurchase),
		slog.Int64("user_id", userID),
		slog.Int64("product_id", productID),
		slog.String("plan_code", planCode),
	)

	// Idempotent fast-path: если ключ уже использован — возвращаем существующее промо.
	if existing, err := u.wallets.GetTransactionByIdempotencyKey(ctx, idempotencyKey); err == nil {
		return u.buildIdempotentResponse(ctx, userID, existing)
	}

	// 1. Тариф.
	plan, err := u.plans.GetByCode(ctx, planCode)
	if err != nil {
		switch {
		case errors.Is(err, promorepo.ErrPlanNotFound):
			return dto.PurchasePromotionResponse{}, ErrPlanNotFound
		case errors.Is(err, promorepo.ErrPlanInactive):
			return dto.PurchasePromotionResponse{}, ErrPlanInactive
		default:
			return dto.PurchasePromotionResponse{}, fmt.Errorf("Purchase: get plan: %w", err)
		}
	}

	// 2. Объявление (через gRPC к catalog).
	ad, err := u.ads.GetAdByID(ctx, productID)
	if err != nil {
		return dto.PurchasePromotionResponse{}, ErrAdNotFound
	}
	if ad.SellerID != userID {
		return dto.PurchasePromotionResponse{}, ErrNotAdOwner
	}
	if ad.Status != models.AdStatusActive && ad.Status != models.AdStatusReserved {
		return dto.PurchasePromotionResponse{}, ErrInvalidAdStatus
	}

	// 3. Транзакция: блокируем кошелёк, считаем expires, создаём промо и списываем.
	tx, err := u.tx.Begin(ctx)
	if err != nil {
		return dto.PurchasePromotionResponse{}, fmt.Errorf("Purchase: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	wallet, err := u.wallets.GetForUpdateTx(ctx, tx, userID)
	if err != nil {
		return dto.PurchasePromotionResponse{}, fmt.Errorf("Purchase: lock wallet: %w", err)
	}
	if wallet.Balance < plan.Price {
		return dto.PurchasePromotionResponse{}, ErrInsufficientFund
	}

	// Если уже есть активный boost/highlight — продлеваем от существующего expires.
	maxExpires, has, err := u.promotions.MaxActiveExpiresTx(ctx, tx, productID, plan.Kind)
	if err != nil {
		return dto.PurchasePromotionResponse{}, fmt.Errorf("Purchase: max expires: %w", err)
	}
	startFrom := time.Now()
	if has && maxExpires.After(startFrom) {
		startFrom = maxExpires
	}
	expiresAt := startFrom.Add(time.Duration(plan.DurationDays) * 24 * time.Hour)

	promo, err := u.promotions.InsertTx(
		ctx, tx, productID, userID, plan.ID, plan.Kind, expiresAt, plan.Price,
	)
	if err != nil {
		return dto.PurchasePromotionResponse{}, fmt.Errorf("Purchase: insert promotion: %w", err)
	}

	promoID := promo.ID
	key := idempotencyKey
	_, err = u.wallets.InsertTransactionTx(
		ctx, tx, userID, -plan.Price, models.WalletTxTypePromotionCharge, &promoID, &key,
	)
	if err != nil {
		if commercepostgres.IsPgUniqueViolation(err) {
			// Параллельный запрос с тем же ключом проскочил вперёд — возвращаем его результат.
			existing, getErr := u.wallets.GetTransactionByIdempotencyKey(ctx, idempotencyKey)
			if getErr == nil {
				return u.buildIdempotentResponse(ctx, userID, existing)
			}
		}
		return dto.PurchasePromotionResponse{}, fmt.Errorf("Purchase: insert wallet tx: %w", err)
	}

	newBalance, err := u.wallets.DecrementBalanceTx(ctx, tx, userID, plan.Price)
	if err != nil {
		return dto.PurchasePromotionResponse{}, fmt.Errorf("Purchase: decrement: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.PurchasePromotionResponse{}, fmt.Errorf("Purchase: commit: %w", err)
	}

	commercemetrics.Get().PromotionPurchases.WithLabelValues(plan.Kind, plan.Code).Inc()

	u.log.InfoContext(ctx, "promotion purchased",
		slog.String("op", opPurchase),
		slog.Int64("user_id", userID),
		slog.Int64("product_id", productID),
		slog.String("kind", plan.Kind),
		slog.Int64("promotion_id", promo.ID),
	)

	return dto.PurchasePromotionResponse{
		Promotion:     promotionToDTO(promo, plan.Code),
		WalletBalance: newBalance,
	}, nil
}

func (u *Usecase) buildIdempotentResponse(
	ctx context.Context, userID int64, existing models.WalletTransaction,
) (dto.PurchasePromotionResponse, error) {
	if existing.Type != models.WalletTxTypePromotionCharge || existing.ReferenceID == nil {
		// Ключ использован для другой операции — это конфликт, обозначаем как INVALID_REQUEST.
		return dto.PurchasePromotionResponse{}, ErrInvalidRequest
	}
	promo, err := u.promotions.GetByID(ctx, *existing.ReferenceID)
	if err != nil {
		return dto.PurchasePromotionResponse{}, fmt.Errorf("idempotent: get promotion: %w", err)
	}
	plan, err := u.plans.GetByID(ctx, promo.PlanID)
	if err != nil {
		return dto.PurchasePromotionResponse{}, fmt.Errorf("idempotent: get plan: %w", err)
	}
	balance := int64(0)
	w, err := u.wallets.Get(ctx, userID)
	switch {
	case err == nil:
		balance = w.Balance
	case errors.Is(err, walletrepo.ErrWalletNotFound):
		// допустимо: запись могла не появиться
	default:
		return dto.PurchasePromotionResponse{}, fmt.Errorf("idempotent: get wallet: %w", err)
	}
	return dto.PurchasePromotionResponse{
		Promotion:     promotionToDTO(promo, plan.Code),
		WalletBalance: balance,
	}, nil
}

// ListActiveByAd возвращает активные промо по объявлению.
func (u *Usecase) ListActiveByAd(ctx context.Context, productID int64) ([]dto.PromotionResponse, error) {
	u.log.DebugContext(ctx, "listing active promotions",
		slog.String("op", opListByAd), slog.Int64("product_id", productID))

	promos, err := u.promotions.GetActiveByAd(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("ListActiveByAd: %w", err)
	}
	return u.promotionsWithPlanCodes(ctx, promos)
}

// ListByUser возвращает историю промо пользователя с пагинацией.
func (u *Usecase) ListByUser(
	ctx context.Context, userID, cursor int64, limit int,
) (dto.PromotionListResponse, error) {
	if limit <= 0 {
		limit = defaultListLim
	}
	if limit > maxListLim {
		limit = maxListLim
	}

	u.log.DebugContext(ctx, "listing user promotions",
		slog.String("op", opListByUser), slog.Int64("user_id", userID))

	promos, err := u.promotions.ListByUser(ctx, userID, cursor, limit)
	if err != nil {
		return dto.PromotionListResponse{}, fmt.Errorf("ListByUser: %w", err)
	}
	items, err := u.promotionsWithPlanCodes(ctx, promos)
	if err != nil {
		return dto.PromotionListResponse{}, err
	}
	resp := dto.PromotionListResponse{Items: items}
	if len(promos) == limit {
		last := promos[len(promos)-1].ID
		resp.NextCursor = &last
	}
	return resp, nil
}

// promotionsWithPlanCodes денормализует plan_code в DTO.
// Не оптимально (N+1), но для типичных страниц <= 20 элементов и маленьких таблиц тарифов — ок.
func (u *Usecase) promotionsWithPlanCodes(
	ctx context.Context, promos []models.Promotion,
) ([]dto.PromotionResponse, error) {
	out := make([]dto.PromotionResponse, 0, len(promos))
	planCache := make(map[int64]string)
	for _, p := range promos {
		code, ok := planCache[p.PlanID]
		if !ok {
			plan, err := u.plans.GetByID(ctx, p.PlanID)
			if err != nil {
				return nil, fmt.Errorf("promotionsWithPlanCodes: %w", err)
			}
			code = plan.Code
			planCache[p.PlanID] = code
		}
		out = append(out, promotionToDTO(p, code))
	}
	return out, nil
}

func promotionToDTO(p models.Promotion, planCode string) dto.PromotionResponse {
	return dto.PromotionResponse{
		ID:        p.ID,
		ProductID: p.ProductID,
		Kind:      p.Kind,
		PlanCode:  planCode,
		StartsAt:  p.StartsAt,
		ExpiresAt: p.ExpiresAt,
		PricePaid: p.PricePaid,
	}
}
