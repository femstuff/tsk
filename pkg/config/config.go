package config

import (
	"os"
)

type Config struct {
	TelegramToken string
	ServerPort    string
	DownDir       string
}

func Load() *Config {
	return &Config{
		TelegramToken: getEnv("TG_TOKEN", ""),
		ServerPort:    getEnv("SERV_PORT", "8080"),
		DownDir:       getEnv("DIRECTORY", "./downloads"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
