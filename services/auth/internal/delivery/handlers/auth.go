package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/usecase/auth"
	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/sanitizer"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/validator"
)

func (h *AuthHandlers) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	req.Name = sanitizer.StripHTML(req.Name)

	validationErrors := dto.ValidationErrors{}

	cleanEmail, err := validator.ValidateEmail(req.Email)
	if err != nil {
		validationErrors.Email = err.Error()
	}

	cleanName, err := validator.ValidateName(req.Name)
	if err != nil {
		validationErrors.Name = err.Error()
	}

	if err := validator.ValidatePassword(req.Password); err != nil {
		validationErrors.Password = err.Error()
	}

	if validationErrors.Email != "" || validationErrors.Password != "" || validationErrors.Name != "" {
		responser.RespondWithJSON(w, http.StatusBadRequest, validationErrors)
		return
	}

	userID, err := h.auth.RegisterNewUser(r.Context(), cleanEmail, req.Password, cleanName)
	if err != nil {
		if errors.Is(err, auth.ErrUserAlreadyExists) {
			responser.RespondWithError(w, http.StatusBadRequest, ErrUserAlreadyExists)
			return
		}

		h.log.ErrorContext(r.Context(), "failed to register user",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToRegisterUser)
		return
	}

	h.log.InfoContext(r.Context(), "user registered successfully",
		slog.Int64("user_id", userID),
	)

	token, refreshToken, user, err := h.auth.Login(r.Context(), cleanEmail, req.Password)
	if err != nil {
		h.log.ErrorContext(r.Context(), "auto-login failed after registration",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrAutoLoginFailed)
		return
	}

	csrfToken := middleware.GenerateCSRFToken()
	h.setAuthCookie(w, token, csrfToken)
	h.setRefreshCookie(w, refreshToken)
	h.respondWithUser(w, user)
}

func (h *AuthHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("token"); err == nil && cookie.Value != "" {
		h.handleTokenLogin(w, r, cookie.Value)
		return
	}

	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responser.RespondWithError(w, http.StatusBadRequest, ErrInvalidRequestBody)
		return
	}

	if req.Email == "" || req.Password == "" {
		validationErrors := dto.ValidationErrors{}
		if req.Email == "" {
			validationErrors.Email = "email is required"
		}
		if req.Password == "" {
			validationErrors.Password = "password is required"
		}
		responser.RespondWithJSON(w, http.StatusUnauthorized, validationErrors)
		return
	}

	h.handleCredentialsLogin(w, r, req.Email, req.Password)
}

func (h *AuthHandlers) handleCredentialsLogin(w http.ResponseWriter, r *http.Request, email, password string) {
	validationErrors := dto.ValidationErrors{}

	cleanEmail, err := validator.ValidateEmail(email)
	if err != nil {
		validationErrors.Email = err.Error()
	}

	if err := validator.ValidatePassword(password); err != nil {
		validationErrors.Password = err.Error()
	}

	if validationErrors.Email != "" || validationErrors.Password != "" {
		responser.RespondWithJSON(w, http.StatusUnauthorized, validationErrors)
		return
	}

	token, refreshToken, user, err := h.auth.Login(r.Context(), cleanEmail, password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			responser.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}

		h.log.ErrorContext(r.Context(), "failed to login",
			slog.String("error", err.Error()),
		)
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToLogin)
		return
	}

	csrfToken := middleware.GenerateCSRFToken()
	h.setAuthCookie(w, token, csrfToken)
	h.setRefreshCookie(w, refreshToken)
	h.respondWithUser(w, user)
}

func (h *AuthHandlers) handleTokenLogin(w http.ResponseWriter, r *http.Request, tokenString string) {
	user, err := h.auth.ValidateTokenAndGetUser(r.Context(), tokenString)
	if err != nil {
		responser.RespondWithError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	csrfToken := middleware.GenerateCSRFToken()
	h.setAuthCookie(w, tokenString, csrfToken)
	h.respondWithUser(w, user)
}

func (h *AuthHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	jti, ok := r.Context().Value(middleware.JtiKey).(string)
	if !ok {
		responser.RespondWithError(w, http.StatusUnauthorized, ErrInternalError)
		return
	}

	var refreshToken string
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	if err := h.auth.Logout(r.Context(), jti, time.Now().Add(h.tokenTTL), refreshToken); err != nil {
		responser.RespondWithError(w, http.StatusInternalServerError, ErrFailedToLogout)
		return
	}

	// Очищаем все cookie
	for _, name := range []string{"token", "csrf_token", "refresh_token"} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: name != "csrf_token",
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
		})
	}

	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandlers) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		responser.RespondWithError(w, http.StatusUnauthorized, "refresh token required")
		return
	}

	newAccess, newRefresh, err := h.auth.Refresh(r.Context(), cookie.Value)
	if err != nil {
		responser.RespondWithError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	csrfToken := middleware.GenerateCSRFToken()
	h.setAuthCookie(w, newAccess, csrfToken)
	h.setRefreshCookie(w, newRefresh)
	responser.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
