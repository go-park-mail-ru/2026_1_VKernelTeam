package jwt

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/domain/models"
)

//
func NewToken(user models.User, app models.App, duration time.Duration) (string, error) {
	panic("implement me")
}


