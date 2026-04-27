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

	// Bitrix24
	BitrixWebhookURL string `mapstructure:"BITRIX_WEBHOOK_URL"`

	// LLM
	LLMAPIURL      string `mapstructure:"LLM_API_URL"`
	LLMAPIKey      string `mapstructure:"LLM_API_KEY"`
	LLMModel       string `mapstructure:"LLM_MODEL"`
	DefaultStageID string `mapstructure:"DEFAULT_STAGE_ID"`
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

	// Дефолтные значения
	if cfg.WhisperAPIURL == "" {
		cfg.WhisperAPIURL = "http://whisper-api:8000"
	}

	if cfg.LLMAPIURL == "" {
		cfg.LLMAPIURL = "https://openrouter.ai/api/v1/chat/completions"
	}

	if cfg.LLMModel == "" {
		cfg.LLMModel = "gpt-4o-mini"
	}

	if cfg.DefaultStageID == "" {
		cfg.DefaultStageID = "C1:NEW"
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
		"bitrix", cfg.BitrixWebhookURL != "",
		"llm", cfg.LLMAPIKey != "",
		"port", cfg.ServerPort,
		"directory", cfg.DownloadDir,
	)

	return &cfg
}
