// Package config отвечает за загрузку и парсинг конфигурации из JSON-файла
// либо переменных окружения.
package config

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config содержит параметры работы сервиса.
// Обратите внимание: теги JSON здесь больше не нужны, так как мы
// не десериализуем в эту структуру напрямую. Поля уже имеют нужный тип.
type Config struct {
	Env             string
	StoragePath     string
	TokenTTL        time.Duration
	HTTP            HTTPConfig
	CleanupInterval time.Duration
	TokenSecret     string
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

	file, err := os.Open(path)
	if err != nil {
		panic("failed to open config file: " + err.Error())
	}
	defer file.Close()

	// Анонимная прокси-структура, которая в точности JSON.
	var rawConfig struct {
		Env             string     `json:"env"`
		StoragePath     string     `json:"storage_path"`
		TokenTTL        string     `json:"token_ttl"`
		HTTP            HTTPConfig `json:"http"`
		CleanupInterval string     `json:"cleanup_interval"`
		TokenSecret     string     `json:"token_secret"`
	}

	if err := json.NewDecoder(file).Decode(&rawConfig); err != nil {
		panic("failed to decode config: " + err.Error())
	}

	// Перекладываем данные в "чистую" бизнес-модель,
	// попутно преобразуя типы с помощью хелпера.
	return &Config{
		Env:             rawConfig.Env,
		StoragePath:     rawConfig.StoragePath,
		TokenTTL:        parseDuration(rawConfig.TokenTTL, "token_ttl"),
		HTTP:            rawConfig.HTTP,
		CleanupInterval: parseDuration(rawConfig.CleanupInterval, "cleanup_interval"),
		TokenSecret:     rawConfig.TokenSecret,
	}
}

// parseDuration — универсальная функция для парсинга времени из строк в конфиге.
func parseDuration(val string, fieldName string) time.Duration {
	duration, err := time.ParseDuration(val)
	if err != nil {
		panic("invalid " + fieldName + " format: " + err.Error())
	}
	return duration
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
