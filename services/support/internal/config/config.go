package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/joho/godotenv"
)

// Config содержит параметры работы Support-сервиса.
type Config struct {
	Env          string
	DatabaseDSN  string
	AuthGRPCAddr string
	Kafka        KafkaConfig
	HTTP         HTTPConfig
}

// KafkaConfig содержит настройки подключения к Kafka.
type KafkaConfig struct {
	Brokers        string
	UserEventTopic string
	GroupID        string
}

// HTTPConfig содержит настройки HTTP-сервера.
type HTTPConfig struct {
	Port int `json:"port"`
}

// MustLoadConfig загружает конфигурацию из файла и переменных окружения, паникует при ошибке.
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
		Env  string     `json:"env"`
		HTTP HTTPConfig `json:"http"`
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
		Env:          rawConfig.Env,
		DatabaseDSN:  mustEnv("DATABASE_DSN"),
		AuthGRPCAddr: envOrDefault("AUTH_GRPC_ADDR", "localhost:9001"),
		Kafka: KafkaConfig{
			Brokers:        envOrDefault("KAFKA_BROKERS", "localhost:9092"),
			UserEventTopic: envOrDefault("KAFKA_USER_EVENT_TOPIC", "clover.auth.user-events"),
			GroupID:        envOrDefault("KAFKA_GROUP_ID", "support-service"),
		},
		HTTP: rawConfig.HTTP,
	}
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
