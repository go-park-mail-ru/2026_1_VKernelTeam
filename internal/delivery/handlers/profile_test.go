package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHandleGetProfile(t *testing.T) {
	authH, _, mockAuth, _ := setupHandlers(t)

	t.Run("Success", func(t *testing.T) {
		userID := int64(42)
		expectedUser := models.User{
			ID:    int64(userID),
			Email: "test@mail.ru",
			Name:  "Ivan",
		}

		// кладем ID в контекст
		req, _ := http.NewRequest(http.MethodGet, "/profile", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			GetProfile(gomock.Any(), userID).
			Return(expectedUser, nil)

		authH.HandleGetProfile(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var actualUser models.User
		err := json.Unmarshal(rr.Body.Bytes(), &actualUser)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, actualUser)
	})

	t.Run("NoUserIDInContext", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/profile", nil)
		rr := httptest.NewRecorder()

		authH.HandleGetProfile(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("InternalError", func(t *testing.T) {
		userID := int64(1)

		// кладем ID в контекст
		req, _ := http.NewRequest(http.MethodGet, "/profile", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			GetProfile(ctx, userID).
			Return(models.User{}, errors.New(ErrInternalError))

		authH.HandleGetProfile(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandleUpdateProfile(t *testing.T) {
	authH, _, mockAuth, _ := setupHandlers(t)

	t.Run("Success", func(t *testing.T) {
		userID := int64(42)
		newName := "New Ivan Name"
		reqBody := dto.UpdateProfileRequest{
			Name: newName,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		// кладем ID в контекст
		req, _ := http.NewRequest(http.MethodPatch, "/profile", bytes.NewBuffer(bodyBytes))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		updatedUser := models.User{
			ID:   userID,
			Name: newName,
		}

		mockAuth.EXPECT().
			GetProfile(gomock.Any(), userID).
			Return(updatedUser, nil)

		authH.HandleGetProfile(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var actualUser models.User
		err := json.Unmarshal(rr.Body.Bytes(), &actualUser)
		assert.NoError(t, err)
		assert.Equal(t, updatedUser, actualUser)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		userID := int64(42)

		// кладем ID в контекст
		req, _ := http.NewRequest(http.MethodPatch, "/profile", bytes.NewBuffer([]byte("{invalid}")))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		authH.HandleUpdateProfile(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("ServiceError", func(t *testing.T) {
		userID := int64(42)
		reqBody := dto.UpdateProfileRequest{
			Name: "Name",
		}

		bodyBytes, _ := json.Marshal(reqBody)

		// кладем ID в контекст
		req, _ := http.NewRequest(http.MethodPatch, "/profile", bytes.NewBuffer(bodyBytes))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			UpdateProfile(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(models.User{}, errors.New(ErrInternalError))

		authH.HandleUpdateProfile(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
