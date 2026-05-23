package review

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
	reviewrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/review"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/review/mocks"
)

const (
	testContentGood   = "great seller"
	testContentEdit   = "edit"
	testContentEdited = "edited"
)

func setupService(t *testing.T) (*ReviewService, *mocks.MockReviewStorage) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	m := mocks.NewMockReviewStorage(ctrl)
	return NewService(m, slog.Default()), m
}

func validCreateReq() dto.CreateReviewRequest {
	return dto.CreateReviewRequest{
		ReceiverID: 2,
		ProductID:  38,
		Rating:     5,
		Content:    testContentGood,
	}
}

func TestCreateReview_InvalidRating(t *testing.T) {
	cases := []int{0, -1, 6, 99}
	for _, r := range cases {
		t.Run("rating", func(t *testing.T) {
			s, _ := setupService(t)
			req := validCreateReq()
			req.Rating = r
			_, err := s.CreateReview(context.Background(), 1, req)
			assert.ErrorIs(t, err, ErrInvalidRating)
		})
	}
}

func TestCreateReview_InvalidContent(t *testing.T) {
	cases := []string{"", "abcd", strings.Repeat("a", 2001)}
	for _, c := range cases {
		t.Run("content", func(t *testing.T) {
			s, _ := setupService(t)
			req := validCreateReq()
			req.Content = c
			_, err := s.CreateReview(context.Background(), 1, req)
			assert.ErrorIs(t, err, ErrInvalidContent)
		})
	}
}

func TestCreateReview_SelfReview(t *testing.T) {
	s, _ := setupService(t)
	req := validCreateReq()
	req.ReceiverID = 1
	_, err := s.CreateReview(context.Background(), 1, req)
	assert.ErrorIs(t, err, ErrSelfReview)
}

func TestCreateReview_ProductNotFound(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetProductSellerID(gomock.Any(), int64(38)).
		Return(int64(0), reviewrepo.ErrProductNotFound)

	_, err := s.CreateReview(context.Background(), 1, validCreateReq())
	assert.ErrorIs(t, err, reviewrepo.ErrProductNotFound)
}

func TestCreateReview_GetSellerInternalError(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetProductSellerID(gomock.Any(), int64(38)).
		Return(int64(0), errors.New("db down"))

	_, err := s.CreateReview(context.Background(), 1, validCreateReq())
	assert.Error(t, err)
	assert.NotErrorIs(t, err, reviewrepo.ErrProductNotFound)
}

func TestCreateReview_ExistsPurchaseError(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetProductSellerID(gomock.Any(), int64(38)).Return(int64(2), nil)
	m.EXPECT().ExistsPurchaseRequest(gomock.Any(), int64(1), int64(2), int64(38)).
		Return(false, errors.New("db down"))

	_, err := s.CreateReview(context.Background(), 1, validCreateReq())
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotPurchased)
}

func TestCreateReview_SellerMismatch(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetProductSellerID(gomock.Any(), int64(38)).
		Return(int64(7), nil)

	_, err := s.CreateReview(context.Background(), 1, validCreateReq())
	assert.ErrorIs(t, err, ErrSellerMismatch)
}

func TestCreateReview_NotPurchased(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetProductSellerID(gomock.Any(), int64(38)).Return(int64(2), nil)
	m.EXPECT().ExistsPurchaseRequest(gomock.Any(), int64(1), int64(2), int64(38)).
		Return(false, nil)

	_, err := s.CreateReview(context.Background(), 1, validCreateReq())
	assert.ErrorIs(t, err, ErrNotPurchased)
}

func TestCreateReview_AlreadyExists(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetProductSellerID(gomock.Any(), int64(38)).Return(int64(2), nil)
	m.EXPECT().ExistsPurchaseRequest(gomock.Any(), int64(1), int64(2), int64(38)).
		Return(true, nil)
	m.EXPECT().Create(gomock.Any(), gomock.Any()).
		Return(models.Review{}, reviewrepo.ErrReviewAlreadyExists)

	_, err := s.CreateReview(context.Background(), 1, validCreateReq())
	assert.ErrorIs(t, err, reviewrepo.ErrReviewAlreadyExists)
}

