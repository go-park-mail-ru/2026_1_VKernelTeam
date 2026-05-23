package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	middleware "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/delivery/handlers/mocks"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	reviewrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/review"
	reviewuc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/review"
)

const (
	testContentGreat    = "great"
	testContentEdit     = "edit"
	testCaseInternalErr = "internal"
)

func setupReviewHandlers(t *testing.T) (*ReviewHandlers, *mocks.MockReviewProvider) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	m := mocks.NewMockReviewProvider(ctrl)
	return NewReviewHandlers(logger, m), m
}

func reqWithUserAndPath(method, target, idValue string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body == nil {
		r = httptest.NewRequestWithContext(context.Background(), method, target, nil)
	} else {
		r = httptest.NewRequestWithContext(context.Background(), method, target, bytes.NewBuffer(body))
	}
	if idValue != "" {
		r.SetPathValue("id", idValue)
	}
	if userID > 0 {
		ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
		r = r.WithContext(ctx)
	}
	return r
}

func decodeError(t *testing.T, body *bytes.Buffer) string {
	t.Helper()
	var m map[string]string
	require.NoError(t, json.NewDecoder(body).Decode(&m))
	return m["error"]
}

func TestHandleCreateReview_Happy(t *testing.T) {
	h, m := setupReviewHandlers(t)
	req := dto.CreateReviewRequest{ReceiverID: 2, ProductID: 38, Rating: 5, Content: testContentGreat}
	body, _ := json.Marshal(req)

	m.EXPECT().CreateReview(gomock.Any(), int64(1), req).
		Return(dto.ReviewResponse{ID: 101, ReceiverID: 2, Rating: 5, Content: testContentGreat}, nil)

	rr := httptest.NewRecorder()
	h.HandleCreateReview(rr, reqWithUserAndPath(http.MethodPost, "/api/v1/reviews", "", body, 1))

	require.Equal(t, http.StatusCreated, rr.Code)
	var resp dto.CreateReviewResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, int64(101), resp.Review.ID)
}

func TestHandleCreateReview_NoAuth(t *testing.T) {
	h, _ := setupReviewHandlers(t)
	body, _ := json.Marshal(dto.CreateReviewRequest{ReceiverID: 2, ProductID: 38, Rating: 5, Content: "hello"})
	rr := httptest.NewRecorder()
	h.HandleCreateReview(rr, reqWithUserAndPath(http.MethodPost, "/api/v1/reviews", "", body, 0))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleCreateReview_BadJSON(t *testing.T) {
	h, _ := setupReviewHandlers(t)
	rr := httptest.NewRecorder()
	h.HandleCreateReview(rr, reqWithUserAndPath(http.MethodPost, "/api/v1/reviews", "", []byte("{bad"), 1))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleCreateReview_ErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		ucErr      error
		wantStatus int
		wantText   string
	}{
		{"invalid_rating", reviewuc.ErrInvalidRating, http.StatusBadRequest, ErrRatingRange},
		{"invalid_content", reviewuc.ErrInvalidContent, http.StatusBadRequest, ErrContentRange},
		{"self_review", reviewuc.ErrSelfReview, http.StatusBadRequest, ErrCannotReviewSelf},
		{"not_purchased", reviewuc.ErrNotPurchased, http.StatusBadRequest, ErrNotPurchased},
		{"seller_mismatch", reviewuc.ErrSellerMismatch, http.StatusBadRequest, ErrSellerMismatch},
		{"already_exists", reviewrepo.ErrReviewAlreadyExists, http.StatusBadRequest, ErrReviewExists},
		{"product_not_found", reviewrepo.ErrProductNotFound, http.StatusBadRequest, ErrProductNotFound},
		{testCaseInternalErr, errors.New("db down"), http.StatusInternalServerError, ErrInternalError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, m := setupReviewHandlers(t)
			req := dto.CreateReviewRequest{ReceiverID: 2, ProductID: 38, Rating: 5, Content: testContentGreat}
			body, _ := json.Marshal(req)

			m.EXPECT().CreateReview(gomock.Any(), int64(1), req).
				Return(dto.ReviewResponse{}, tc.ucErr)

			rr := httptest.NewRecorder()
			h.HandleCreateReview(rr, reqWithUserAndPath(http.MethodPost, "/api/v1/reviews", "", body, 1))

			assert.Equal(t, tc.wantStatus, rr.Code)
			assert.Equal(t, tc.wantText, decodeError(t, rr.Body))
		})
	}
}

func TestHandleUpdateReview_Happy(t *testing.T) {
	h, m := setupReviewHandlers(t)
	req := dto.UpdateReviewRequest{Rating: 4, Content: testContentEdit}
	body, _ := json.Marshal(req)
	m.EXPECT().UpdateReview(gomock.Any(), int64(1), int64(101), req).
		Return(dto.ReviewResponse{ID: 101, Rating: 4, Content: testContentEdit}, nil)

	rr := httptest.NewRecorder()
	h.HandleUpdateReview(rr, reqWithUserAndPath(http.MethodPut, "/api/v1/reviews/101", "101", body, 1))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp dto.ReviewResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 4, resp.Rating)
}

