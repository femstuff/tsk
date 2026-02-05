package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"tg-tsk-bot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotService struct {
	bot               *tgbotapi.BotAPI
	messageRepo       repository.MessageRepo
	transcriptionServ *TranscriptionService
	telegramToken     string
}

func NewBotService(bot *tgbotapi.BotAPI, messageRepo repository.MessageRepo, token string, transcriptionServ *TranscriptionService) *BotService {
	return &BotService{
		bot:               bot,
		messageRepo:       messageRepo,
		telegramToken:     token,
		transcriptionServ: transcriptionServ,
	}
}

func (s *BotService) ProcessVoice(voiceID string, chatID int64) error {
	s.bot.Send(tgbotapi.NewMessage(chatID, " Голосовое получено. Начинаю обработку..."))

	go func() {
		file, err := s.bot.GetFile(tgbotapi.FileConfig{FileID: voiceID})
		if err != nil {
			slog.Error("Ошибка получения файла", "error", err)
			s.bot.Send(tgbotapi.NewMessage(chatID, " Ошибка при получении файла от Telegram"))
			return
		}

		url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", s.telegramToken, file.FilePath)
		slog.Debug("Скачиваем файл", "url", url)

		resp, err := http.Get(url)
		if err != nil {
			slog.Error("Ошибка скачивания", "error", err)
			s.bot.Send(tgbotapi.NewMessage(chatID, " Ошибка при скачивании файла"))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slog.Error("Ошибка HTTP", "status", resp.Status)
			s.bot.Send(tgbotapi.NewMessage(chatID, " Ошибка сервера Telegram"))
			return
		}

		audioData, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("Ошибка чтения файла", "error", err)
			s.bot.Send(tgbotapi.NewMessage(chatID, " Ошибка при чтении файла"))
			return
		}

		slog.Debug("Файл скачан", "bytes", len(audioData))

		text, err := s.sendToWhisperAPI(audioData, "voice.oga")
		if err != nil {
			slog.Error("Ошибка API", "error", err)
			s.bot.Send((tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка транскрипции. Повторите позднее. %s", err))))
			return
		}

		slog.Info("Транскрипция завершена", "text_length", len(text))
		s.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf(" Результат:\n%s", text)))
	}()
	return nil
}

func (s *BotService) sendToWhisperAPI(audioData []byte, filename string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("ошибка создания формы: %v", err)
	}

	_, err = part.Write(audioData)
	if err != nil {
		return "", fmt.Errorf("ошибка записи данных: %v", err)
	}

	writer.Close()

	apiURL := s.transcriptionServ.api + "/transcribe"
	slog.Debug("Отправляю на API", "url", apiURL, "bytes", len(audioData))

	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка отправки на API: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	slog.Debug("Ответ от API", "bytes", len(respBody))

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return "", fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	if text, ok := result["text"].(string); ok {
		return text, nil
	}

	return "", fmt.Errorf("неверный формат ответа")
}

func (s *BotService) ProcessText(text string, chatID int64) error {
	slog.Debug("Обработка текста", "text", text, "chat_id", chatID)

	resp := fmt.Sprintf(" Текст получен: %s", text)
	msg := tgbotapi.NewMessage(chatID, resp)
	_, err := s.bot.Send(msg)

	if err != nil {
		slog.Error("Ошибка отправки сообщения", "error", err)
	}

	return err
}