func TestCreateReview_StorageInternal(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetProductSellerID(gomock.Any(), int64(38)).Return(int64(2), nil)
	m.EXPECT().ExistsPurchaseRequest(gomock.Any(), int64(1), int64(2), int64(38)).
		Return(true, nil)
	m.EXPECT().Create(gomock.Any(), gomock.Any()).
		Return(models.Review{}, errors.New("db down"))

	_, err := s.CreateReview(context.Background(), 1, validCreateReq())
	assert.Error(t, err)
	assert.NotErrorIs(t, err, reviewrepo.ErrReviewAlreadyExists)
}

func TestCreateReview_GetResponseByIDError(t *testing.T) {
	// INSERT прошёл, но повторное чтение с JOIN упало — должны вернуть ошибку.
	s, m := setupService(t)
	now := time.Now()
	m.EXPECT().GetProductSellerID(gomock.Any(), int64(38)).Return(int64(2), nil)
	m.EXPECT().ExistsPurchaseRequest(gomock.Any(), int64(1), int64(2), int64(38)).
		Return(true, nil)
	m.EXPECT().Create(gomock.Any(), gomock.Any()).
		Return(models.Review{
			ID: 101, SenderID: 1, ReceiverID: 2, ProductID: 38,
			Rating: 5, Content: testContentGood, CreatedAt: now, UpdatedAt: now,
		}, nil)
	m.EXPECT().GetResponseByID(gomock.Any(), int64(101)).
		Return(dto.ReviewResponse{}, errors.New("db down"))

	_, err := s.CreateReview(context.Background(), 1, validCreateReq())
	assert.Error(t, err)
}

func TestCreateReview_Happy(t *testing.T) {
	s, m := setupService(t)
	now := time.Now()
	m.EXPECT().GetProductSellerID(gomock.Any(), int64(38)).Return(int64(2), nil)
	m.EXPECT().ExistsPurchaseRequest(gomock.Any(), int64(1), int64(2), int64(38)).
		Return(true, nil)
	m.EXPECT().Create(gomock.Any(), gomock.Any()).
		Return(models.Review{
			ID: 101, SenderID: 1, ReceiverID: 2, ProductID: 38,
			Rating: 5, Content: testContentGood, CreatedAt: now, UpdatedAt: now,
		}, nil)
	m.EXPECT().GetResponseByID(gomock.Any(), int64(101)).
		Return(dto.ReviewResponse{
			ID:         101,
			Sender:     dto.UserPreview{ID: 1, Name: "Ivan", AvatarPath: "ava.png"},
			ReceiverID: 2,
			Product:    dto.AdPreview{ID: 38, Title: "iPhone", Price: 1000, Status: "active", Photo: "img.jpg"},
			Rating:     5,
			Content:    testContentGood,
			CreatedAt:  now,
			UpdatedAt:  now,
		}, nil)

	resp, err := s.CreateReview(context.Background(), 1, validCreateReq())
	require.NoError(t, err)
	assert.Equal(t, int64(101), resp.ID)
	assert.Equal(t, int64(2), resp.ReceiverID)
	assert.Equal(t, 5, resp.Rating)
	// Главный регресс-ассерт: превью отправителя и товара заполнены.
	assert.Equal(t, int64(1), resp.Sender.ID)
	assert.Equal(t, "Ivan", resp.Sender.Name)
	assert.Equal(t, int64(38), resp.Product.ID)
	assert.Equal(t, "iPhone", resp.Product.Title)
	assert.Equal(t, int64(1000), resp.Product.Price)
	assert.Equal(t, "active", resp.Product.Status)
}

func TestUpdateReview_NotFound(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{}, reviewrepo.ErrReviewNotFound)

	_, err := s.UpdateReview(context.Background(), 1, 101, dto.UpdateReviewRequest{Rating: 4, Content: testContentEdit})
	assert.ErrorIs(t, err, reviewrepo.ErrReviewNotFound)
}

