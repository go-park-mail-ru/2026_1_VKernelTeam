package jwt

import (
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToken(t *testing.T) {
	user := models.User{
		ID:    1,
		Email: "test@example.com",
	}
	duration := time.Hour
	secret := "test_secret"

	tokenStr, err := NewToken(user, duration, secret)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)
}
