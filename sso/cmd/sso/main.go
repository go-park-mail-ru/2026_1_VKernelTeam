package main

import (
	"fmt"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/config"
)

func main() {
	cfg := config.MustLoadConfig()
	fmt.Println(cfg)
}
