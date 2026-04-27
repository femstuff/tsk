package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"tg-tsk-bot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotService struct {
	bot               *tgbotapi.BotAPI
	messageRepo       repository.MessageRepo
	telegramToken     string
	transcriptionServ *TranscriptionService
	decisionServ      *DecisionService
	bitrixServ        *BitrixService
	pendingMu         sync.RWMutex
	pendingDecisions  map[int64]*pendingDecision
}

type pendingDecision struct {
	OriginalText string
	Direction    string
	TargetStage  string
	Decision     Decision
}

func NewBotService(
	bot *tgbotapi.BotAPI,
	messageRepo repository.MessageRepo,
	token string,
	transcriptionServ *TranscriptionService,
	decisionServ *DecisionService,
	bitrixServ *BitrixService,
) *BotService {
	return &BotService{
		bot:               bot,
		messageRepo:       messageRepo,
		telegramToken:     token,
		transcriptionServ: transcriptionServ,
		decisionServ:      decisionServ,
		bitrixServ:        bitrixServ,
		pendingDecisions:  make(map[int64]*pendingDecision),
	}
}

func (s *BotService) ProcessVoice(voiceID string, chatID int64) error {
	log.Printf("Голосовое получено от chat %d", chatID)
	s.bot.Send(tgbotapi.NewMessage(chatID, "Голосовое получено. Начинаю обработку..."))

	go func() {
		file, err := s.bot.GetFile(tgbotapi.FileConfig{FileID: voiceID})
		if err != nil {
			log.Printf("Ошибка получения файла: %v", err)
			s.bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении файла от Telegram"))
			return
		}

		url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", s.telegramToken, file.FilePath)
		log.Printf("Скачиваем файл: %s", url)

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Ошибка скачивания: %v", err)
			s.bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при скачивании файла"))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Ошибка HTTP: %d", resp.StatusCode)
			s.bot.Send(tgbotapi.NewMessage(chatID, "Ошибка сервера Telegram"))
			return
		}

		audioData, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Ошибка чтения файла: %v", err)
			s.bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при чтении файла"))
			return
		}

		log.Printf("Файл скачан, размер: %d байт", len(audioData))

		text, err := s.sendToWhisperAPI(audioData, "voice.oga")
		if err != nil {
			log.Printf("Ошибка транскрипции: %v", err)
			s.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка транскрипции: %v", err)))
			return
		}

		log.Printf("Транскрипция: %s", text)

		response := fmt.Sprintf("Распознанный текст:\n%s", text)

		if s.decisionServ != nil {
			log.Printf("Запускаем LLM анализ")
			decision, err := s.decisionServ.Analyze(text)
			if err != nil {
				log.Printf("Ошибка LLM анализа: %v", err)
				response += fmt.Sprintf("\n\nОшибка анализа: %v", err)
				s.bot.Send(tgbotapi.NewMessage(chatID, response))
				return
			}

			response += formatDecision(decision)

			if decision.ShouldMove && s.bitrixServ != nil {
				log.Printf("Пытаемся переместить сделку")
				dealID := decision.DealID

				if dealID == 0 && decision.DealTitle != "" {
					log.Printf("Ищем сделку по названию: %s", decision.DealTitle)
					foundID, err := s.bitrixServ.FindDealIDByTitle(decision.DealTitle)
					if err == nil && foundID > 0 {
						dealID = foundID
						log.Printf("Найдена сделка: %d", dealID)
					} else {
						log.Printf("Сделка не найдена: %v", err)

						// Сохраняем запрос для уточнения
						s.pendingMu.Lock()
						s.pendingDecisions[chatID] = &pendingDecision{
							OriginalText: text,
							Direction:    decision.Direction,
							TargetStage:  decision.GetTargetStage(),
							Decision:     decision,
						}
						s.pendingMu.Unlock()

						// Отправляем сообщение об ошибке
						msgText := fmt.Sprintf("Сделка с названием \"%s\" не найдена в Bitrix24.\n\nВы имели в виду: %s\n\nЕсли название верное - напишите его еще раз текстом, я повторю поиск.\n\nЕсли название неверное - напишите точное название сделки как она называется в Bitrix24.\n\nПример: ТЕСТОВЫЙ ПРОЕКТ 2024\n\nИли отправьте /cancel чтобы отменить операцию.", decision.DealTitle, decision.DealTitle)

						s.bot.Send(tgbotapi.NewMessage(chatID, msgText))
						return
					}
				}

				if dealID > 0 {
					direction := decision.Direction
					if direction == "" {
						direction = "next"
					}

					targetStage := decision.GetTargetStage()
					log.Printf("Перемещаем сделку %d с направлением: %s, целевая стадия: %s", dealID, direction, targetStage)

					var moveDirection MoveDirection
					var targetStageForMove string

					switch direction {
					case "prev":
						moveDirection = MoveDirectionPrev
					case "to_stage":
						moveDirection = MoveDirectionAuto
						targetStageForMove = targetStage
					default:
						moveDirection = MoveDirectionNext
					}

					movedDealID, newStage, err := s.bitrixServ.MoveDealToStage(dealID, targetStageForMove, moveDirection)
					if err != nil {
						log.Printf("Ошибка перемещения: %v", err)
						response += fmt.Sprintf("\n\nОшибка Bitrix: %v", err)
					} else {
						log.Printf("Сделка перемещена! ID=%d, новая стадия=%s", movedDealID, newStage)
						response += fmt.Sprintf("\n\nСделка #%d перемещена в стадию: %s", movedDealID, newStage)
					}
				}
			} else {
				if !decision.ShouldMove {
					response += "\n\nℹРешение: перемещение не требуется"
				}
			}
		}

		s.bot.Send(tgbotapi.NewMessage(chatID, response))
	}()
	return nil
}

