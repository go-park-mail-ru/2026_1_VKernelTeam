package config

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config содержит параметры работы Catalog-сервиса.
type Config struct {
	Env          string
	DatabaseDSN  string
	RedisAddr    string
	AuthGRPCAddr string
	HTTP         HTTPConfig
	GRPC         GRPCConfig
	S3Storage    S3Config
	Kafka        KafkaConfig
	Search       SearchConfig
	Views        ViewsConfig
}

type HTTPConfig struct {
	Port int `json:"port"`
}

type GRPCConfig struct {
	Port int `json:"port"`
}

type S3Config struct {
	EndpointURL     string
	RegionName      string
	BucketName      string
	AccessKeyID     string
	SecretAccessKey string
}

type KafkaConfig struct {
	Brokers        string
	UserEventTopic string
	GroupID        string
}

// SearchConfig — параметры pg_trgm полнотекстового поиска.
type SearchConfig struct {
	MaxResults              int     `json:"max_results"`
	MinQueryLength          int     `json:"min_query_length"`
	SimilarityThreshold     float64 `json:"similarity_threshold"`
	WordSimilarityThreshold float64 `json:"word_similarity_threshold"`
}

// ViewsConfig — параметры view-pipeline (Redis Streams + батч-инсёрт).
type ViewsConfig struct {
	DedupTTL      time.Duration
	StreamKey     string
	ConsumerGroup string
	BatchSize     int64
	FlushInterval time.Duration
	BlockTimeout  time.Duration
	CountCacheTTL time.Duration
	ClaimTimeout  time.Duration
}

// MustLoadConfig читает path из --config / CONFIG_PATH, мержит с .env переменными.
// Падает с panic при отсутствии обязательных значений.
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
		Env    string       `json:"env"`
		HTTP   HTTPConfig   `json:"http"`
		GRPC   GRPCConfig   `json:"grpc"`
		Search SearchConfig `json:"search"`
		Views  struct {
			DedupTTL      string `json:"dedup_ttl"`
			StreamKey     string `json:"stream_key"`
			ConsumerGroup string `json:"consumer_group"`
			BatchSize     int64  `json:"batch_size"`
			FlushInterval string `json:"flush_interval"`
			BlockTimeout  string `json:"block_timeout"`
			CountCacheTTL string `json:"count_cache_ttl"`
			ClaimTimeout  string `json:"claim_timeout"`
		} `json:"views"`
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
		Env:          raw.Env,
		DatabaseDSN:  mustEnv("DATABASE_DSN"),
		RedisAddr:    mustEnv("REDIS_ADDR"),
		AuthGRPCAddr: envOrDefault("AUTH_GRPC_ADDR", "localhost:9001"),
		HTTP:         raw.HTTP,
		GRPC:         raw.GRPC,
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
			GroupID:        envOrDefault("KAFKA_GROUP_ID", "catalog-service"),
		},
		Search: raw.Search,
		Views: ViewsConfig{
			DedupTTL:      parseDuration(raw.Views.DedupTTL, "views.dedup_ttl"),
			StreamKey:     raw.Views.StreamKey,
			ConsumerGroup: raw.Views.ConsumerGroup,
			BatchSize:     raw.Views.BatchSize,
			FlushInterval: parseDuration(raw.Views.FlushInterval, "views.flush_interval"),
			BlockTimeout:  parseDuration(raw.Views.BlockTimeout, "views.block_timeout"),
			CountCacheTTL: parseDuration(raw.Views.CountCacheTTL, "views.count_cache_ttl"),
			ClaimTimeout:  parseDuration(raw.Views.ClaimTimeout, "views.claim_timeout"),
		},
	}
}

func parseDuration(val, fieldName string) time.Duration {
	if val == "" {
		return 0
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		panic("invalid " + fieldName + " format: " + err.Error())
	}
	return d
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
