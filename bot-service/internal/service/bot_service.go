package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
			log.Printf("Ошибка получения файла: %v", err)
			s.bot.Send(tgbotapi.NewMessage(chatID, " Ошибка при получении файла от Telegram"))
			return
		}

		url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", s.telegramToken, file.FilePath)
		log.Printf("Скачиваем файл: %s", url)

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Ошибка скачивания: %v", err)
			s.bot.Send(tgbotapi.NewMessage(chatID, " Ошибка при скачивании файла"))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Ошибка HTTP: %s", resp.Status)
			s.bot.Send(tgbotapi.NewMessage(chatID, " Ошибка сервера Telegram"))
			return
		}

		audioData, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Ошибка чтения файла: %v", err)
			s.bot.Send(tgbotapi.NewMessage(chatID, " Ошибка при чтении файла"))
			return
		}

		log.Printf("Файл скачан (%d байт), отправляю на Python API...", len(audioData))

		text, err := s.sendToWhisperAPI(audioData, "voice.oga", chatID)
		if err != nil {
			log.Printf("Ошибка API: %v", err)
			text = " Ошибка транскрипции. Тестовый текст: ПРИВЕТ ЭТО ТЕСТ"
		}

		s.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf(" Результат:\n%s", text)))
	}()
	return nil
}

func (s *BotService) sendToWhisperAPI(audioData []byte, filename string, chatID int64) (string, error) {
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
	log.Printf("Отправляю на %s", apiURL)

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

	log.Printf("Ответ от API (%d байт): %s", len(respBody), string(respBody))

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return "", fmt.Errorf("ошибка парсинга JSON: %v. Ответ: %s", err, string(respBody))
	}

	if text, ok := result["text"].(string); ok {
		return text, nil
	}

	return "", fmt.Errorf("неверный формат ответа: %v", result)
}

func (s *BotService) ProcessText(text string, chatID int64) error {
	resp := fmt.Sprintf(" Текст получен: %s", text)
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
