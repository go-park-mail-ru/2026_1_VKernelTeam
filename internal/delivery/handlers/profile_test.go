package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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
