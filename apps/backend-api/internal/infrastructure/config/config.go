package config

import (
	"os"
	"time"
)

type Config struct {
	ServiceName         string
	Environment         string
	HTTPPort            string
	DatabaseURL         string
	RedisURL            string
	StorageRoot         string
	BitrixWebhookURL        string
	WhisperBaseURL          string
	ApprovalEmail           string
	BitrixOAuthClientID     string
	BitrixOAuthClientSecret string
	BitrixPortalDomain      string
	BitrixOAuthRedirectURI  string
	BitrixMobileAppScheme   string
	PrometheusURL           string
	ReadTimeout             time.Duration
	WriteTimeout        time.Duration
	ShutdownTimeout     time.Duration
	ProcessingInterval  time.Duration
	JobRecoveryInterval time.Duration
}

func Load() Config {
	return Config{
		ServiceName:        getenv("SERVICE_NAME", "tsk-backend-api"),
		Environment:        getenv("APP_ENV", "local"),
		HTTPPort:           getenv("APP_PORT", "8080"),
		DatabaseURL:         getenv("DATABASE_URL", "postgres://tsk:tsk@localhost:5432/tsk?sslmode=disable"),
		RedisURL:            getenv("REDIS_URL", ""),
		StorageRoot:         getenv("STORAGE_ROOT", "./data/storage"),
		BitrixWebhookURL:        getenv("BITRIX_WEBHOOK_URL", ""),
		WhisperBaseURL:          getenv("WHISPER_BASE_URL", ""),
		ApprovalEmail:           getenv("APPROVAL_EMAIL", "ops@example.local"),
		BitrixOAuthClientID:     getenv("BITRIX_OAUTH_CLIENT_ID", ""),
		BitrixOAuthClientSecret: getenv("BITRIX_OAUTH_CLIENT_SECRET", ""),
		BitrixPortalDomain:      getenv("BITRIX_PORTAL_DOMAIN", ""),
		BitrixOAuthRedirectURI:  getenv("BITRIX_OAUTH_REDIRECT_URI", ""),
		BitrixMobileAppScheme:   getenv("BITRIX_MOBILE_APP_SCHEME", "tsk"),
		PrometheusURL:             getenv("PROMETHEUS_URL", ""),
		ReadTimeout:             durationFromEnv("HTTP_READ_TIMEOUT", 300*time.Second),
		WriteTimeout:            durationFromEnv("HTTP_WRITE_TIMEOUT", 300*time.Second),
		ShutdownTimeout:     durationFromEnv("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		ProcessingInterval:  durationFromEnv("JOB_PROCESSING_INTERVAL", 2*time.Second),
		JobRecoveryInterval: durationFromEnv("JOB_RECOVERY_INTERVAL", 10*time.Minute),
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
