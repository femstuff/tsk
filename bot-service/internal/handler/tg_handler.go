package handler

import (
	"log"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramHandler struct {
	botService interface {
		ProcessVoice(voiceFileID string, chatID int64) error
		ProcessText(text string, chatID int64) error
	}
}

func NewTelegramHandler(botService interface {
	ProcessVoice(voiceFileID string, chatID int64) error
	ProcessText(text string, chatID int64) error
}) *TelegramHandler {
	return &TelegramHandler{
		botService: botService,
	}
}

func (h *TelegramHandler) Updates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Ошибка обработки сообщения: %v", r)
				}
			}()

			if update.Message == nil {
				return
			}

			msg := update.Message

			if msg.Voice != nil {
				chatID := msg.Chat.ID
				fileID := msg.Voice.FileID

				slog.Info("ГОЛОСОВОЕ СООБЩЕНИЕ",
					"chat_id", chatID,
					"voice_id", fileID,
				)
				go h.botService.ProcessVoice(fileID, chatID)
				return
			}

			if msg.Text != "" {
				chatID := msg.Chat.ID
				text := msg.Text
				slog.Info("ТЕКСТОВОЕ СООБЩЕНИЕ",
					"chat_id", chatID,
					"text", msg.Text)
				go h.botService.ProcessText(text, chatID)
			}
		}()
	}
}
