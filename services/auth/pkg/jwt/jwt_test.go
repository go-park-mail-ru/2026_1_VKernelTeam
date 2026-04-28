package jwt

import (
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/models"
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

func TestParseToken(t *testing.T) {
	user := models.User{ID: 42}
	secret := "my_super_secret"
	duration := time.Minute * 5

	tokenStr, err := NewToken(user, duration, secret)
	require.NoError(t, err)

	t.Run("Valid token", func(t *testing.T) {
		parsedToken, err := ParseToken(tokenStr, secret)
		require.NoError(t, err)
		assert.True(t, parsedToken.Valid)
	})

	t.Run("Invalid secret", func(t *testing.T) {
		wrongSecret := "wrong_secret"
		_, err := ParseToken(tokenStr, wrongSecret)
		assert.Error(t, err)
	})
}
