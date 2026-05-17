// Package config — параметры commerce-сервиса.
package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/joho/godotenv"
)

// Config — параметры commerce.
type Config struct {
	Env             string
	DatabaseDSN     string
	AuthGRPCAddr    string
	CatalogGRPCAddr string
	HTTP            HTTPConfig
	Kafka           KafkaConfig
}

type HTTPConfig struct {
	Port int `json:"port"`
}

type KafkaConfig struct {
	Brokers      string
	AdEventTopic string
	GroupID      string
}

// MustLoadConfig читает path из --config / CONFIG_PATH и .env.
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

	var raw struct {
		Env  string     `json:"env"`
		HTTP HTTPConfig `json:"http"`
	}
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
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
		Env:             raw.Env,
		DatabaseDSN:     mustEnv("DATABASE_DSN"),
		AuthGRPCAddr:    envOrDefault("AUTH_GRPC_ADDR", "localhost:9001"),
		CatalogGRPCAddr: envOrDefault("CATALOG_GRPC_ADDR", "localhost:9004"),
		HTTP:            raw.HTTP,
		Kafka: KafkaConfig{
			Brokers:      envOrDefault("KAFKA_BROKERS", "localhost:9092"),
			AdEventTopic: envOrDefault("KAFKA_AD_EVENT_TOPIC", "clover.catalog.ad-events"),
			GroupID:      envOrDefault("KAFKA_GROUP_ID", "commerce-service"),
		},
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
