package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	ad "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/ad"
	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/sanitizer"
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
// @Failure 400 {object} dto.ErrorResponse "ad not found: Объявление не найдено"
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
			responser.RespondWithError(w, http.StatusBadRequest, ErrAdNotFound)
			return
		}
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, adItem)
}

// HandleCreateAd обрабатывает запрос на создание нового объявления
// @Summary Создать объявление
// @Description Создает новое объявление с фотографиями через multipart/form-data. Доступно только авторизованным пользователям.
// @Tags ads
// @Accept multipart/form-data
// @Produce json
// @Param data formData string true "JSON с данными объявления (title, description, price, category_id, status, location)"
// @Param photos formData file false "Фотографии объявления (можно несколько)"
// @Success 200 {object} map[string]int64 "ID созданного объявления"
// @Failure 400 {object} dto.ErrorResponse "invalid request body / ошибки валидации"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /ads [post]
func (h *AdsHandlers) HandleCreateAd(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.HandleCreateAd"

	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Ограничиваем размер запроса (50 MB на все фото)
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		h.log.Error("parse multipart form error", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusBadRequest, ErrFileTooBig)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	// Парсим JSON-данные из поля "data"
	var req dto.CreateAdRequest
	dataField := r.FormValue("data")
	if dataField == "" {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}
	if err := json.NewDecoder(strings.NewReader(dataField)).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	req.UserID = userID

	// Удаляем HTML-теги из текстовых полей (защита от XSS)
	req.Title = sanitizer.StripHTML(req.Title)
	req.Description = sanitizer.StripHTML(req.Description)
	req.Location = sanitizer.StripHTML(req.Location)

	// Валидируем запрос
	validationErrors := validator.ValidateCreateAdRequest(&req)
	if validationErrors.HasErrors() {
		responser.RespondWithJSON(w, http.StatusBadRequest, validationErrors)
		return
	}

	// Загружаем фотографии в S3
	photoURLs, err := h.uploadPhotosFromForm(r)
	if err != nil {
		h.log.Error(ErrFailedToUploadPhotos, slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusBadRequest, ErrFailedToUploadPhotos)
		return
	}
	req.Photos = photoURLs

	// Создаем новое объявление
	adID, err := h.services.Ads.CreateAd(r.Context(), &req)
	if err != nil {
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	// Возвращаем ID созданного объявления
	responser.RespondWithJSON(w, http.StatusOK, map[string]int64{"ad_id": adID})
}

// HandleUpdateAdByID обрабатывает запрос на обновление объявления
// @Summary Обновить объявление
// @Description Обновляет объявление по заданному ID с возможностью замены фотографий через multipart/form-data. Доступно только владельцу.
// @Tags ads
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID объявления"
// @Param data formData string true "JSON с данными для обновления (title, description, price, category_id, status, location)"
// @Param photos formData file false "Новые фотографии объявления (заменяют старые)"
// @Success 200 {object} map[string]string "объявление успешно обновлено"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id / invalid request body / ошибки валидации"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 400 {object} dto.ErrorResponse "ad not found: Объявление не найдено или не принадлежит пользователю"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /ads/{id} [put]
func (h *AdsHandlers) HandleUpdateAdByID(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.HandleUpdateAdByID"

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

	// Ограничиваем размер запроса (50 MB на все фото)
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		h.log.Error("parse multipart form error", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusBadRequest, ErrFileTooBig)
		return
	}

	// Парсим JSON-данные из поля "data"
	var req dto.UpdateAdRequest
	dataField := r.FormValue("data")
	if dataField == "" {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}
	if err := json.NewDecoder(strings.NewReader(dataField)).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	req.ID = id
	req.UserID = userID

	// Удаляем HTML-теги из текстовых полей (защита от XSS)
	req.Title = sanitizer.StripHTML(req.Title)
	req.Description = sanitizer.StripHTML(req.Description)
	req.Location = sanitizer.StripHTML(req.Location)

	// Валидируем запрос
	validationErrors := validator.ValidateUpdateAdRequest(&req)
	if validationErrors.HasErrors() {
		responser.RespondWithJSON(w, http.StatusBadRequest, validationErrors)
		return
	}

	// Загружаем фотографии в S3 (если есть)
	photoURLs, err := h.uploadPhotosFromForm(r)
	if err != nil {
		h.log.Error(ErrFailedToUploadPhotos, slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusBadRequest, ErrFailedToUploadPhotos)
		return
	}
	req.Photos = photoURLs

	// Обновляем объявление
	if err := h.services.Ads.UpdateAd(r.Context(), &req); err != nil {
		if errors.Is(err, ad.ErrAdNotFound) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrAdNotFound)
			return
		}
		if errors.Is(err, ad.ErrAdForbidden) {
			responser.RespondWithError(w, http.StatusForbidden, ErrForbidden)
			return
		}
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// uploadPhotosFromForm извлекает фотографии из multipart формы и загружает их в S3.
func (h *AdsHandlers) uploadPhotosFromForm(r *http.Request) ([]string, error) {
	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		return nil, nil
	}

	var openFiles []multipart.File
	var filenames []string
	defer func() {
		for _, f := range openFiles {
			f.Close()
		}
	}()

	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open uploaded file: %w", err)
		}
		openFiles = append(openFiles, f)
		filenames = append(filenames, fh.Filename)
	}

	return h.services.Ads.UploadAdPhotos(r.Context(), openFiles, filenames)
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
// @Failure 400 {object} dto.ErrorResponse "ad not found: Объявление не найдено или не принадлежит пользователю"
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
			responser.RespondWithError(w, http.StatusBadRequest, ErrAdNotFound)
			return
		}
		if errors.Is(err, ad.ErrAdForbidden) {
			responser.RespondWithError(w, http.StatusForbidden, ErrForbidden)
			return
		}
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

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
// @Failure 0 {object} dto.ErrorResponse "ad not found: Объявление не найдено, уже архивировано или не принадлежит пользователю"
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
			responser.RespondWithError(w, http.StatusBadRequest, ErrAdNotFound)
			return
		}
		if errors.Is(err, ad.ErrAdForbidden) {
			responser.RespondWithError(w, http.StatusForbidden, ErrForbidden)
			return
		}
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

