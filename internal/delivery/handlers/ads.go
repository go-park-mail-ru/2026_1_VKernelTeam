package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	ad "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/ad"
	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/validator"
)

// HandleGetAds обрабатывает запросы на получение списка объявлений
// @Summary Получить список объявлений
// @Description Возвращает список всех объявлений
// @Tags ads
// @Produce json
// @Success 200 {array} models.Ad "список объявлений успешно получен"
// @Failure 400 {object} dto.ErrorResponse "Method not allowed: Метод не поддерживается (ожидается GET)"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера при получении объявлений"
// @Router /ads [get]
func (h *AdsHandlers) HandleGetAds(w http.ResponseWriter, r *http.Request) {
	// обрабатываем только GET запросы
	if r.Method != http.MethodGet {
		responser.RespondWithError(w, http.StatusBadRequest, ErrMethodNotAllowed)
		return
	}

	// копируем список объявлений из сервиса и возвращаем его клиенту
	adsList, err := h.services.Ads.GetAllAds(r.Context())
	if err != nil {
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	// формируем и отправляем ответ
	responser.RespondWithJSON(w, http.StatusOK, adsList)
}

// HandleGetAdByID обрабатывает запрос на получение объявления по ID
// @Summary Получить объявление по ID
// @Description Возвращает объявление по заданному ID
// @Tags ads
// @Produce json
// @Param id path int true "ID объявления"
// @Success 200 {object} models.Ad "объявление успешно получено"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id: Некорректный ID объявления"
// @Failure 404 {object} dto.ErrorResponse "ad not found: Объявление не найдено"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера при получении объявления"
// @Router /ads/{id} [get]
func (h *AdsHandlers) HandleGetAdByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	adItem, err := h.services.Ads.GetAdByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ad.ErrAdNotFound) {
			responser.RespondWithError(w, http.StatusNotFound, ErrAdNotFound)
			return
		}
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, adItem)
}

