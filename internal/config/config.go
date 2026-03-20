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
	DatabaseDSN     string
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
	// В тестах .env файл может не существовать, поэтому ошибки
	// чтения игнорируем. Обычно отсутствие .env не является фатальной ошибкой,
	// конфигурация может быть передана через реальное окружение.
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
		TokenTTL        string     `json:"token_ttl"`
		HTTP            HTTPConfig `json:"http"`
		CleanupInterval string     `json:"cleanup_interval"`
	}

	if err := json.NewDecoder(file).Decode(&rawConfig); err != nil {
		panic("failed to decode config: " + err.Error())
	}

	secret := os.Getenv("TOKEN_SECRET")
	if secret == "" {
		panic("TOKEN_SECRET is not set in environment or .env file")
	}

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		panic("DATABASE_DSN is not set in environment or .env file")
	}

	// Перекладываем данные в "чистую" бизнес-модель,
	// попутно преобразуя типы с помощью хелпера.
	return &Config{
		Env:             rawConfig.Env,
		DatabaseDSN:     dsn,
		TokenTTL:        parseDuration(rawConfig.TokenTTL, "token_ttl"),
		HTTP:            rawConfig.HTTP,
		CleanupInterval: parseDuration(rawConfig.CleanupInterval, "cleanup_interval"),
		TokenSecret:     secret,
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
	if !flag.Parsed() {
		flag.Parse()
	}

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
