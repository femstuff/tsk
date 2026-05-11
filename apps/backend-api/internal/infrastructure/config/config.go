package config

import (
	"os"
	"time"
)

type Config struct {
	ServiceName        string
	Environment        string
	HTTPPort           string
	DatabaseURL        string
	StorageRoot        string
	BitrixWebhookURL   string
	WhisperBaseURL     string
	ApprovalEmail      string
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	ShutdownTimeout    time.Duration
	ProcessingInterval time.Duration
}

func Load() Config {
	return Config{
		ServiceName:        getenv("SERVICE_NAME", "tsk-backend-api"),
		Environment:        getenv("APP_ENV", "local"),
		HTTPPort:           getenv("APP_PORT", "8080"),
		DatabaseURL:        getenv("DATABASE_URL", "postgres://tsk:tsk@localhost:5432/tsk?sslmode=disable"),
		StorageRoot:        getenv("STORAGE_ROOT", "./data/storage"),
		BitrixWebhookURL:   getenv("BITRIX_WEBHOOK_URL", ""),
		WhisperBaseURL:     getenv("WHISPER_BASE_URL", ""),
		ApprovalEmail:      getenv("APPROVAL_EMAIL", "ops@example.local"),
		ReadTimeout:        durationFromEnv("HTTP_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:       durationFromEnv("HTTP_WRITE_TIMEOUT", 120*time.Second),
		ShutdownTimeout:    durationFromEnv("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		ProcessingInterval: durationFromEnv("JOB_PROCESSING_INTERVAL", 2*time.Second),
	}
}

func (c Config) Address() string {
	return ":" + c.HTTPPort
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
