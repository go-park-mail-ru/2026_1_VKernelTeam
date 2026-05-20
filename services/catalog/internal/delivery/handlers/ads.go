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

	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/sanitizer"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/dto"
	ad "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/ad"
	adsUC "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/usecase/ads"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/validator"
)

const (
	opHandleCreateAd            = "handlers.HandleCreateAd"
	opHandleUpdateAdByID        = "handlers.HandleUpdateAdByID"
	opHandleGetUserAds          = "handlers.HandleGetUserAds"
	opHandleAddToFavorites      = "handlers.HandleAddToFavorites"
	opHandleDeleteFromFavorites = "handlers.HandleDeleteFromFavorites"
	opHandleGetFavorites        = "handlers.HandleGetFavorites"
	opHandleSearchAds           = "handlers.HandleSearchAds"
	opHandleGetPriceHistory     = "handlers.HandleGetPriceHistory"

	ErrSearchQueryRequired = "query parameter is required"
	ErrSearchQueryTooShort = "search query is too short"

	statusKey = "status"
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
	if r.Method != http.MethodGet {
		responser.RespondWithError(w, http.StatusBadRequest, ErrMethodNotAllowed)
		return
	}

	adsList, err := h.services.Ads.GetAllAds(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get ads list",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, adsList)
}

// HandleSearchAds обрабатывает запросы на поиск объявлений
// @Summary Поиск объявлений
// @Description Поиск объявлений по заголовку и описанию с поддержкой триграмм, транслитерации и смены раскладки
// @Tags ads
// @Produce json
// @Param query query string true "Поисковый запрос"
// @Param category_id query int false "ID категории (фильтр)"
// @Success 200 {array} models.Ad "результаты поиска"
// @Failure 400 {object} dto.ErrorResponse "query parameter is required / search query is too short"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Router /ads/search [get]
func (h *AdsHandlers) HandleSearchAds(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		responser.RespondWithError(w, http.StatusBadRequest, ErrSearchQueryRequired)
		return
	}

	var categoryID int64
	if catStr := r.URL.Query().Get("category_id"); catStr != "" {
		catID, err := strconv.ParseInt(catStr, 10, 64)
		if err != nil {
			responser.RespondWithError(w, http.StatusBadRequest, "invalid category_id parameter")
			return
		}
		categoryID = catID
	}

	ads, err := h.services.Ads.SearchAds(r.Context(), query, categoryID)
	if err != nil {
		if errors.Is(err, adsUC.ErrQueryTooShort) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrSearchQueryTooShort)
			return
		}
		h.log.ErrorContext(r.Context(), "failed to search ads",
			slog.String("op", opHandleSearchAds),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, ads)
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
		h.log.ErrorContext(r.Context(), "failed to get ad by id",
			slog.Int64("ad_id", id),
			slog.String("error", err.Error()),
		)
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
// @Param data formData string true "JSON с данными объявления (title, description, price, category_id, status, location, lat, lon)"
// @Param photos formData file false "Фотографии объявления (можно несколько)"
// @Success 200 {object} map[string]int64 "ID созданного объявления"
// @Failure 400 {object} dto.ErrorResponse "invalid request body / ошибки валидации"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /ads [post]
func (h *AdsHandlers) HandleCreateAd(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		h.log.ErrorContext(r.Context(), "parse multipart form error",
			slog.String("op", opHandleCreateAd),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusBadRequest, ErrFileTooBig)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

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

	req.Title = sanitizer.StripHTML(req.Title)
	req.Description = sanitizer.StripHTML(req.Description)
	req.Location = sanitizer.StripHTML(req.Location)

	validationErrors := validateCreateAdRequest(&req)
	if validationErrors.HasErrors() {
		responser.RespondWithJSON(w, http.StatusBadRequest, validationErrors)
		return
	}

	photoURLs, err := h.uploadPhotosFromForm(r)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to upload photos",
			slog.String("op", opHandleCreateAd),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusBadRequest, ErrFailedToUploadPhotos)
		return
	}
	req.Photos = photoURLs

	adID, err := h.services.Ads.CreateAd(r.Context(), &req)
	if err != nil {
		if isCharacteristicValidationError(err) {
			responser.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.log.ErrorContext(r.Context(), "failed to create ad",
			slog.String("op", opHandleCreateAd),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]int64{"ad_id": adID})
}

