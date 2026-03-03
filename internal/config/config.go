// Package config отвечает за загрузку и парсинг конфигурации из YAML-файла
// либо переменных окружения.
package config

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config содержит параметры работы сервиса: окружение, путь к хранилищу,
// время жизни токена и настройки HTTP-сервера.
type Config struct {
	Env         string        `json:"env"`
	StoragePath string        `json:"storage_path"`
	TokenTTL    time.Duration `json:"token_ttl"`
	HTTP        HTTPConfig    `json:"http"`
	TokenSecret string        `json:"token_secret"`
}

// HTTPConfig содержит настройки HTTP-сервера.
type HTTPConfig struct {
	Port int `json:"port"`
}

// MustLoadConfig загружает конфигурацию и паникует в случае ошибки.
// Это удобный хелпер для вызова из main.
func MustLoadConfig() *Config {
	_ = godotenv.Load()

	path := fetchConfigPath()
	if path == "" {
		panic("config path is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file does not exist: " + path)
	}

	var cfg Config

	file, err := os.Open(path)
	if err != nil {
		panic("failed to open config file: " + err.Error())
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		panic("failed to decode config: " + err.Error())
	}

	return &cfg
}

// fetchConfigPath определяет путь к файлу конфигурации из флага
// командной строки или переменной окружения.
func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "Path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
