package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
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

// a.router.HandleFunc("GET "+api.ApiPrefix+"/ads", a.adsHandlers.HandleGetAds)
// a.router.HandleFunc("POST "+api.ApiPrefix+"/ads", a.adsHandlers.HandleCreateAd)
// a.router.HandleFunc("GET "+api.ApiPrefix+"/ads/{id}", a.adsHandlers.HandleGetAdByID)
// a.router.HandleFunc("PUT "+api.ApiPrefix+"/ads/{id}", a.adsHandlers.HandleUpdateAdByID)
// a.router.HandleFunc("DELETE "+api.ApiPrefix+"/ads/{id}", a.adsHandlers.HandleDeleteAd)
// a.router.HandleFunc("POST "+api.ApiPrefix+"/ads/{id}/close", a.adsHandlers.HandleCloseAdByID)