// HandleUpdateAdByID обрабатывает запрос на обновление объявления
// @Summary Обновить объявление
// @Description Обновляет объявление по заданному ID с возможностью замены фотографий через multipart/form-data. Доступно только владельцу.
// @Tags ads
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID объявления"
// @Param data formData string true "JSON с данными для обновления (title, description, price, category_id, status, location, lat, lon)"
// @Param photos formData file false "Новые фотографии объявления (заменяют старые)"
// @Success 200 {object} map[string]string "объявление успешно обновлено"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id / invalid request body / ошибки валидации"
// @Failure 401 {object} dto.ErrorResponse "unauthorized: Пользователь не авторизован"
// @Failure 400 {object} dto.ErrorResponse "ad not found: Объявление не найдено или не принадлежит пользователю"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера"
// @Security CookieAuth
// @Router /ads/{id} [put]
func (h *AdsHandlers) HandleUpdateAdByID(w http.ResponseWriter, r *http.Request) {
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

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		h.log.ErrorContext(r.Context(), "parse multipart form error",
			slog.String("op", opHandleUpdateAdByID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusBadRequest, ErrFileTooBig)
		return
	}

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

	if req.Title != nil {
		*req.Title = sanitizer.StripHTML(*req.Title)
	}
	if req.Description != nil {
		*req.Description = sanitizer.StripHTML(*req.Description)
	}
	if req.Location != nil {
		*req.Location = sanitizer.StripHTML(*req.Location)
	}

	validationErrors := validateUpdateAdRequest(&req)
	if validationErrors.HasErrors() {
		responser.RespondWithJSON(w, http.StatusBadRequest, validationErrors)
		return
	}

	photoURLs, err := h.uploadPhotosFromForm(r)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to upload photos",
			slog.String("op", opHandleUpdateAdByID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusBadRequest, ErrFailedToUploadPhotos)
		return
	}
	req.Photos = photoURLs

	if err := h.services.Ads.UpdateAd(r.Context(), &req); err != nil {
		if errors.Is(err, ad.ErrAdNotFound) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrAdNotFound)
			return
		}
		if errors.Is(err, ad.ErrAdForbidden) {
			responser.RespondWithError(w, http.StatusForbidden, ErrForbidden)
			return
		}
		if isCharacteristicValidationError(err) {
			responser.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.log.ErrorContext(r.Context(), "failed to update ad",
			slog.String("op", opHandleUpdateAdByID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{statusKey: "updated"})
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
			_ = f.Close()
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
		h.log.ErrorContext(r.Context(), "failed to delete ad",
			slog.Int64("ad_id", id),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{statusKey: "deleted"})
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
		h.log.ErrorContext(r.Context(), "failed to close ad",
			slog.Int64("ad_id", id),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{statusKey: "archived"})
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
	idStr := r.PathValue("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.log.WarnContext(r.Context(), "invalid user id",
			slog.String("op", opHandleGetUserAds),
			slog.String("id", idStr),
		)
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidUserID)
		return
	}

	ads, err := h.services.Ads.GetAdsByUserID(r.Context(), userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get user ads",
			slog.String("op", opHandleGetUserAds),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToGetUserAds)
		return
	}

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
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		h.log.ErrorContext(r.Context(), "user id not found in context",
			slog.String("op", opHandleAddToFavorites),
		)
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	adID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	if err := h.services.Ads.AddFavorite(r.Context(), userID, adID); err != nil {
		h.log.ErrorContext(r.Context(), "failed to add favorite",
			slog.String("op", opHandleAddToFavorites),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{statusKey: "ok"})
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
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		h.log.ErrorContext(r.Context(), "user id not found in context",
			slog.String("op", opHandleDeleteFromFavorites),
		)
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	adID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	if err := h.services.Ads.RemoveFavorite(r.Context(), userID, adID); err != nil {
		h.log.ErrorContext(r.Context(), "failed to remove favorite",
			slog.String("op", opHandleDeleteFromFavorites),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{statusKey: "ok"})
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
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		h.log.ErrorContext(r.Context(), "user id not found in context",
			slog.String("op", opHandleGetFavorites),
		)
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	favorites, err := h.services.Ads.GetUserFavorites(r.Context(), userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get favorites",
			slog.String("op", opHandleGetFavorites),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"ads": favorites,
	})
}

// HandleGetCategoryCharacteristics возвращает определения характеристик для категории.
// @Summary Получить характеристики категории
// @Description Возвращает список предопределённых характеристик категории с допустимыми значениями
// @Tags categories
// @Produce json
// @Param id path int true "ID категории"
// @Success 200 {array} models.CategoryCharacteristic "список характеристик"
// @Failure 400 {object} dto.ErrorResponse "invalid category id"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Router /categories/{id}/characteristics [get]
func (h *AdsHandlers) HandleGetCategoryCharacteristics(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	categoryID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, "invalid category id")
		return
	}

	chars, err := h.services.Ads.GetCategoryCharacteristics(r.Context(), categoryID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get category characteristics",
			slog.Int64("category_id", categoryID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, chars)
}