func TestHandleUpdateReview_NoAuth(t *testing.T) {
	h, _ := setupReviewHandlers(t)
	body, _ := json.Marshal(dto.UpdateReviewRequest{Rating: 4, Content: testContentEdit})
	rr := httptest.NewRecorder()
	h.HandleUpdateReview(rr, reqWithUserAndPath(http.MethodPut, "/api/v1/reviews/101", "101", body, 0))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleUpdateReview_BadID(t *testing.T) {
	h, _ := setupReviewHandlers(t)
	body, _ := json.Marshal(dto.UpdateReviewRequest{Rating: 4, Content: testContentEdit})
	rr := httptest.NewRecorder()
	h.HandleUpdateReview(rr, reqWithUserAndPath(http.MethodPut, "/api/v1/reviews/abc", "abc", body, 1))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, ErrInvalidReviewID, decodeError(t, rr.Body))
}

func TestHandleUpdateReview_BadJSON(t *testing.T) {
	h, _ := setupReviewHandlers(t)
	rr := httptest.NewRecorder()
	h.HandleUpdateReview(rr, reqWithUserAndPath(http.MethodPut, "/api/v1/reviews/101", "101", []byte("{bad"), 1))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleUpdateReview_ErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		ucErr      error
		wantStatus int
		wantText   string
	}{
		{"not_author", reviewuc.ErrForbiddenReviewEdit, http.StatusBadRequest, ErrNotReviewAuthor},
		{"not_found", reviewrepo.ErrReviewNotFound, http.StatusBadRequest, ErrReviewNotFound},
		{testCaseInternalErr, errors.New("db down"), http.StatusInternalServerError, ErrInternalError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, m := setupReviewHandlers(t)
			req := dto.UpdateReviewRequest{Rating: 4, Content: testContentEdit}
			body, _ := json.Marshal(req)
			m.EXPECT().UpdateReview(gomock.Any(), int64(1), int64(101), req).
				Return(dto.ReviewResponse{}, tc.ucErr)

			rr := httptest.NewRecorder()
			h.HandleUpdateReview(rr, reqWithUserAndPath(http.MethodPut, "/api/v1/reviews/101", "101", body, 1))

			assert.Equal(t, tc.wantStatus, rr.Code)
			assert.Equal(t, tc.wantText, decodeError(t, rr.Body))
		})
	}
}

func TestHandleDeleteReview_Happy(t *testing.T) {
	h, m := setupReviewHandlers(t)
	m.EXPECT().DeleteReview(gomock.Any(), int64(1), int64(101)).Return(nil)

	rr := httptest.NewRecorder()
	h.HandleDeleteReview(rr, reqWithUserAndPath(http.MethodDelete, "/api/v1/reviews/101", "101", nil, 1))
	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestHandleDeleteReview_NoAuth(t *testing.T) {
	h, _ := setupReviewHandlers(t)
	rr := httptest.NewRecorder()
	h.HandleDeleteReview(rr, reqWithUserAndPath(http.MethodDelete, "/api/v1/reviews/101", "101", nil, 0))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleDeleteReview_BadID(t *testing.T) {
	h, _ := setupReviewHandlers(t)
	rr := httptest.NewRecorder()
	h.HandleDeleteReview(rr, reqWithUserAndPath(http.MethodDelete, "/api/v1/reviews/abc", "abc", nil, 1))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, ErrInvalidReviewID, decodeError(t, rr.Body))
}

func TestHandleDeleteReview_ErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		ucErr      error
		wantStatus int
		wantText   string
	}{
		{"not_author", reviewuc.ErrForbiddenReviewEdit, http.StatusBadRequest, ErrNotReviewAuthor},
		{"not_found", reviewrepo.ErrReviewNotFound, http.StatusBadRequest, ErrReviewNotFound},
		{testCaseInternalErr, errors.New("db down"), http.StatusInternalServerError, ErrInternalError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, m := setupReviewHandlers(t)
			m.EXPECT().DeleteReview(gomock.Any(), int64(1), int64(101)).Return(tc.ucErr)

			rr := httptest.NewRecorder()
			h.HandleDeleteReview(rr, reqWithUserAndPath(http.MethodDelete, "/api/v1/reviews/101", "101", nil, 1))

			assert.Equal(t, tc.wantStatus, rr.Code)
			assert.Equal(t, tc.wantText, decodeError(t, rr.Body))
		})
	}
}

