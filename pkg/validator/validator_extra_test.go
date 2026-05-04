package validator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAdDescription(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		assert.ErrorIs(t, ValidateAdDescription(""), ErrAdDescriptionEmpty)
	})
	t.Run("too short", func(t *testing.T) {
		assert.ErrorIs(t, ValidateAdDescription("short"), ErrAdDescriptionTooShort)
	})
	t.Run("too long", func(t *testing.T) {
		assert.ErrorIs(t, ValidateAdDescription(strings.Repeat("a", 5001)), ErrAdDescriptionTooLong)
	})
	t.Run("ok", func(t *testing.T) {
		assert.NoError(t, ValidateAdDescription(strings.Repeat("a", 50)))
	})
}

func TestValidateUserID(t *testing.T) {
	assert.ErrorIs(t, ValidateUserID(0), ErrUserIDInvalid)
	assert.ErrorIs(t, ValidateUserID(-5), ErrUserIDInvalid)
	assert.NoError(t, ValidateUserID(1))
}

func TestValidateProductID(t *testing.T) {
	assert.ErrorIs(t, ValidateProductID(0), ErrProductIDInvalid)
	assert.ErrorIs(t, ValidateProductID(-1), ErrProductIDInvalid)
	assert.NoError(t, ValidateProductID(42))
}
