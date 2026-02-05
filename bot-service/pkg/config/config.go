package config

import (
	"log/slog"
	"path/filepath"
	"runtime"
	"tg-tsk-bot/pkg/logger"

	"github.com/spf13/viper"
)

type Config struct {
	TelegramToken string `mapstructure:"TG_TOKEN"`
	WhisperAPIURL string `mapstructure:"WHISPER_API_URL"`
	ServerPort    string `mapstructure:"SERV_PORT"`
	DownloadDir   string `mapstructure:"DIRECTORY"`
}

func Load() *Config {
	if logger.Log == nil {
		logger.Init("debug")
	}

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	projectRoot := filepath.Join(dir, "..", "..")

	slog.Debug("Ищем конфиг",
		"project_root", projectRoot,
	)

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(projectRoot)
	viper.AddConfigPath(".")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		slog.Warn(".env не найден, использую переменные окружения", "error", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		slog.Error("Ошибка парсинга конфига", "error", err)
	}

	if cfg.TelegramToken == "" {
		slog.Error("TG_TOKEN не установлен")
	}

	tokenPreview := cfg.TelegramToken
	if len(tokenPreview) > 8 {
		tokenPreview = tokenPreview[:8] + "..."
	}

	slog.Info("Конфиг загружен",
		"token", tokenPreview,
		"whisper_url", cfg.WhisperAPIURL,
		"port", cfg.ServerPort,
		"directory", cfg.DownloadDir,
	)

	return &cfg
}