func TestUpdateReview_NotAuthor(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{ID: 101, SenderID: 9}, nil)

	_, err := s.UpdateReview(context.Background(), 1, 101, dto.UpdateReviewRequest{Rating: 4, Content: testContentEdit})
	assert.ErrorIs(t, err, ErrForbiddenReviewEdit)
}

func TestUpdateReview_GetByIDInternalError(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{}, errors.New("db down"))

	_, err := s.UpdateReview(context.Background(), 1, 101, dto.UpdateReviewRequest{Rating: 4, Content: testContentEdit})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, reviewrepo.ErrReviewNotFound)
	assert.NotErrorIs(t, err, ErrForbiddenReviewEdit)
}

func TestUpdateReview_InvalidInput(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{ID: 101, SenderID: 1}, nil)

	_, err := s.UpdateReview(context.Background(), 1, 101, dto.UpdateReviewRequest{Rating: 0, Content: "edit text"})
	assert.ErrorIs(t, err, ErrInvalidRating)
}

func TestUpdateReview_InvalidContent(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{ID: 101, SenderID: 1}, nil)

	_, err := s.UpdateReview(context.Background(), 1, 101, dto.UpdateReviewRequest{Rating: 4, Content: "abc"})
	assert.ErrorIs(t, err, ErrInvalidContent)
}

func TestUpdateReview_StorageRaceNotFound(t *testing.T) {
	// Гонка: GetByID нашёл, но Update вернул ErrReviewNotFound
	// (другой пользователь удалил отзыв между двумя запросами).
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{ID: 101, SenderID: 1}, nil)
	m.EXPECT().Update(gomock.Any(), int64(101), int64(1), 4, testContentEdited).
		Return(models.Review{}, reviewrepo.ErrReviewNotFound)

	_, err := s.UpdateReview(context.Background(), 1, 101, dto.UpdateReviewRequest{Rating: 4, Content: testContentEdited})
	assert.ErrorIs(t, err, reviewrepo.ErrReviewNotFound)
}

func TestUpdateReview_StorageInternal(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{ID: 101, SenderID: 1}, nil)
	m.EXPECT().Update(gomock.Any(), int64(101), int64(1), 4, testContentEdited).
		Return(models.Review{}, errors.New("db down"))

	_, err := s.UpdateReview(context.Background(), 1, 101, dto.UpdateReviewRequest{Rating: 4, Content: testContentEdited})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, reviewrepo.ErrReviewNotFound)
}

func TestUpdateReview_Happy(t *testing.T) {
	s, m := setupService(t)
	now := time.Now()
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{ID: 101, SenderID: 1}, nil)
	m.EXPECT().Update(gomock.Any(), int64(101), int64(1), 4, testContentEdited).
		Return(models.Review{
			ID: 101, SenderID: 1, ReceiverID: 2, ProductID: 38,
			Rating: 4, Content: testContentEdited, CreatedAt: now, UpdatedAt: now,
		}, nil)
	m.EXPECT().GetResponseByID(gomock.Any(), int64(101)).
		Return(dto.ReviewResponse{
			ID:         101,
			Sender:     dto.UserPreview{ID: 1, Name: "Ivan", AvatarPath: "ava.png"},
			ReceiverID: 2,
			Product:    dto.AdPreview{ID: 38, Title: "iPhone", Price: 1000, Status: "active", Photo: "img.jpg"},
			Rating:     4,
			Content:    testContentEdited,
			CreatedAt:  now,
			UpdatedAt:  now,
		}, nil)

	resp, err := s.UpdateReview(context.Background(), 1, 101, dto.UpdateReviewRequest{Rating: 4, Content: testContentEdited})
	require.NoError(t, err)
	assert.Equal(t, 4, resp.Rating)
	assert.Equal(t, testContentEdited, resp.Content)
	// Главный регресс-ассерт: превью заполнены и в PUT.
	assert.Equal(t, int64(1), resp.Sender.ID)
	assert.Equal(t, "Ivan", resp.Sender.Name)
	assert.Equal(t, int64(38), resp.Product.ID)
	assert.Equal(t, "iPhone", resp.Product.Title)
}

