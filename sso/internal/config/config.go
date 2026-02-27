// Пакет config отвечает за загрузку и парсинг конфигурации из YAML-файла
// либо переменных окружения.
package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string        `yaml:"env" env-required:"true"`
	StoragePath string        `yaml:"storage_path" env-required:"true"`
	TokenTTL    time.Duration `yaml:"token_ttl" env-required:"true"`
	HTTP        HTTPConfig    `yaml:"http"`
}

// Config содержит параметры работы сервиса: окружение, путь к хранилищу,
// время жизни токена и настройки HTTP-сервера.

type HTTPConfig struct {
	Port int `yaml:"port"`
}

func MustLoadConfig() *Config {
	path := fetchConfigPath()
	if path == "" {
		panic("config path is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file does not exist: " + path)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return &cfg
}

// MustLoadConfig загружает конфигурацию и паникует в случае ошибки.
// Это удобный хелпер для вызова из main.

func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "Path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}

// fetchConfigPath определяет путь к файлу конфигурации из флага
// командной строки или переменной окружения.
