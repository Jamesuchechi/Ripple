package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL        string
	RedisURL           string
	QueueURL           string
	CelebrityThreshold int
	FanoutConcurrency  int
	FeedCacheSize      int
	CelebrityCacheTTL  int
	APIPort            string
	NotifierPort       string
	WebhookHMACSecret  string
	MaxDLQRetries      int
}

func Load() *Config {
	return &Config{
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://ripple:secret@localhost:5432/ripple?sslmode=disable"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379/0"),
		QueueURL:           getEnv("QUEUE_URL", "nats://localhost:4222"),
		CelebrityThreshold: getEnvAsInt("CELEBRITY_THRESHOLD", 10000),
		FanoutConcurrency:  getEnvAsInt("FANOUT_CONCURRENCY", 100),
		FeedCacheSize:      getEnvAsInt("FEED_CACHE_SIZE", 1000),
		CelebrityCacheTTL:  getEnvAsInt("CELEBRITY_CACHE_TTL", 10),
		APIPort:            getEnv("API_PORT", "8080"),
		NotifierPort:       getEnv("NOTIFIER_PORT", "8081"),
		WebhookHMACSecret:  getEnv("WEBHOOK_HMAC_SECRET", "dev-secret-change-me"),
		MaxDLQRetries:      getEnvAsInt("MAX_DLQ_RETRIES", 3),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return fallback
	}
	return val
}
