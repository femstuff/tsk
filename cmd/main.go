package main

import (
	"tg-tsk-bot/internal/repository"
	"tg-tsk-bot/internal/service"
	"tg-tsk-bot/pkg/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg := config.Load()

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return
	}

	//trans := service.NewTranscriptionService(cfg.TelegramToken) // endpoint python serv nushen
	msgRepo := repository.NewInMemoryMessage()
	svc := service.NewBotService(bot, msgRepo, cfg.TelegramToken)

}
