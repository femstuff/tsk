package main

import (
	"log"
	"tg-tsk-bot/internal/handler"
	"tg-tsk-bot/internal/repository"
	"tg-tsk-bot/internal/service"
	"tg-tsk-bot/pkg/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg := config.Load()
	log.Println("Конфиг встау")
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("\nБот упау ERROR: %s", err)
	}

	//trans := service.NewTranscriptionService(cfg.TelegramToken) // endpoint python serv nushen
	msgRepo := repository.NewInMemoryMessage()
	svc := service.NewBotService(bot, msgRepo, cfg.TelegramToken)

	log.Println("Сервисы встали")

	tgHand := handler.NewTelegramHandler(svc)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	tgHand.Updates(updates)
}
