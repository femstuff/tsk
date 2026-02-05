package main

import (
	"log/slog"
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

	trans := service.NewTranscriptionService(cfg.WhisperAPIURL)
	msgRepo := repository.NewInMemoryMessage()
	svc := service.NewBotService(bot, msgRepo, cfg.TelegramToken, trans)

	tgHand := handler.NewTelegramHandler(svc)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	slog.Info("Бот запущен, слушает канал")

	tgHand.Updates(updates)
}