func TestHandleListUserReviews_Happy(t *testing.T) {
	h, m := setupReviewHandlers(t)
	cur := int64(50)
	m.EXPECT().GetUserReviews(gomock.Any(), int64(2), &cur, 5).
		Return(dto.ReviewListResponse{Reviews: []dto.ReviewResponse{{ID: 49}}, NextCursor: nil}, nil)

	rr := httptest.NewRecorder()
	h.HandleListUserReviews(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/users/2/reviews?cursor=50&limit=5", "2", nil, 0))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp dto.ReviewListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Len(t, resp.Reviews, 1)
}

func TestHandleListUserReviews_NoCursor(t *testing.T) {
	h, m := setupReviewHandlers(t)
	m.EXPECT().GetUserReviews(gomock.Any(), int64(2), gomock.Nil(), 0).
		Return(dto.ReviewListResponse{Reviews: []dto.ReviewResponse{}}, nil)

	rr := httptest.NewRecorder()
	h.HandleListUserReviews(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/users/2/reviews", "2", nil, 0))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleListUserReviews_LimitPassedThrough(t *testing.T) {
	h, m := setupReviewHandlers(t)
	m.EXPECT().GetUserReviews(gomock.Any(), int64(2), gomock.Nil(), 999).
		Return(dto.ReviewListResponse{Reviews: []dto.ReviewResponse{}}, nil)

	rr := httptest.NewRecorder()
	h.HandleListUserReviews(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/users/2/reviews?limit=999", "2", nil, 0))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleListUserReviews_BadID(t *testing.T) {
	h, _ := setupReviewHandlers(t)
	rr := httptest.NewRecorder()
	h.HandleListUserReviews(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/users/abc/reviews", "abc", nil, 0))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, ErrInvalidUserID, decodeError(t, rr.Body))
}

func TestHandleListUserReviews_InvalidCursorTreatedAsNil(t *testing.T) {
	// parseCursor: non-empty но не-парсящееся значение -> nil cursor.
	h, m := setupReviewHandlers(t)
	m.EXPECT().GetUserReviews(gomock.Any(), int64(2), gomock.Nil(), 0).
		Return(dto.ReviewListResponse{Reviews: []dto.ReviewResponse{}}, nil)

	rr := httptest.NewRecorder()
	h.HandleListUserReviews(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/users/2/reviews?cursor=notanint", "2", nil, 0))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleListUserReviews_Internal(t *testing.T) {
	h, m := setupReviewHandlers(t)
	m.EXPECT().GetUserReviews(gomock.Any(), int64(2), gomock.Nil(), 0).
		Return(dto.ReviewListResponse{}, errors.New("db down"))

	rr := httptest.NewRecorder()
	h.HandleListUserReviews(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/users/2/reviews", "2", nil, 0))
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandleUserReviewsSummary_Happy(t *testing.T) {
	h, m := setupReviewHandlers(t)
	m.EXPECT().GetUserReviewsSummary(gomock.Any(), int64(2)).
		Return(dto.ReviewSummaryResponse{
			Average:      4.5,
			Total:        4,
			Distribution: map[int]int{5: 3, 4: 1},
		}, nil)

	rr := httptest.NewRecorder()
	h.HandleUserReviewsSummary(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/users/2/reviews/summary", "2", nil, 0))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp dto.ReviewSummaryResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 4, resp.Total)
}

func TestHandleUserReviewsSummary_Empty(t *testing.T) {
	h, m := setupReviewHandlers(t)
	m.EXPECT().GetUserReviewsSummary(gomock.Any(), int64(2)).
		Return(dto.ReviewSummaryResponse{Distribution: map[int]int{}}, nil)

	rr := httptest.NewRecorder()
	h.HandleUserReviewsSummary(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/users/2/reviews/summary", "2", nil, 0))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp dto.ReviewSummaryResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 0, resp.Total)
	assert.Equal(t, float64(0), resp.Average)
}

func TestHandleUserReviewsSummary_BadID(t *testing.T) {
	h, _ := setupReviewHandlers(t)
	rr := httptest.NewRecorder()
	h.HandleUserReviewsSummary(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/users/abc/reviews/summary", "abc", nil, 0))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleUserReviewsSummary_Internal(t *testing.T) {
	h, m := setupReviewHandlers(t)
	m.EXPECT().GetUserReviewsSummary(gomock.Any(), int64(2)).
		Return(dto.ReviewSummaryResponse{}, errors.New("db down"))

	rr := httptest.NewRecorder()
	h.HandleUserReviewsSummary(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/users/2/reviews/summary", "2", nil, 0))
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandleListMyReviews_Happy(t *testing.T) {
	h, m := setupReviewHandlers(t)
	m.EXPECT().GetMyReviews(gomock.Any(), int64(1), gomock.Nil(), 0).
		Return(dto.ReviewListResponse{Reviews: []dto.ReviewResponse{{ID: 5}}}, nil)

	rr := httptest.NewRecorder()
	h.HandleListMyReviews(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/profile/reviews", "", nil, 1))

	require.Equal(t, http.StatusOK, rr.Code)
	var resp dto.ReviewListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Len(t, resp.Reviews, 1)
}

func TestHandleListMyReviews_NoAuth(t *testing.T) {
	h, _ := setupReviewHandlers(t)
	rr := httptest.NewRecorder()
	h.HandleListMyReviews(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/profile/reviews", "", nil, 0))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestHandleListMyReviews_Internal(t *testing.T) {
	h, m := setupReviewHandlers(t)
	m.EXPECT().GetMyReviews(gomock.Any(), int64(1), gomock.Nil(), 0).
		Return(dto.ReviewListResponse{}, errors.New("db down"))

	rr := httptest.NewRecorder()
	h.HandleListMyReviews(rr, reqWithUserAndPath(http.MethodGet, "/api/v1/profile/reviews", "", nil, 1))
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
