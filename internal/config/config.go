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
	RedisAddr       string
	TokenSecret     string
	RefreshTTL      time.Duration
	TokenTTL        time.Duration
	CleanupInterval time.Duration
	HTTP            HTTPConfig
	S3Storage       S3Config
}

// HTTPConfig содержит настройки HTTP-сервера.
type HTTPConfig struct {
	Port int `json:"port"`
}

type S3Config struct {
	EndpointURL     string `json:"endpoint_url"`
	RegionName      string `json:"region_name"`
	BucketName      string `json:"bucket_name"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
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
		RefreshTTL      string     `json:"refresh_ttl"`
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

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		panic("REDIS_ADDR is not set in environment or .env file")
	}

	s3EndpointURL := os.Getenv("S3_ENDPOINT_URL")
	if s3EndpointURL == "" {
		panic("S3_ENDPOINT_URL is not set in environment or .env file")
	}

	s3RegionName := os.Getenv("S3_REGION_NAME")
	if s3RegionName == "" {
		panic("S3_REGION_NAME is not set in environment or .env file")
	}

	s3BucketName := os.Getenv("S3_BUCKET_NAME")
	if s3BucketName == "" {
		panic("S3_BUCKET_NAME is not set in environment or .env file")
	}

	s3AccessKeyID := os.Getenv("S3_ACCESS_KEY_ID")
	if s3AccessKeyID == "" {
		panic("S3_ACCESS_KEY_ID is not set in environment or .env file")
	}

	s3SecretAccessKey := os.Getenv("S3_SECRET_ACCESS_KEY")
	if s3SecretAccessKey == "" {
		panic("S3_SECRET_ACCESS_KEY is not set in environment or .env file")
	}

	// Перекладываем данные в "чистую" бизнес-модель,
	// попутно преобразуя типы с помощью хелпера.
	return &Config{
		Env:             rawConfig.Env,
		DatabaseDSN:     dsn,
		RedisAddr:       redisAddr,
		RefreshTTL:      parseDuration(rawConfig.RefreshTTL, "refresh_ttl"),
		TokenTTL:        parseDuration(rawConfig.TokenTTL, "token_ttl"),
		HTTP:            rawConfig.HTTP,
		CleanupInterval: parseDuration(rawConfig.CleanupInterval, "cleanup_interval"),
		S3Storage: S3Config{
			EndpointURL:     s3EndpointURL,
			RegionName:      s3RegionName,
			BucketName:      s3BucketName,
			AccessKeyID:     s3AccessKeyID,
			SecretAccessKey: s3SecretAccessKey,
		},
		TokenSecret: secret,
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
