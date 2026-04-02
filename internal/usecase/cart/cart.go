package cart

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
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
	Checkout(ctx context.Context, buyerID int64) ([]int64, map[int64]*dto.SellerContact, error)
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
	const op = "usecase.cart.AddToCart"
	log := u.log.With(slog.String("op", op), slog.Int64("user_id", userID), slog.Int64("product_id", productID))

	// Проверяем статус и владельца
	ad, err := u.adsStorage.GetAdByID(ctx, productID)
	if err != nil {
		log.Error("failed to get product", "error", err)
		return fmt.Errorf("failed to get product: %w", err)
	}

	if ad.Status != "active" {
		return ErrProductNotActive
	}

	if ad.SellerID == userID {
		return ErrCannotAddOwnProduct
	}

	err = u.cartStorage.Add(ctx, userID, productID)
	if err != nil {
		log.Error("failed to add product to cart", "error", err)
		return err
	}

	log.Info("product added to cart successfully")
	return nil
}

// RemoveFromCart удаляет товар из корзины.
func (u *Usecase) RemoveFromCart(ctx context.Context, userID, productID int64) error {
	const op = "usecase.cart.RemoveFromCart"
	log := u.log.With(slog.String("op", op), slog.Int64("user_id", userID), slog.Int64("product_id", productID))

	err := u.cartStorage.Remove(ctx, userID, productID)
	if err != nil {
		log.Error("failed to remove product from cart", "error", err)
		return err
	}

	return nil
}

// GetCart возвращает корзину.
func (u *Usecase) GetCart(ctx context.Context, userID int64) (*dto.CartResponse, error) {
	const op = "usecase.cart.GetCart"
	log := u.log.With(slog.String("op", op), slog.Int64("user_id", userID))

	items, err := u.cartStorage.GetByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to get cart items", "error", err)
		return nil, err
	}

	var total int64
	for _, item := range items {
		total += item.Price
	}

	return &dto.CartResponse{
		Items:      items,
		TotalPrice: total,
	}, nil
}

// Checkout оформляет заказ на всю корзину.
func (u *Usecase) Checkout(ctx context.Context, userID int64) (*dto.CheckoutResponse, error) {
	const op = "usecase.cart.Checkout"
	log := u.log.With(slog.String("op", op), slog.Int64("user_id", userID))

	orderIDs, sellers, err := u.cartStorage.Checkout(ctx, userID)
	if err != nil {
		log.Error("failed to checkout", "error", err)
		return nil, err
	}

	log.Info("checkout completed successfully")

	return &dto.CheckoutResponse{
		OrderIDs: orderIDs,
		Sellers:  sellers,
	}, nil
}
