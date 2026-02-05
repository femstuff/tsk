package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	ServerPort    string
	DownDir       string
	WhisperAPIURL string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Print("ошибка в загрузке енв")
	}

	return &Config{
		TelegramToken: getEnv("TG_TOKEN", ""),
		ServerPort:    getEnv("SERV_PORT", "8080"),
		DownDir:       getEnv("DIRECTORY", "./downloads"),
		WhisperAPIURL: getEnv("WHISPER_API_URL", "http://localhost:8000"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	} else {
		log.Printf("ХУЕТА ПУСТАЯ: %s", key)
	}
	return defaultValue
}
