package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/mailru/easyjson"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/validator"
)

// HandleGetCart обрабатывает запросы на получение списка товаров в корзине
// @Summary Получить корзину
// @Description Возвращает список товаров в корзине текущего пользователя
// @Tags cart
// @Produce json
// @Success 200 {object} dto.CartResponse "корзина успешно получена"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /cart [get]
func (h *CartHandlers) HandleGetCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	cart, err := h.services.Cart.GetCart(r.Context(), userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get cart",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, cart)
}

// HandleAddToCart добавляет товар в корзину
// @Summary Добавить в корзину
// @Description Добавляет товар в корзину пользователя
// @Tags cart
// @Accept json
// @Produce json
// @Param body body dto.AddToCartRequest true "Данные (product_id)"
// @Success 200 {object} map[string]string "статус операции"
// @Failure 400 {object} dto.ErrorResponse "invalid request body / invalid product id / product is not active / cannot add own product to cart / product already in cart: Ошибка валидации или бизнес-логики"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /cart [post]
func (h *CartHandlers) HandleAddToCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	var req dto.AddToCartRequest
	if err := easyjson.UnmarshalFromReader(r.Body, &req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	if err := validator.ValidatePositiveInt64("product_id", req.ProductID); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	err := h.services.Cart.AddToCart(r.Context(), userID, req.ProductID)
	if err != nil {
		h.log.WarnContext(r.Context(), "failed to add to cart",
			slog.Int64("user_id", userID),
			slog.Int64("product_id", req.ProductID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

// HandleRemoveFromCart удаляет товар из корзины
// @Summary Удалить из корзины
// @Description Удаляет товар из корзины пользователя по ID товара
// @Tags cart
// @Produce json
// @Param id path int true "ID продукта"
// @Success 200 {object} map[string]string "статус операции"
// @Failure 400 {object} dto.ErrorResponse "invalid product id / cart item not found: Некорректный ID или товар не найден в корзине"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /cart/{id} [delete]
func (h *CartHandlers) HandleRemoveFromCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	productID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidProductID)
		return
	}

	if err := validator.ValidatePositiveInt64("product_id", productID); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.services.Cart.RemoveFromCart(r.Context(), userID, productID)
	if err != nil {
		h.log.WarnContext(r.Context(), "failed to remove from cart",
			slog.Int64("user_id", userID),
			slog.Int64("product_id", productID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}
