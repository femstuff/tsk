package main

import (
	"log/slog"
	"net/http"
	"tg-tsk-bot/internal/handler"
	"tg-tsk-bot/internal/repository"
	"tg-tsk-bot/internal/service"
	"tg-tsk-bot/pkg/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg := config.Load()

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		slog.Error("Ошибка создания бота", "error", err)
		return
	}

	slog.Info("Бот авторизован", "username", bot.Self.UserName)

	httpClient := &http.Client{}

	trans := service.NewTranscriptionService(cfg.WhisperAPIURL)
	msgRepo := repository.NewInMemoryMessage()

	var bitrixService *service.BitrixService
	if cfg.BitrixWebhookURL != "" {
		bitrixService = service.NewBitrixService(cfg.BitrixWebhookURL, httpClient)
		slog.Info("Bitrix сервис инициализирован")
	} else {
		slog.Warn("Bitrix не настроен (BITRIX_WEBHOOK_URL пуст)")
	}

	var decisionService *service.DecisionService
	if cfg.LLMAPIKey != "" && cfg.LLMAPIURL != "" {
		decisionService = service.NewDecisionService(
			cfg.LLMAPIURL,
			cfg.LLMAPIKey,
			cfg.LLMModel,
			cfg.DefaultStageID,
			httpClient,
		)
		slog.Info("Decision сервис инициализирован", "model", cfg.LLMModel)
	} else {
		slog.Warn("LLM не настроен (LLM_API_KEY или LLM_API_URL пуст)")
	}

	svc := service.NewBotService(bot, msgRepo, cfg.TelegramToken, trans, decisionService, bitrixService)

	tgHand := handler.NewTelegramHandler(svc)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	slog.Info("Бот запущен, слушает канал")

	tgHand.Updates(updates)
}