func (h *AdsHandlers) HandleCreateAd(w http.ResponseWriter, r *http.Request) {
	// Извлекаем UserID из JWT-контекста (установлен AuthMiddleware)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.CreateAdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	// Устанавливаем UserID из токена, а не из тела запроса
	req.UserID = userID

	// Собираем все ошибки валидации
	validationErrors := dto.ValidationErrors{}
	if err := validator.ValidateCategoryID(req.CategoryID); err != nil {
		validationErrors.CategoryID = err.Error()
	}
	if err := validator.ValidateAdTitle(req.Title); err != nil {
		validationErrors.Title = err.Error()
	}
	if err := validator.ValidateAdDescription(req.Description); err != nil {
		validationErrors.Description = err.Error()
	}
	if err := validator.ValidateAdPrice(req.Price); err != nil {
		validationErrors.Price = err.Error()
	}
	if err := validator.ValidateAdStatus(req.Status); err != nil {
		validationErrors.Status = err.Error()
	}

	// Если есть хотя бы одна ошибка валидации, возвращаем их все
	if validationErrors.CategoryID != "" || validationErrors.Title != "" || validationErrors.Description != "" || validationErrors.Price != "" || validationErrors.Status != "" {
		responser.RespondWithJSON(w, http.StatusBadRequest, validationErrors)
		return
	}

	// Создаем новое объявление
	adID, err := h.services.Ads.CreateAd(r.Context(), &req)
	if err != nil {
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	// Обновляем CSRF-токен после успешного мутирующего запроса
	csrfToken := middleware.GenerateCSRFToken()
	setCsrfCookie(w, csrfToken, h.tokenTTL)

	// Возвращаем ID созданного объявления
	responser.RespondWithJSON(w, http.StatusOK, map[string]int64{"ad_id": adID})
}

// HandleUpdateAdByID обрабатывает запрос на обновление объявления
// @Summary Обновить объявление
// @Description Обновляет объявление по заданному ID. Доступно только владельцу объявления.
// @Tags ads
// @Accept json
// @Produce json
// @Param id path int true "ID объявления"
// @Param body body dto.UpdateAdRequest true "Данные для обновления"
// @Success 200 {object} map[string]string "объявление успешно обновлено"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id / invalid request body / ошибки валидации"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 404 {object} dto.ErrorResponse "ad not found: Объявление не найдено или не принадлежит пользователю"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /ads/{id} [put]
func (h *AdsHandlers) HandleUpdateAdByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем UserID из JWT-контекста
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Парсим ID из URL
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	// Декодируем тело запроса
	var req dto.UpdateAdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	// Устанавливаем ID и UserID из URL и токена
	req.ID = id
	req.UserID = userID

	// Собираем ошибки валидации
	validationErrors := dto.ValidationErrors{}
	if err := validator.ValidateCategoryID(req.CategoryID); err != nil {
		validationErrors.CategoryID = err.Error()
	}
	if err := validator.ValidateAdTitle(req.Title); err != nil {
		validationErrors.Title = err.Error()
	}
	if err := validator.ValidateAdDescription(req.Description); err != nil {
		validationErrors.Description = err.Error()
	}
	if err := validator.ValidateAdPrice(req.Price); err != nil {
		validationErrors.Price = err.Error()
	}
	if err := validator.ValidateAdStatus(req.Status); err != nil {
		validationErrors.Status = err.Error()
	}

	if validationErrors.CategoryID != "" || validationErrors.Title != "" || validationErrors.Description != "" || validationErrors.Price != "" || validationErrors.Status != "" {
		responser.RespondWithJSON(w, http.StatusBadRequest, validationErrors)
		return
	}

	// Обновляем объявление
	if err := h.services.Ads.UpdateAd(r.Context(), &req); err != nil {
		if errors.Is(err, ad.ErrAdNotFound) {
			responser.RespondWithError(w, http.StatusNotFound, ErrAdNotFound)
			return
		}
		if errors.Is(err, ad.ErrAdForbidden) {
			responser.RespondWithError(w, http.StatusForbidden, ErrForbidden)
			return
		}
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	// Обновляем CSRF-токен после успешного мутирующего запроса
	csrfToken := middleware.GenerateCSRFToken()
	setCsrfCookie(w, csrfToken, h.tokenTTL)

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// HandleDeleteAd обрабатывает запрос на удаление объявления
// @Summary Удалить объявление
// @Description Мягкое удаление объявления. Доступно только владельцу.
// @Tags ads
// @Produce json
// @Param id path int true "ID объявления"
// @Success 200 {object} map[string]string "объявление успешно удалено"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id: Некорректный ID"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 404 {object} dto.ErrorResponse "ad not found: Объявление не найдено или не принадлежит пользователю"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /ads/{id} [delete]
func (h *AdsHandlers) HandleDeleteAd(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	if err := h.services.Ads.DeleteAd(r.Context(), id, userID); err != nil {
		if errors.Is(err, ad.ErrAdNotFound) {
			responser.RespondWithError(w, http.StatusNotFound, ErrAdNotFound)
			return
		}
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	csrfToken := middleware.GenerateCSRFToken()
	setCsrfCookie(w, csrfToken, h.tokenTTL)

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// HandleCloseAdByID обрабатывает запрос на закрытие объявления
// @Summary Закрыть объявление
// @Description Устанавливает статус объявления в 'archived'. Доступно только владельцу.
// @Tags ads
// @Produce json
// @Param id path int true "ID объявления"
// @Success 200 {object} map[string]string "объявление успешно архивировано"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id: Некорректный ID"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 404 {object} dto.ErrorResponse "ad not found: Объявление не найдено, уже архивировано или не принадлежит пользователю"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /ads/{id}/close [post]
func (h *AdsHandlers) HandleCloseAdByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	if err := h.services.Ads.CloseAd(r.Context(), id, userID); err != nil {
		if errors.Is(err, ad.ErrAdNotFound) {
			responser.RespondWithError(w, http.StatusNotFound, ErrAdNotFound)
			return
		}
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	csrfToken := middleware.GenerateCSRFToken()
	setCsrfCookie(w, csrfToken, h.tokenTTL)

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "archived"})
}
