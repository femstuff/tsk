package main

import (
	"tg-tsk-bot/internal/repository"
	"tg-tsk-bot/pkg/config"
)

func main() {
	cfg := config.Load()

	msgRepo := repository.NewInMemoryMessage()
}
