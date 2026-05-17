package supportticket_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/models"
	supportticket "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/usecase/support_ticket"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/usecase/support_ticket/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStats_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockTicketStorage(ctrl)
	svc := supportticket.New(discardLogger(), mockStorage)

	want := &dto.StatsResponse{Total: 5}
	mockStorage.EXPECT().GetStats(gomock.Any()).Return(want, nil)

	got, err := svc.GetStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetStats_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockTicketStorage(ctrl)
	svc := supportticket.New(discardLogger(), mockStorage)

	mockStorage.EXPECT().GetStats(gomock.Any()).Return(nil, errors.New("db down"))

	_, err := svc.GetStats(context.Background())
	assert.Error(t, err)
}

func TestRateTicket_InvalidRating(t *testing.T) {
	svc := supportticket.New(discardLogger(), nil)
	_, err := svc.RateTicket(context.Background(), 1, 1, 0)
	assert.ErrorIs(t, err, supportticket.ErrInvalidRating)

	_, err = svc.RateTicket(context.Background(), 1, 1, 6)
	assert.ErrorIs(t, err, supportticket.ErrInvalidRating)
}

func TestRateTicket_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockTicketStorage(ctrl)
	svc := supportticket.New(discardLogger(), mockStorage)

	mockStorage.EXPECT().GetByID(gomock.Any(), int64(5)).
		Return(&models.SupportTicket{ID: 5, UserID: 999, Status: statusClosed}, nil)

	_, err := svc.RateTicket(context.Background(), 7, 5, 3)
	assert.ErrorIs(t, err, supportticket.ErrForbidden)
}

func TestRateTicket_NotClosed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockTicketStorage(ctrl)
	svc := supportticket.New(discardLogger(), mockStorage)

	mockStorage.EXPECT().GetByID(gomock.Any(), int64(5)).
		Return(&models.SupportTicket{ID: 5, UserID: 7, Status: statusOpen}, nil)

	_, err := svc.RateTicket(context.Background(), 7, 5, 3)
	assert.ErrorIs(t, err, supportticket.ErrTicketNotClosed)
}

func TestRateTicket_AlreadyRated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockTicketStorage(ctrl)
	svc := supportticket.New(discardLogger(), mockStorage)

	rating := 4
	mockStorage.EXPECT().GetByID(gomock.Any(), int64(5)).
		Return(&models.SupportTicket{ID: 5, UserID: 7, Status: statusClosed, Rating: &rating}, nil)

	_, err := svc.RateTicket(context.Background(), 7, 5, 3)
	assert.ErrorIs(t, err, supportticket.ErrAlreadyRated)
}

func TestRateTicket_GetByIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockTicketStorage(ctrl)
	svc := supportticket.New(discardLogger(), mockStorage)

	mockStorage.EXPECT().GetByID(gomock.Any(), int64(5)).Return(nil, errors.New("db"))

	_, err := svc.RateTicket(context.Background(), 7, 5, 3)
	assert.Error(t, err)
}

func TestRateTicket_SetRatingError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockTicketStorage(ctrl)
	svc := supportticket.New(discardLogger(), mockStorage)

	mockStorage.EXPECT().GetByID(gomock.Any(), int64(5)).
		Return(&models.SupportTicket{ID: 5, UserID: 7, Status: statusClosed}, nil)
	mockStorage.EXPECT().SetRating(gomock.Any(), int64(5), 3).Return(errors.New("db"))

	_, err := svc.RateTicket(context.Background(), 7, 5, 3)
	assert.Error(t, err)
}
