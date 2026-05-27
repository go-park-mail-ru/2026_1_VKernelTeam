package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
)

const (
	opHandleRecordView = "handlers.HandleRecordView"

	// ErrInvalidDeviceID — сообщение об ошибке при отсутствии или невалидном заголовке X-Device-ID.
	ErrInvalidDeviceID = "invalid or missing X-Device-ID header (must be UUID v4)"
)

// HandleRecordView обрабатывает запрос на фиксацию просмотра объявления.
// @Summary Зафиксировать просмотр объявления
// @Description Записывает просмотр с дедупликацией и возвращает актуальный счётчик
// @Tags ads
// @Produce json
// @Param id path int true "ID объявления"
// @Param X-Device-ID header string true "UUID v4 идентификатор устройства"
// @Success 200 {object} map[string]int64 "views_count"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id / invalid X-Device-ID"
// @Failure 404 {object} dto.ErrorResponse "ad not found"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Router /ads/{id}/view [post]
func (h *ViewsHandlers) HandleRecordView(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	productID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	// Валидируем X-Device-ID (строгая валидация UUID v4)
	deviceIDStr := r.Header.Get("X-Device-ID")
	deviceUUID, err := uuid.Parse(deviceIDStr)
	if err != nil || deviceUUID.Version() != 4 {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidDeviceID)
		return
	}

	// Извлекаем user_id из контекста (может отсутствовать — анонимный пользователь)
	var userID *int64
	if uid, ok := r.Context().Value(middleware.UserIDKey).(int64); ok {
		userID = &uid
	}

	viewsCount, err := h.viewsSvc.RecordView(r.Context(), productID, userID, deviceUUID.String())
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to record view",
			slog.String("op", opHandleRecordView),
			slog.Int64("product_id", productID),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]int64{
		"views_count": viewsCount,
	})
}
