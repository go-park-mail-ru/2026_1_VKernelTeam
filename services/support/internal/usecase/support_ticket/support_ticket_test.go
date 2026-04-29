package supportticket_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/models"
	supportticket "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/usecase/support_ticket"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/usecase/support_ticket/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestSupportTicketService_CreateTicket(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mocks.NewMockTicketStorage(ctrl)
	service := supportticket.New(log, storageMock)

	now := time.Now()

	storageMock.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, t *models.SupportTicket) (int64, error) {
		t.ID = 100
		t.Status = "open"
		t.CreatedAt = now
		t.UpdatedAt = now
		return int64(100), nil
	})

	resp, err := service.CreateTicket(ctx, int64(9), &dto.CreateTicketRequest{Category: "bug", Title: "Title", Description: "Desc"})
	require.NoError(t, err)
	assert.Equal(t, int64(100), resp.ID)
	assert.Equal(t, "open", resp.Status)
}

func TestSupportTicketService_CreateTicket_InvalidInput(t *testing.T) {
	service := supportticket.New(discardLogger(), nil)
	_, err := service.CreateTicket(context.Background(), 1, &dto.CreateTicketRequest{Category: "", Title: "", Description: ""})
	assert.ErrorIs(t, err, supportticket.ErrCategoryRequired)
}

func TestSupportTicketService_GetMyTickets(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mocks.NewMockTicketStorage(ctrl)
	service := supportticket.New(log, storageMock)

	tickets := []models.SupportTicket{{ID: 10, UserID: 5, Category: "bug", Status: "open", Title: "Title", Description: "Desc", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	storageMock.EXPECT().GetByUserID(ctx, int64(5)).Return(tickets, nil)

	resp, err := service.GetMyTickets(ctx, 5)
	require.NoError(t, err)
	assert.Len(t, resp, 1)
	assert.Equal(t, int64(10), resp[0].ID)
}

func TestSupportTicketService_GetAllTickets(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mocks.NewMockTicketStorage(ctrl)
	service := supportticket.New(log, storageMock)

	tickets := []models.SupportTicket{{ID: 11, UserID: 6, Category: "complaint", Status: "open", Title: "Title", Description: "Desc", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	storageMock.EXPECT().GetAll(ctx).Return(tickets, nil)

	resp, err := service.GetAllTickets(ctx)
	require.NoError(t, err)
	assert.Len(t, resp, 1)
	assert.Equal(t, "complaint", resp[0].Category)
}

func TestSupportTicketService_GetTicket_Forbidden(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mocks.NewMockTicketStorage(ctrl)
	service := supportticket.New(log, storageMock)

	storageMock.EXPECT().GetByID(ctx, int64(2)).Return(&models.SupportTicket{ID: 2, UserID: 5}, nil)

	_, err := service.GetTicket(ctx, 2, 3)
	assert.ErrorIs(t, err, supportticket.ErrForbidden)
}

func TestSupportTicketService_ChangeStatus_Success(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mocks.NewMockTicketStorage(ctrl)
	service := supportticket.New(log, storageMock)

	timeValue := time.Now()
	storageMock.EXPECT().UpdateStatus(ctx, int64(4), "closed").Return(timeValue, nil)

	resp, err := service.ChangeStatus(ctx, 4, &dto.ChangeStatusRequest{Status: "closed"})
	require.NoError(t, err)
	assert.Equal(t, "closed", resp.Status)
	assert.WithinDuration(t, timeValue, resp.UpdatedAt, time.Second)
}

func TestSupportTicketService_RateTicket_AlreadyRated(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mocks.NewMockTicketStorage(ctrl)
	service := supportticket.New(log, storageMock)

	storageMock.EXPECT().GetByID(ctx, int64(5)).Return(&models.SupportTicket{ID: 5, UserID: 7, Status: "closed", Rating: ptr(3)}, nil)

	_, err := service.RateTicket(ctx, 7, 5, 4)
	assert.ErrorIs(t, err, supportticket.ErrAlreadyRated)
}

func ptr(v int) *int { return &v }

func TestSupportTicketService_UpdateTicket(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mocks.NewMockTicketStorage(ctrl)
	service := supportticket.New(log, storageMock)

	ticket := &models.SupportTicket{ID: 3, UserID: 5, Status: "open", Category: "bug", Title: "Old", Description: "Old desc", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	storageMock.EXPECT().GetByID(ctx, int64(3)).Return(ticket, nil)
	storageMock.EXPECT().Update(ctx, gomock.Any()).Return(nil)

	resp, err := service.UpdateTicket(ctx, 3, 5, &dto.UpdateTicketRequest{Category: "suggestion", Title: "New", Description: "New desc"})
	require.NoError(t, err)
	assert.Equal(t, "suggestion", resp.Category)
	assert.Equal(t, "New", resp.Title)
}

func TestSupportTicketService_ChangeStatus_InvalidStatus(t *testing.T) {
	service := supportticket.New(discardLogger(), nil)
	_, err := service.ChangeStatus(context.Background(), 1, &dto.ChangeStatusRequest{Status: "invalid"})
	assert.ErrorIs(t, err, supportticket.ErrInvalidStatus)
}

func TestSupportTicketService_RateTicket(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageMock := mocks.NewMockTicketStorage(ctrl)
	service := supportticket.New(log, storageMock)

	storageMock.EXPECT().GetByID(ctx, int64(5)).Return(&models.SupportTicket{ID: 5, UserID: 7, Status: "closed", Rating: nil}, nil)
	storageMock.EXPECT().SetRating(ctx, int64(5), 4).Return(nil)

	resp, err := service.RateTicket(ctx, 7, 5, 4)
	require.NoError(t, err)
	assert.NotNil(t, resp.Rating)
	assert.Equal(t, 4, *resp.Rating)
}