func TestUpdateReview_GetResponseByIDError(t *testing.T) {
	// UPDATE прошёл, но повторное чтение упало — usecase возвращает ошибку.
	s, m := setupService(t)
	now := time.Now()
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{ID: 101, SenderID: 1}, nil)
	m.EXPECT().Update(gomock.Any(), int64(101), int64(1), 4, testContentEdited).
		Return(models.Review{
			ID: 101, SenderID: 1, ReceiverID: 2, ProductID: 38,
			Rating: 4, Content: testContentEdited, CreatedAt: now, UpdatedAt: now,
		}, nil)
	m.EXPECT().GetResponseByID(gomock.Any(), int64(101)).
		Return(dto.ReviewResponse{}, errors.New("db down"))

	_, err := s.UpdateReview(context.Background(), 1, 101, dto.UpdateReviewRequest{Rating: 4, Content: testContentEdited})
	assert.Error(t, err)
}

func TestDeleteReview_NotFound(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{}, reviewrepo.ErrReviewNotFound)

	err := s.DeleteReview(context.Background(), 1, 101)
	assert.ErrorIs(t, err, reviewrepo.ErrReviewNotFound)
}

func TestDeleteReview_NotAuthor(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{ID: 101, SenderID: 9}, nil)

	err := s.DeleteReview(context.Background(), 1, 101)
	assert.ErrorIs(t, err, ErrForbiddenReviewEdit)
}

func TestDeleteReview_GetByIDInternalError(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{}, errors.New("db down"))

	err := s.DeleteReview(context.Background(), 1, 101)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, reviewrepo.ErrReviewNotFound)
	assert.NotErrorIs(t, err, ErrForbiddenReviewEdit)
}

func TestDeleteReview_StorageRaceNotFound(t *testing.T) {
	// Гонка: GetByID нашёл, но Delete вернул ErrReviewNotFound.
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{ID: 101, SenderID: 1}, nil)
	m.EXPECT().Delete(gomock.Any(), int64(101), int64(1)).
		Return(reviewrepo.ErrReviewNotFound)

	err := s.DeleteReview(context.Background(), 1, 101)
	assert.ErrorIs(t, err, reviewrepo.ErrReviewNotFound)
}

func TestDeleteReview_StorageInternal(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{ID: 101, SenderID: 1}, nil)
	m.EXPECT().Delete(gomock.Any(), int64(101), int64(1)).
		Return(errors.New("db down"))

	err := s.DeleteReview(context.Background(), 1, 101)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, reviewrepo.ErrReviewNotFound)
}

func TestDeleteReview_Happy(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().GetByID(gomock.Any(), int64(101)).
		Return(models.Review{ID: 101, SenderID: 1}, nil)
	m.EXPECT().Delete(gomock.Any(), int64(101), int64(1)).Return(nil)

	err := s.DeleteReview(context.Background(), 1, 101)
	assert.NoError(t, err)
}

func TestGetUserReviews_ClampLimit(t *testing.T) {
	cases := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero -> default", 0, defaultListLim},
		{"negative -> default", -5, defaultListLim},
		{"over max -> max", 1000, maxListLim},
		{"normal", 25, 25},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, m := setupService(t)
			m.EXPECT().ListByReceiver(gomock.Any(), int64(2), gomock.Nil(), tc.expected+1).
				Return([]dto.ReviewResponse{}, nil)

			_, err := s.GetUserReviews(context.Background(), 2, nil, tc.input)
			assert.NoError(t, err)
		})
	}
}

func TestGetUserReviews_Empty(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().ListByReceiver(gomock.Any(), int64(2), gomock.Nil(), 21).
		Return([]dto.ReviewResponse{}, nil)

	resp, err := s.GetUserReviews(context.Background(), 2, nil, 0)
	require.NoError(t, err)
	assert.Nil(t, resp.NextCursor)
	assert.Len(t, resp.Reviews, 0)
}

