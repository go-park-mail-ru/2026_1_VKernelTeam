package cart

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

const (
	opAddToCart      = "usecase.cart.AddToCart"
	opRemoveFromCart = "usecase.cart.RemoveFromCart"
	opGetCart        = "usecase.cart.GetCart"
)

// Sentinel-ошибки
var (
	ErrProductNotActive    = errors.New("product is not active")
	ErrCannotAddOwnProduct = errors.New("cannot add own product to cart")
)

//go:generate mockgen -source=usecase.go -destination=mocks/mock_cart.go -package=mocks

// CartProvider интерфейс репозитория корзины
type CartProvider interface {
	Add(ctx context.Context, userID, productID int64) error
	Remove(ctx context.Context, userID, productID int64) error
	GetByUserID(ctx context.Context, userID int64) ([]dto.CartItemResponse, error)
	Clear(ctx context.Context, userID int64) error
}

// AdsProvider нужен для получения инфы об объявлении при добавлении в корзину
type AdsProvider interface {
	GetAdByID(ctx context.Context, id int64) (models.Ad, error)
}

type Usecase struct {
	log         *slog.Logger
	cartStorage CartProvider
	adsStorage  AdsProvider
}

// New создаёт новый экземпляр usecase для корзины.
func New(log *slog.Logger, cartStorage CartProvider, adsStorage AdsProvider) *Usecase {
	return &Usecase{
		log:         log,
		cartStorage: cartStorage,
		adsStorage:  adsStorage,
	}
}

// AddToCart добавляет товар в корзину с проверками.
func (u *Usecase) AddToCart(ctx context.Context, userID, productID int64) error {
	u.log.InfoContext(ctx, "adding product to cart",
		slog.String("op", opAddToCart),
		slog.Int64("user_id", userID),
		slog.Int64("product_id", productID),
	)

	ad, err := u.adsStorage.GetAdByID(ctx, productID)
	if err != nil {
		u.log.ErrorContext(ctx, "failed to get product",
			slog.String("op", opAddToCart),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to get product: %w", err)
	}

	if ad.Status != "active" {
		u.log.WarnContext(ctx, "product is not active",
			slog.String("op", opAddToCart),
			slog.Int64("product_id", productID),
			slog.String("status", ad.Status),
		)
		return ErrProductNotActive
	}

	if ad.SellerID == userID {
		u.log.WarnContext(ctx, "user tried to add own product to cart",
			slog.String("op", opAddToCart),
			slog.Int64("user_id", userID),
			slog.Int64("product_id", productID),
		)
		return ErrCannotAddOwnProduct
	}

	err = u.cartStorage.Add(ctx, userID, productID)
	if err != nil {
		u.log.ErrorContext(ctx, "failed to add product to cart",
			slog.String("op", opAddToCart),
			slog.String("error", err.Error()),
		)
		return err
	}

	u.log.InfoContext(ctx, "product added to cart successfully",
		slog.String("op", opAddToCart),
		slog.Int64("user_id", userID),
		slog.Int64("product_id", productID),
	)
	return nil
}

// RemoveFromCart удаляет товар из корзины.
func (u *Usecase) RemoveFromCart(ctx context.Context, userID, productID int64) error {
	u.log.InfoContext(ctx, "removing product from cart",
		slog.String("op", opRemoveFromCart),
		slog.Int64("user_id", userID),
		slog.Int64("product_id", productID),
	)

	err := u.cartStorage.Remove(ctx, userID, productID)
	if err != nil {
		u.log.ErrorContext(ctx, "failed to remove product from cart",
			slog.String("op", opRemoveFromCart),
			slog.String("error", err.Error()),
		)
		return err
	}

	u.log.InfoContext(ctx, "product removed from cart",
		slog.String("op", opRemoveFromCart),
		slog.Int64("user_id", userID),
		slog.Int64("product_id", productID),
	)
	return nil
}

// GetCart возвращает корзину.
func (u *Usecase) GetCart(ctx context.Context, userID int64) (*dto.CartResponse, error) {
	u.log.DebugContext(ctx, "getting cart",
		slog.String("op", opGetCart),
		slog.Int64("user_id", userID),
	)

	items, err := u.cartStorage.GetByUserID(ctx, userID)
	if err != nil {
		u.log.ErrorContext(ctx, "failed to get cart items",
			slog.String("op", opGetCart),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	var total int64
	for _, item := range items {
		total += item.Price
	}

	u.log.DebugContext(ctx, "cart fetched",
		slog.String("op", opGetCart),
		slog.Int64("user_id", userID),
		slog.Int("items_count", len(items)),
	)
	return &dto.CartResponse{
		Items:      items,
		TotalPrice: total,
	}, nil
}

