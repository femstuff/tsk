package service

import (
	"fmt"
	"log"
	"tg-tsk-bot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotService struct {
	bot               *tgbotapi.BotAPI
	messageRepo       repository.MessageRepo
	transcriptionServ TranscriptionService
	telegramToken     string
}

func NewBotService(bot *tgbotapi.BotAPI, messageRepo repository.MessageRepo, token string) *BotService {
	return &BotService{
		bot:           bot,
		messageRepo:   messageRepo,
		telegramToken: token,
	}
}

func (s *BotService) ProcessVoice(voiceID string, chatID int64) error {
	s.bot.Send(tgbotapi.NewMessage(chatID, "Было получено голосове сообщение. Обработка началась..."))

	go func() {
		filePath, err := s.downloadVoice(voiceID, chatID)
		if err != nil {
			return
		}

		// text, err := s. надо дописать как будем отправлять во внещний сервис для транскрибации
		text, err := s.transcriptionServ.SendTest(filePath)

		s.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Что в голосовом сообщении: \n%s", text)))
	}()
	return nil
}

func (s *BotService) ProcessText(text string, chatID int64) error {
	resp := fmt.Sprintf("Я короче пока туда сюда не ебу как обрабатывать текст.🦡🦡🦡🚽🚽🚽 Тест текста: %s", text)
	msg := tgbotapi.NewMessage(chatID, resp)
	_, err := s.bot.Send(msg)
	return err
}

func (s *BotService) downloadVoice(fileID string, chatID int64) (string, error) {
	file, err := s.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", s.telegramToken, file.FilePath)

	log.Printf("File tipo download from %s", url)
	return "name-gs", nil
}
