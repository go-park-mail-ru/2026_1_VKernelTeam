package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/sanitizer"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/validator"
)

func (h *AuthHandlers) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	user, err := h.auth.GetProfile(r.Context(), userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get profile",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, user)
}

func (h *AuthHandlers) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	var req dto.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	req.Name = sanitizer.StripHTML(req.Name)

	cleanName, err := validator.ValidateName(req.Name)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	updatedUser, err := h.auth.UpdateProfile(r.Context(), userID, cleanName)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to update profile",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, updatedUser)
}

func (h *AuthHandlers) HandleGetPublicProfile(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidUserID)
		return
	}

	user, err := h.auth.GetProfile(r.Context(), userID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get public profile",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	response := dto.PublicUserResponse{
		ID:           user.ID,
		Name:         user.Name,
		AvatarPath:   user.AvatarPath,
		Rating:       user.Rating,
		ReviewsCount: user.ReviewsCount,
		AdsCount:     user.AdsCount,
		CreatedAt:    user.CreatedAt,
	}

	responser.RespondWithJSON(w, http.StatusOK, response)
}

func (h *AuthHandlers) HandleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrFileTooBig)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrFailedToGetFile)
		return
	}
	defer file.Close()

	updatedUser, err := h.auth.UpdateAvatar(r.Context(), userID, file, header.Filename)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to update avatar",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrInternalError)
		return
	}

	responser.RespondWithJSON(w, http.StatusOK, updatedUser)
}
