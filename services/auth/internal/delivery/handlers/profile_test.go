package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/models"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHandleGetProfile(t *testing.T) {
	authH, mockAuth := setupHandlers(t)

	t.Run("Success", func(t *testing.T) {
		userID := int64(42)
		expectedUser := models.User{
			ID:    int64(userID),
			Email: "test@mail.ru",
			Name:  "Ivan",
		}

		req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "/profile", nil)
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
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/profile", nil)
		rr := httptest.NewRecorder()

		authH.HandleGetProfile(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("InternalError", func(t *testing.T) {
		userID := int64(1)

		req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "/profile", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			GetProfile(gomock.Any(), userID).
			Return(models.User{}, errors.New(ErrInternalError))

		authH.HandleGetProfile(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandleUpdateProfile(t *testing.T) {
	authH, mockAuth := setupHandlers(t)

	t.Run("Success", func(t *testing.T) {
		userID := int64(42)
		newName := "New Ivan Name"
		reqBody := dto.UpdateProfileRequest{
			Name: newName,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPatch, "/profile", bytes.NewBuffer(bodyBytes))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		updatedUser := models.User{
			ID:   userID,
			Name: newName,
		}

		mockAuth.EXPECT().
			UpdateProfile(gomock.Any(), userID, newName).
			Return(updatedUser, nil)

		authH.HandleUpdateProfile(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var actualUser models.User
		err := json.Unmarshal(rr.Body.Bytes(), &actualUser)
		assert.NoError(t, err)
		assert.Equal(t, updatedUser, actualUser)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		userID := int64(42)

		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPatch, "/profile", bytes.NewBuffer([]byte("{invalid}")))
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

		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPatch, "/profile", bytes.NewBuffer(bodyBytes))
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

func TestHandleGetPublicProfile(t *testing.T) {
	authH, mockAuth := setupHandlers(t)

	t.Run("Success", func(t *testing.T) {
		userID := int64(100)
		idStr := "100"
		user := models.User{
			ID:     userID,
			Name:   "Public User",
			Rating: 4.8,
		}

		req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/"+idStr, nil)
		req.SetPathValue("id", idStr)

		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			GetProfile(gomock.Any(), userID).
			Return(user, nil)

		authH.HandleGetPublicProfile(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.PublicUserResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, resp.ID)
		assert.Equal(t, user.Name, resp.Name)
	})

	t.Run("InvalidID", func(t *testing.T) {
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/not-an-int", nil)
		req.SetPathValue("id", "not-an-int")
		rr := httptest.NewRecorder()

		authH.HandleGetPublicProfile(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("UserNotFound", func(t *testing.T) {
		userID := int64(404)
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/404", nil)
		req.SetPathValue("id", "404")
		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			GetProfile(gomock.Any(), userID).
			Return(models.User{}, errors.New(ErrInvalidRequestBody))

		authH.HandleGetPublicProfile(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandleUploadAvatar(t *testing.T) {
	authH, mockAuth := setupHandlers(t)

	t.Run("Success", func(t *testing.T) {
		userID := int64(42)
		filename := "test.png"
		fileContent := []byte("\x89PNG\r\n\x1a\n")

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("avatar", filename)
		assert.NoError(t, err)
		_, _ = part.Write(fileContent)
		_ = writer.Close()

		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, "/profile/avatar", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		mockAuth.EXPECT().
			UpdateAvatar(gomock.Any(), userID, gomock.Any(), filename).
			Return(models.User{ID: userID, AvatarPath: "/new/path.png"}, nil)

		authH.HandleUploadAvatar(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp models.User
		err = json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "/new/path.png", resp.AvatarPath)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, "/profile/avatar", nil)

		rr := httptest.NewRecorder()

		authH.HandleUploadAvatar(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("NoFile", func(t *testing.T) {
		userID := int64(42)
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.Close()

		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, "/profile/avatar", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		authH.HandleUploadAvatar(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