func TestGetUserReviews_HasNextCursor(t *testing.T) {
	s, m := setupService(t)
	items := []dto.ReviewResponse{
		{ID: 10}, {ID: 9}, {ID: 8},
	}
	m.EXPECT().ListByReceiver(gomock.Any(), int64(2), gomock.Nil(), 3).
		Return(items, nil)

	resp, err := s.GetUserReviews(context.Background(), 2, nil, 2)
	require.NoError(t, err)
	require.NotNil(t, resp.NextCursor)
	assert.Equal(t, int64(9), *resp.NextCursor)
	assert.Len(t, resp.Reviews, 2)
}

func TestGetUserReviews_StorageError(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().ListByReceiver(gomock.Any(), int64(2), gomock.Nil(), 21).
		Return(nil, errors.New("db down"))

	_, err := s.GetUserReviews(context.Background(), 2, nil, 0)
	assert.Error(t, err)
}

func TestGetMyReviews_CursorProxy(t *testing.T) {
	s, m := setupService(t)
	cur := int64(50)
	m.EXPECT().ListBySender(gomock.Any(), int64(1), &cur, 21).
		Return([]dto.ReviewResponse{}, nil)

	resp, err := s.GetMyReviews(context.Background(), 1, &cur, 0)
	require.NoError(t, err)
	assert.Nil(t, resp.NextCursor)
}

func TestGetMyReviews_HasNextCursor(t *testing.T) {
	s, m := setupService(t)
	items := []dto.ReviewResponse{{ID: 5}, {ID: 4}, {ID: 3}}
	m.EXPECT().ListBySender(gomock.Any(), int64(1), gomock.Nil(), 3).
		Return(items, nil)

	resp, err := s.GetMyReviews(context.Background(), 1, nil, 2)
	require.NoError(t, err)
	require.NotNil(t, resp.NextCursor)
	assert.Equal(t, int64(4), *resp.NextCursor)
	assert.Len(t, resp.Reviews, 2)
}

func TestGetMyReviews_StorageError(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().ListBySender(gomock.Any(), int64(1), gomock.Nil(), 21).
		Return(nil, errors.New("db down"))

	_, err := s.GetMyReviews(context.Background(), 1, nil, 0)
	assert.Error(t, err)
}

func TestGetUserReviewsSummary_Happy(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().SummaryByReceiver(gomock.Any(), int64(2)).Return(dto.ReviewSummaryResponse{
		Average:      4.5,
		Total:        4,
		Distribution: map[int]int{5: 3, 4: 1},
	}, nil)

	resp, err := s.GetUserReviewsSummary(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, 4, resp.Total)
	assert.InDelta(t, 4.5, resp.Average, 0.001)
}

func TestGetUserReviewsSummary_Empty(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().SummaryByReceiver(gomock.Any(), int64(2)).Return(dto.ReviewSummaryResponse{
		Distribution: map[int]int{},
	}, nil)

	resp, err := s.GetUserReviewsSummary(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Total)
	assert.Equal(t, float64(0), resp.Average)
	assert.NotNil(t, resp.Distribution)
	assert.Len(t, resp.Distribution, 0)
}

func TestGetUserReviewsSummary_StorageError(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().SummaryByReceiver(gomock.Any(), int64(2)).
		Return(dto.ReviewSummaryResponse{}, errors.New("db down"))

	_, err := s.GetUserReviewsSummary(context.Background(), 2)
	assert.Error(t, err)
}

func TestGetUserReviewsSummary_NilDistributionFromRepo(t *testing.T) {
	s, m := setupService(t)
	m.EXPECT().SummaryByReceiver(gomock.Any(), int64(2)).
		Return(dto.ReviewSummaryResponse{Average: 5, Total: 1}, nil)

	resp, err := s.GetUserReviewsSummary(context.Background(), 2)
	require.NoError(t, err)
	assert.NotNil(t, resp.Distribution)
}