func (s *BotService) ProcessText(text string, chatID int64) error {
	log.Printf("Обработка текста: %s, chat_id: %d", text, chatID)

	// Проверка на отмену
	if strings.ToLower(text) == "/cancel" {
		s.pendingMu.Lock()
		_, exists := s.pendingDecisions[chatID]
		if exists {
			delete(s.pendingDecisions, chatID)
			s.pendingMu.Unlock()
			s.bot.Send(tgbotapi.NewMessage(chatID, "Операция отменена. Можете отправить новое голосовое сообщение."))
			return nil
		}
		s.pendingMu.Unlock()
	}

	// Проверяем, есть ли ожидающий запрос
	s.pendingMu.RLock()
	pending, exists := s.pendingDecisions[chatID]
	s.pendingMu.RUnlock()

	if exists {
		log.Printf("Найден pending запрос для chat %d", chatID)

		clarifiedTitle := strings.TrimSpace(text)

		s.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Ищу сделку \"%s\" в Bitrix24...", clarifiedTitle)))

		foundID, err := s.bitrixServ.FindDealIDByTitle(clarifiedTitle)
		if err != nil || foundID == 0 {
			log.Printf("Сделка не найдена: %v", err)
			msgText := fmt.Sprintf("Сделка \"%s\" не найдена.\n\nПроверьте название и попробуйте снова, или отправьте /cancel", clarifiedTitle)
			s.bot.Send(tgbotapi.NewMessage(chatID, msgText))
			return nil
		}

		log.Printf("Найдена сделка: ID=%d, Title=%s", foundID, clarifiedTitle)

		s.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Сделка найдена! ID: %d\nВыполняю перемещение...", foundID)))

		var moveDirection MoveDirection
		var targetStage string

		switch pending.Direction {
		case "prev":
			moveDirection = MoveDirectionPrev
		case "to_stage":
			moveDirection = MoveDirectionAuto
			targetStage = pending.TargetStage
		default:
			moveDirection = MoveDirectionNext
		}

		movedDealID, newStage, err := s.bitrixServ.MoveDealToStage(foundID, targetStage, moveDirection)
		if err != nil {
			log.Printf("Ошибка перемещения: %v", err)
			s.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка перемещения: %v", err)))
		} else {
			s.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Сделка #%d (%s) перемещена в стадию: %s", movedDealID, clarifiedTitle, newStage)))
		}

		// Удаляем pending запрос
		s.pendingMu.Lock()
		delete(s.pendingDecisions, chatID)
		s.pendingMu.Unlock()

		return nil
	}

	// Обычный текст
	resp := fmt.Sprintf("Текст получен: %s\n\nОтправьте голосовое сообщение для транскрипции и перемещения сделки.", text)
	s.bot.Send(tgbotapi.NewMessage(chatID, resp))

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

func formatDecision(decision Decision) string {
	result := "\n\nАнализ голового"
	result += fmt.Sprintf("\n- Переместить сделку: %v", decision.ShouldMove)
	result += fmt.Sprintf("\n- Направление: %s", decision.Direction)
	if decision.DealID > 0 {
		result += fmt.Sprintf("\n- ID сделки: %d", decision.DealID)
	}
	if decision.DealTitle != "" {
		result += fmt.Sprintf("\n- Название: %s", decision.DealTitle)
	}
	targetStage := decision.GetTargetStage()
	if targetStage != "" && targetStage != "0" {
		result += fmt.Sprintf("\n- Целевая стадия: %s", targetStage)
	}
	if decision.Reason != "" {
		result += fmt.Sprintf("\n- Причина: %s", decision.Reason)
	}
	return result
}
