package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/sanitizer"
	ad "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/ad"
)

const (
	opHandleAdminDeleteAd      = "handlers.HandleAdminDeleteAd"
	opHandleApproveAd          = "handlers.HandleApproveAd"
	opHandleRejectAd           = "handlers.HandleRejectAd"
	opHandleGetModerationQueue = "handlers.HandleGetModerationQueue"
	opHandleSetModerationFlag  = "handlers.HandleSetModerationFlag"
)

type moderationFlagBody struct {
	Enabled bool `json:"enabled"`
}

type rejectBody struct {
	Reason string `json:"reason"`
}

func adminIDFromCtx(r *http.Request) (int64, bool) {
	id, ok := r.Context().Value(middleware.UserIDKey).(int64)
	return id, ok && id != 0
}

// HandleAdminDeleteAd — DELETE /api/v1/ads/{id}/admin.
// @Summary Удалить объявление (админ)
// @Description Жёсткое удаление от имени админа: товар становится недоступен и
// продавцу приходит системное сообщение.
// @Tags admin
// @Produce json
// @Param id path int true "ID объявления"
// @Success 200 {object} map[string]string "status=deleted"
// @Failure 400 {object} dto.ErrorResponse "invalid ad id"
// @Failure 401 {object} dto.ErrorResponse "unauthorized / forbidden role"
// @Failure 404 {object} dto.ErrorResponse "ad not found"
// @Failure 500 {object} dto.ErrorResponse "internal error"
// @Security CookieAuth
// @Router /ads/{id}/admin [delete]
func (h *AdsHandlers) HandleAdminDeleteAd(w http.ResponseWriter, r *http.Request) {
	adminID, ok := adminIDFromCtx(r)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	if err := h.services.Ads.AdminDeleteAd(r.Context(), id, adminID); err != nil {
		if errors.Is(err, ad.ErrAdNotFound) {
			responser.RespondWithError(w, http.StatusNotFound, ErrAdNotFound)
			return
		}
		h.log.ErrorContext(r.Context(), "admin delete failed",
			slog.String("op", opHandleAdminDeleteAd),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{statusKey: "deleted"})
}

// HandleGetModerationFlag — GET /api/v1/admin/moderation/settings.
// @Summary Текущее состояние модерации
// @Tags admin
// @Produce json
// @Success 200 {object} map[string]bool
// @Security CookieAuth
// @Router /admin/moderation/settings [get]
func (h *AdsHandlers) HandleGetModerationFlag(w http.ResponseWriter, r *http.Request) {
	enabled := h.services.Ads.IsModerationEnabled(r.Context())
	responser.RespondWithJSON(w, http.StatusOK, map[string]bool{"enabled": enabled})
}

// HandleSetModerationFlag — PUT /api/v1/admin/moderation/settings.
// @Summary Включить/выключить модерацию
// @Tags admin
// @Accept json
// @Produce json
// @Param body body moderationFlagBody true "флаг"
// @Success 200 {object} map[string]bool
// @Security CookieAuth
// @Router /admin/moderation/settings [put]
func (h *AdsHandlers) HandleSetModerationFlag(w http.ResponseWriter, r *http.Request) {
	adminID, ok := adminIDFromCtx(r)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	var body moderationFlagBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	if err := h.services.Ads.SetModerationEnabled(r.Context(), body.Enabled, adminID); err != nil {
		h.log.ErrorContext(r.Context(), "set moderation failed",
			slog.String("op", opHandleSetModerationFlag),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]bool{"enabled": body.Enabled})
}

// HandleGetModerationQueue — GET /api/v1/admin/moderation/queue.
// @Summary Список объявлений на модерации
// @Tags admin
// @Produce json
// @Success 200 {object} map[string][]models.Ad
// @Security CookieAuth
// @Router /admin/moderation/queue [get]
func (h *AdsHandlers) HandleGetModerationQueue(w http.ResponseWriter, r *http.Request) {
	ads, err := h.services.Ads.GetModerationQueue(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "moderation queue failed",
			slog.String("op", opHandleGetModerationQueue),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, map[string]interface{}{adsKey: ads})
}

// HandleApproveAd — POST /api/v1/admin/moderation/ads/{id}/approve.
// @Summary Одобрить объявление
// @Tags admin
// @Produce json
// @Param id path int true "ID объявления"
// @Success 200 {object} map[string]string
// @Security CookieAuth
// @Router /admin/moderation/ads/{id}/approve [post]
func (h *AdsHandlers) HandleApproveAd(w http.ResponseWriter, r *http.Request) {
	adminID, ok := adminIDFromCtx(r)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	if err := h.services.Ads.ApproveAd(r.Context(), id, adminID); err != nil {
		if errors.Is(err, ad.ErrAdNotFound) {
			responser.RespondWithError(w, http.StatusNotFound, ErrAdNotFound)
			return
		}
		h.log.ErrorContext(r.Context(), "approve failed",
			slog.String("op", opHandleApproveAd),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, map[string]string{statusKey: "active"})
}

// HandleRejectAd — POST /api/v1/admin/moderation/ads/{id}/reject.
// @Summary Отклонить объявление
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "ID объявления"
// @Param body body rejectBody false "причина отказа"
// @Success 200 {object} map[string]string
// @Security CookieAuth
// @Router /admin/moderation/ads/{id}/reject [post]
func (h *AdsHandlers) HandleRejectAd(w http.ResponseWriter, r *http.Request) {
	adminID, ok := adminIDFromCtx(r)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidAdID)
		return
	}

	var body rejectBody
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
			return
		}
	}
	body.Reason = sanitizer.StripHTML(body.Reason)

	if err := h.services.Ads.RejectAd(r.Context(), id, adminID, body.Reason); err != nil {
		if errors.Is(err, ad.ErrAdNotFound) {
			responser.RespondWithError(w, http.StatusNotFound, ErrAdNotFound)
			return
		}
		h.log.ErrorContext(r.Context(), "reject failed",
			slog.String("op", opHandleRejectAd),
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}
	responser.RespondWithJSON(w, http.StatusOK, map[string]string{statusKey: "rejected"})
}
