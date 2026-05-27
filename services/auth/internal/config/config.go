package config

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config содержит параметры работы Auth-сервиса.
// Сюда входят только настройки, релевантные для авторизации.
// В монолите здесь были ещё Search, Views и т.д. — они ушли в свои сервисы.
type Config struct {
	Env         string
	DatabaseDSN string
	RedisAddr   string
	TokenSecret string
	RefreshTTL  time.Duration
	TokenTTL    time.Duration
	HTTP        HTTPConfig
	GRPC        GRPCConfig
	S3Storage   S3Config
	Kafka       KafkaConfig
}

// KafkaConfig содержит параметры подключения к Kafka и имена используемых топиков.
type KafkaConfig struct {
	Brokers        string
	UserEventTopic string
}

// HTTPConfig содержит параметры HTTP-сервера Auth-сервиса.
type HTTPConfig struct {
	Port int `json:"port"`
}

// GRPCConfig содержит параметры gRPC-сервера Auth-сервиса.
type GRPCConfig struct {
	Port int `json:"port"`
}

// S3Config содержит параметры доступа к S3-хранилищу для пользовательских файлов.
type S3Config struct {
	EndpointURL     string
	RegionName      string
	BucketName      string
	AccessKeyID     string
	SecretAccessKey string
}

// MustLoadConfig читает конфигурацию из JSON-файла и переменных окружения и паникует при ошибке.
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
	defer func() { _ = file.Close() }()

	var rawConfig struct {
		Env        string     `json:"env"`
		RefreshTTL string     `json:"refresh_ttl"`
		TokenTTL   string     `json:"token_ttl"`
		HTTP       HTTPConfig `json:"http"`
		GRPC       GRPCConfig `json:"grpc"`
	}

	if err := json.NewDecoder(file).Decode(&rawConfig); err != nil {
		panic("failed to decode config: " + err.Error())
	}

	mustEnv := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			panic(key + " is not set in environment or .env file")
		}
		return val
	}

	envOrDefault := func(key, def string) string {
		if val := os.Getenv(key); val != "" {
			return val
		}
		return def
	}

	return &Config{
		Env:         rawConfig.Env,
		DatabaseDSN: mustEnv("DATABASE_DSN"),
		RedisAddr:   mustEnv("REDIS_ADDR"),
		TokenSecret: mustEnv("TOKEN_SECRET"),
		RefreshTTL:  parseDuration(rawConfig.RefreshTTL, "refresh_ttl"),
		TokenTTL:    parseDuration(rawConfig.TokenTTL, "token_ttl"),
		HTTP:        rawConfig.HTTP,
		GRPC:        rawConfig.GRPC,
		S3Storage: S3Config{
			EndpointURL:     mustEnv("S3_ENDPOINT_URL"),
			RegionName:      mustEnv("S3_REGION_NAME"),
			BucketName:      mustEnv("S3_BUCKET_NAME"),
			AccessKeyID:     mustEnv("S3_ACCESS_KEY_ID"),
			SecretAccessKey: mustEnv("S3_SECRET_ACCESS_KEY"),
		},
		Kafka: KafkaConfig{
			Brokers:        envOrDefault("KAFKA_BROKERS", "localhost:9092"),
			UserEventTopic: envOrDefault("KAFKA_USER_EVENT_TOPIC", "clover.auth.user-events"),
		},
	}
}

func parseDuration(val string, fieldName string) time.Duration {
	duration, err := time.ParseDuration(val)
	if err != nil {
		panic("invalid " + fieldName + " format: " + err.Error())
	}
	return duration
}

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