// HandleGetUserAds обрабатывает запросы на получение объявлений продавца по его ID
// @Summary Получить объявления пользователя
// @Description Возвращает список всех объявлений продавца по его ID
// @Tags ads
// @Produce json
// @Param id path int true "ID пользователя"
// @Success 200 {object} map[string][]models.Ad "объявления успешно получены"
// @Failure 400 {object} map[string]string "некорректный ID пользователя"
// @Failure 500 {object} map[string]string "ошибка сервера"
// @Router /users/{id}/ads [get]
func (h *AdsHandlers) HandleGetUserAds(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.HandleGetUserAds"

	// получаем ID из /api/v1/users/{id}/ads
	idStr := r.PathValue("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.log.Error(ErrInvalidUserID, slog.String("op", op), slog.String("id", idStr))
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidUserID)
		return
	}

	// получаем список объявлений
	ads, err := h.services.Ads.GetAdsByUserID(r.Context(), userID)
	if err != nil {
		h.log.Error(ErrFailedToGetUserAds, slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToGetUserAds)
		return
	}

	// формируем ответ
	responser.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"ads": ads,
	})
}

// HandleAddToFavorites обрабатывает запросы на добавление объявления в избранное
// @Summary Добавить в избранное
// @Description Добавляет объявление в список избранного текущего пользователя
// @Tags favorites
// @Produce json
// @Param id path int true "ID объявления"
// @Success 200 {object} map[string]string "статус операции"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id: Некорректный ID"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /ads/{id}/favorite [post]
func (h *AdsHandlers) HandleAddToFavorites(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.HandleAddToFavorites"

	// извлекаем userID из контекста (туда его положил authMW)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		h.log.Error("user id not found in context", slog.String("op", op))
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	// получаем ID объявления
	idStr := r.PathValue("id")
	adID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	// добавляем объявление в избранное
	if err := h.services.Ads.AddFavorite(r.Context(), userID, adID); err != nil {
		h.log.Error("failed to add favorite", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleDeleteFromFavorites обрабатывает запросы на удаление объявления из избранного
// @Summary Удалить из избранного
// @Description Удаляет объявление из списка избранного текущего пользователя
// @Tags favorites
// @Produce json
// @Param id path int true "ID объявления"
// @Success 200 {object} map[string]string "статус операции"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id: Некорректный ID"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /ads/{id}/favorite [delete]
func (h *AdsHandlers) HandleDeleteFromFavorites(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.HandleDeleteFromFavorites"

	// извлекаем userID из контекста (туда его положил authMW)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		h.log.Error("user id not found in context", slog.String("op", op))
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	// получаем ID объявления
	idStr := r.PathValue("id")
	adID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	// удаляем объявление из избранного
	if err := h.services.Ads.RemoveFavorite(r.Context(), userID, adID); err != nil {
		h.log.Error("failed to remove favorite", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleGetFavorites обрабатывает запросы на получение объявлений в избранном
// @Summary Получить избранное
// @Description Возвращает список всех объявлений, добавленных текущим пользователем в избранное
// @Tags favorites
// @Produce json
// @Success 200 {object} map[string][]models.Ad "список избранных объявлений"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /profile/favorites [get]
func (h *AdsHandlers) HandleGetFavorites(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.HandleGetFavorites"

	// извлекаем userID из контекста (туда его положил authMW)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		h.log.Error("user id not found in context", slog.String("op", op))
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	// получаем список объявлений из избранного
	favorites, err := h.services.Ads.GetUserFavorites(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get favorites", slog.String("op", op), slog.String("error", err.Error()))
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"ads": favorites,
	})
}