// HandleGetPriceHistory обрабатывает запрос на получение истории цен объявления.
// @Summary История цен объявления
// @Description Возвращает историю изменения цены объявления (даты и соответствующие цены)
// @Tags ads
// @Produce json
// @Param id path int true "ID объявления"
// @Success 200 {object} dto.PriceHistoryResponse "история цен успешно получена"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id: Некорректный ID объявления"
// @Failure 400 {object} dto.ErrorResponse "ad not found: Объявление не найдено"
// @Failure 500 {object} dto.ErrorResponse "internal error: Ошибка сервера при получении истории цен"
// @Router /ads/{id}/price-history [get]
func (h *AdsHandlers) HandleGetPriceHistory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	history, err := h.services.Ads.GetPriceHistory(r.Context(), id)
	if err != nil {
		if errors.Is(err, ad.ErrAdNotFound) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrAdNotFound)
			return
		}
		h.log.ErrorContext(r.Context(), "failed to get price history",
			slog.String("op", opHandleGetPriceHistory),
			slog.Int64("ad_id", id),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, dto.PriceHistoryResponse{History: history})
}

// isCharacteristicValidationError проверяет, является ли ошибка валидацией характеристик.
func isCharacteristicValidationError(err error) bool {
	return errors.Is(err, validator.ErrCharacteristicIDInvalid) ||
		errors.Is(err, validator.ErrCharacteristicValueTooLong) ||
		errors.Is(err, validator.ErrCharacteristicValueNotInEnum) ||
		errors.Is(err, validator.ErrCustomCharacteristicsTooMany) ||
		errors.Is(err, validator.ErrCustomCharNameEmpty) ||
		errors.Is(err, validator.ErrCustomCharNameTooLong) ||
		errors.Is(err, validator.ErrCustomCharNameDuplicate) ||
		errors.Is(err, validator.ErrCustomCharValueTooLong)
}

// validateCreateAdRequest и validateUpdateAdRequest — локальные обёртки. После
// рефакторинга pkg/validator (он больше не зависит от dto/models) агрегаторы
// живут в самих handlers, чтобы каждый сервис собирал свой ValidationErrors.
// Монолит будет удалён в фазе 4.
func validateCreateAdRequest(req *dto.CreateAdRequest) *dto.ValidationErrors {
	errs := dto.ValidationErrors{}
	if err := validator.ValidateCategoryID(req.CategoryID); err != nil {
		errs.CategoryID = err.Error()
	}
	if err := validator.ValidateAdTitle(req.Title); err != nil {
		errs.Title = err.Error()
	}
	if err := validator.ValidateAdDescription(req.Description); err != nil {
		errs.Description = err.Error()
	}
	if err := validator.ValidateAdPrice(req.Price); err != nil {
		errs.Price = err.Error()
	}
	if err := validator.ValidateAdStatus(req.Status); err != nil {
		errs.Status = err.Error()
	}
	if err := validator.ValidateAdLocation(req.Location); err != nil {
		errs.Location = err.Error()
	}
	if err := validator.ValidateAdCoords(req.Lat, req.Lon); err != nil {
		errs.Coords = err.Error()
	}
	return &errs
}

func validateUpdateAdRequest(req *dto.UpdateAdRequest) *dto.ValidationErrors {
	errs := dto.ValidationErrors{}
	if req.CategoryID != nil {
		if err := validator.ValidateCategoryID(*req.CategoryID); err != nil {
			errs.CategoryID = err.Error()
		}
	}
	if req.Title != nil {
		if err := validator.ValidateAdTitle(*req.Title); err != nil {
			errs.Title = err.Error()
		}
	}
	if req.Description != nil {
		if err := validator.ValidateAdDescription(*req.Description); err != nil {
			errs.Description = err.Error()
		}
	}
	if req.Price != nil {
		if err := validator.ValidateAdPrice(*req.Price); err != nil {
			errs.Price = err.Error()
		}
	}
	if req.Status != nil {
		if err := validator.ValidateAdStatus(*req.Status); err != nil {
			errs.Status = err.Error()
		}
	}
	if req.Location != nil {
		if err := validator.ValidateAdLocation(*req.Location); err != nil {
			errs.Location = err.Error()
		}
	}
	if req.Lat != nil || req.Lon != nil {
		if err := validator.ValidateAdCoords(req.Lat, req.Lon); err != nil {
			errs.Coords = err.Error()
		}
	}
	return &errs
}
