package supportmessage_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/models"
	supportmessage "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/usecase/support_message"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/usecase/support_message/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestSupportMessageService_SendMessage(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	t.Run("SuccessAsAuthor", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ticketMock := mocks.NewMockTicketStorage(ctrl)
		messageMock := mocks.NewMockMessageStorage(ctrl)
		roleMock := mocks.NewMockRoleProvider(ctrl)
		service := supportmessage.New(log, messageMock, ticketMock, roleMock)

		ticketMock.EXPECT().GetByID(ctx, int64(1)).Return(&models.SupportTicket{ID: 1, UserID: 2}, nil)
		messageMock.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, msg *models.SupportMessage) (int64, error) {
			msg.ID = 11
			msg.CreatedAt = time.Now()
			return int64(11), nil
		})

		resp, err := service.SendMessage(ctx, 1, 2, &dto.SendMessageRequest{Text: "Hello"})
		require.NoError(t, err)
		assert.Equal(t, int64(11), resp.ID)
		assert.Equal(t, "Hello", resp.Text)
	})

	t.Run("Forbidden", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ticketMock := mocks.NewMockTicketStorage(ctrl)
		messageMock := mocks.NewMockMessageStorage(ctrl)
		roleMock := mocks.NewMockRoleProvider(ctrl)
		service := supportmessage.New(log, messageMock, ticketMock, roleMock)

		ticketMock.EXPECT().GetByID(ctx, int64(1)).Return(&models.SupportTicket{ID: 1, UserID: 2}, nil)
		roleMock.EXPECT().GetUserRole(ctx, int64(3)).Return("user", nil)

		_, err := service.SendMessage(ctx, 1, 3, &dto.SendMessageRequest{Text: "Hello"})
		assert.ErrorIs(t, err, supportmessage.ErrForbidden)
	})

	t.Run("TextRequired", func(t *testing.T) {
		service := supportmessage.New(log, nil, nil, nil)
		_, err := service.SendMessage(ctx, 1, 2, &dto.SendMessageRequest{Text: ""})
		assert.ErrorIs(t, err, supportmessage.ErrTextRequired)
	})
}

func TestSupportMessageService_GetMessages(t *testing.T) {
	log := discardLogger()
	ctx := context.Background()

	t.Run("SuccessAsSupport", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ticketMock := mocks.NewMockTicketStorage(ctrl)
		messageMock := mocks.NewMockMessageStorage(ctrl)
		roleMock := mocks.NewMockRoleProvider(ctrl)
		service := supportmessage.New(log, messageMock, ticketMock, roleMock)

		message := models.SupportMessage{ID: 1, TicketID: 1, UserID: 3, Text: "Hi", CreatedAt: time.Now()}
		ticketMock.EXPECT().GetByID(ctx, int64(1)).Return(&models.SupportTicket{ID: 1, UserID: 2}, nil)
		roleMock.EXPECT().GetUserRole(ctx, int64(3)).Return("support", nil)
		messageMock.EXPECT().GetByTicketID(ctx, int64(1)).Return([]models.SupportMessage{message}, nil)

		messages, err := service.GetMessages(ctx, 1, 3)
		require.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, "Hi", messages[0].Text)
	})

	t.Run("Forbidden", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ticketMock := mocks.NewMockTicketStorage(ctrl)
		messageMock := mocks.NewMockMessageStorage(ctrl)
		roleMock := mocks.NewMockRoleProvider(ctrl)
		service := supportmessage.New(log, messageMock, ticketMock, roleMock)

		ticketMock.EXPECT().GetByID(ctx, int64(1)).Return(&models.SupportTicket{ID: 1, UserID: 2}, nil)
		roleMock.EXPECT().GetUserRole(ctx, int64(3)).Return("user", nil)

		_, err := service.GetMessages(ctx, 1, 3)
		assert.ErrorIs(t, err, supportmessage.ErrForbidden)
	})
}
