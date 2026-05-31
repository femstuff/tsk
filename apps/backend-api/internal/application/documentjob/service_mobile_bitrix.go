package documentjob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"tsk/backend-api/internal/application/voiceintent"
	platform "tsk/backend-api/internal/domain/platform"
	domain "tsk/backend-api/internal/domain/documentjob"
	"tsk/backend-api/internal/infrastructure/cache"
	"tsk/backend-api/internal/integrations/bitrixclient"
)

// MobileBitrixIntentInput — текст и/или голос + подсказки по сделке/стадии (без заявки на документ).
type MobileBitrixIntentInput struct {
	Text              string
	Audio             []byte
	FileName          string
	MimeType          string
	DealIDOverride    int
	DealTitleOverride string
	DealHintText      string
	StageHintText     string
}

// MobileBitrixIntentResult — результат разбора и вызовов Bitrix24.
type MobileBitrixIntentResult struct {
	Transcript       string   `json:"transcript"`
	ParsedAction     string   `json:"parsedAction"`
	ParsedDealID     int      `json:"parsedDealId"`
	ParsedDealTitle  string   `json:"parsedDealTitle"`
	BitrixSteps      []string `json:"bitrixSteps"`
	BitrixConfigured bool     `json:"bitrixConfigured"`
}

func (s *Service) auditMobileBitrixIntent(ctx context.Context, res MobileBitrixIntentResult, err error) {
	if s == nil || s.events == nil {
		return
	}
	level := "info"
	eventType := "mobile.bitrix_intent.ok"
	msg := "Мобильное приложение: Bitrix — выполнено"
	if err != nil {
		level = "warning"
		eventType = "mobile.bitrix_intent.error"
		msg = "Мобильное приложение: Bitrix — ошибка"
	}
	details := map[string]any{
		"transcript":      truncateForEvent(res.Transcript, 4000),
		"parsedAction":    res.ParsedAction,
		"parsedDealId":    res.ParsedDealID,
		"parsedDealTitle": res.ParsedDealTitle,
		"bitrixSteps":     res.BitrixSteps,
		"bitrixConfigured": res.BitrixConfigured,
	}
	if err != nil {
		details["error"] = err.Error()
	}
	raw, _ := json.Marshal(details)
	if _, e := s.createEvent(ctx, nil, level, eventType, msg, string(raw)); e != nil {
		return
	}
}

// RunMobileBitrixIntent: опционально Whisper → разбор текста → действия CRM/задачи в Bitrix (без document job).
func (s *Service) RunMobileBitrixIntent(ctx context.Context, in MobileBitrixIntentInput) (res MobileBitrixIntentResult, err error) {
	defer func() {
		s.auditMobileBitrixIntent(ctx, res, err)
	}()

	transcript := strings.TrimSpace(in.Text)
	if len(in.Audio) > 0 {
		if s.whisper == nil {
			err = fmt.Errorf("whisper is not configured (set WHISPER_BASE_URL)")
			return res, err
		}
		fn := sanitizeFileName(in.FileName, "mobile-voice.m4a")
		mt := fallbackString(strings.TrimSpace(in.MimeType), "audio/mp4")
		t, trErr := s.whisper.Transcribe(ctx, fn, mt, in.Audio)
		if trErr != nil {
			if s.metrics != nil {
				s.metrics.RecordError("whisper_transcribe")
			}
			err = fmt.Errorf("transcription: %w", trErr)
			return res, err
		}
		transcript = strings.TrimSpace(t)
	}
	if transcript == "" {
		err = fmt.Errorf("укажите текст запроса или приложите аудио")
		return res, err
	}

	var persistedSourceID *string
	if len(in.Audio) > 0 {
		fn := sanitizeFileName(in.FileName, "mobile-voice.m4a")
		mt := fallbackString(strings.TrimSpace(in.MimeType), "audio/mp4")
		storedFile, saveErr := s.storage.Save(ctx, "source", fn, in.Audio)
		if saveErr == nil {
			src, docErr := s.sourceDocuments.CreateSourceDocument(ctx, domain.SourceDocumentCreateParams{
				ID:         "src-" + uuid.NewString(),
				Kind:       domain.SourceDocumentKindVoiceRecording,
				Origin:     "mobile-app-bitrix",
				FileName:   storedFile.FileName,
				MimeType:   mt,
				StorageKey: storedFile.StorageKey,
				SizeBytes:  storedFile.SizeBytes,
				CreatedAt:  s.now().UTC(),
			})
			if docErr == nil {
				persistedSourceID = &src.ID
				s.registerStoredFile(ctx, "source", storedFile, platform.EntityTypeSourceDocument, src.ID, in.Audio)
			}
		}
	}

	s.saveTranscription(ctx, platform.TranscriptionSourceMobileBitrixIntent, transcript, nil, persistedSourceID)

	adminHints := AdminVoiceBitrixInput{
		DealIDOverride:    in.DealIDOverride,
		DealTitleOverride: in.DealTitleOverride,
		DealHintText:      in.DealHintText,
		StageHintText:     in.StageHintText,
	}
	intent := voiceintent.Parse(transcript, adminHints.DealIDOverride)
	if strings.TrimSpace(adminHints.DealTitleOverride) != "" && strings.TrimSpace(intent.DealTitle) == "" {
		intent.DealTitle = strings.TrimSpace(adminHints.DealTitleOverride)
	}
	applyAdminVoiceTextHints(&intent, adminHints)

	res = MobileBitrixIntentResult{
		Transcript:       transcript,
		ParsedAction:     intent.Action,
		ParsedDealID:     intent.DealID,
		ParsedDealTitle:  strings.TrimSpace(intent.DealTitle),
		BitrixSteps:      []string{},
		BitrixConfigured: s.bitrix != nil && s.bitrix.WebhookConfigured(),
	}

	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		res.BitrixSteps = append(res.BitrixSteps, "Bitrix: вебхук не задан — действия не выполнялись")
		err = fmt.Errorf("BITRIX_WEBHOOK_URL не настроен на сервере")
		return res, err
	}

	switch intent.Action {
	case voiceintent.ActionCreateTask:
		taskID, addErr := s.bitrix.AddTask(ctx, intent.TaskTitle, "TSK: запрос из мобильного приложения\n\n"+transcript)
		if addErr != nil {
			res.BitrixSteps = append(res.BitrixSteps, "Задача: ошибка — "+addErr.Error())
			err = fmt.Errorf("создание задачи Bitrix: %w", addErr)
			return res, err
		}
		res.BitrixSteps = append(res.BitrixSteps, fmt.Sprintf("Задача создана, id=%s", taskID))
		return res, nil

	case voiceintent.ActionMoveNext, voiceintent.ActionMovePrev, voiceintent.ActionMoveStage:
		dealID, matchedTitle, resolveSteps := s.resolveDealIDForBitrix(ctx, intent, adminHints)
		res.BitrixSteps = append(res.BitrixSteps, resolveSteps...)
		if matchedTitle != "" {
			res.ParsedDealTitle = matchedTitle
		}
		if dealID <= 0 {
			res.BitrixSteps = append(res.BitrixSteps,
				"Сделка: не найден ID — укажите dealId, название сделки в форме или в тексте запроса")
			err = errors.New("сделка не найдена по ID или названию")
			return res, err
		}
		res.ParsedDealID = dealID

		var dir bitrixclient.MoveDirection
		var stageArg string
		switch intent.Action {
		case voiceintent.ActionMoveNext:
			dir = bitrixclient.MoveDirectionNext
		case voiceintent.ActionMovePrev:
			dir = bitrixclient.MoveDirectionPrev
		default:
			dir = bitrixclient.MoveDirectionAuto
			resolved, stErr := s.bitrix.ResolveStageID(ctx, dealID, intent.StageHint)
			if stErr != nil {
				res.BitrixSteps = append(res.BitrixSteps, "Сделка: не удалось определить стадию — "+stErr.Error())
				err = fmt.Errorf("стадия Bitrix: %w", stErr)
				return res, err
			}
			stageArg = resolved
		}

		newDealID, newStage, mvErr := s.bitrix.MoveDealToStage(ctx, dealID, stageArg, dir)
		if mvErr != nil {
			res.BitrixSteps = append(res.BitrixSteps, "Сделка: ошибка — "+mvErr.Error())
			err = fmt.Errorf("перенос сделки Bitrix: %w", mvErr)
			return res, err
		}
		res.BitrixSteps = append(res.BitrixSteps, fmt.Sprintf("Сделка #%d переведена на стадию %s", newDealID, newStage))
		return res, nil

	default:
		res.BitrixSteps = append(res.BitrixSteps, "Действие Bitrix не распознано (ожидаются: следующий этап, назад, стадия …, создай задачу …)")
		err = errors.New("действие Bitrix не распознано по тексту запроса")
		return res, err
	}
}

// BitrixMobileTaskStats — агрегаты по открытым задачам ответственного (в пределах limit записей из Bitrix).
type BitrixMobileTaskStats struct {
	TotalOpen  int `json:"totalOpen"`
	InProgress int `json:"inProgress"`
	Overdue    int `json:"overdue"`
}

// BitrixMobileTasksResponse — список задач + счётчики для главного экрана мобильного приложения.
type BitrixMobileTasksResponse struct {
	ResponsibleUserID int                       `json:"responsibleUserId"`
	Stats             BitrixMobileTaskStats     `json:"stats"`
	Items             []bitrixclient.BitrixTaskBrief `json:"items"`
	AuthMode          string                    `json:"authMode,omitempty"`
}

func bitrixTaskClosed(t bitrixclient.BitrixTaskBrief) bool {
	if strings.TrimSpace(t.ClosedDate) != "" {
		return true
	}
	switch strings.TrimSpace(t.Status) {
	case "4", "5", "7": // завершена / отклонена (коды Bitrix24)
		return true
	default:
		return false
	}
}

func parseBitrixDeadline(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	loc := time.Local
	for _, layout := range []string{"02.01.2006 15:04:05", "02.01.2006 15:04", "02.01.2006", time.RFC3339} {
		if tt, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return tt, true
		}
	}
	return time.Time{}, false
}

func computeBitrixTaskStats(items []bitrixclient.BitrixTaskBrief, now time.Time) BitrixMobileTaskStats {
	var st BitrixMobileTaskStats
	for _, t := range items {
		if bitrixTaskClosed(t) {
			continue
		}
		st.TotalOpen++
		if strings.TrimSpace(t.Status) == "3" {
			st.InProgress++
		}
		if dl, ok := parseBitrixDeadline(t.Deadline); ok && now.After(dl) {
			st.Overdue++
		}
	}
	return st
}

func bitrixTaskStatusLabel(status string, closed bool) string {
	if closed {
		return "Завершена"
	}
	switch strings.TrimSpace(status) {
	case "2":
		return "Ждёт выполнения"
	case "3":
		return "В работе"
	case "4":
		return "Завершена"
	case "5", "6":
		return "Отложена"
	case "7":
		return "Отклонена"
	default:
		if status == "" {
			return "Неизвестно"
		}
		return "Статус " + status
	}
}

func bitrixTaskMatchesFilter(t bitrixclient.BitrixTaskBrief, filter string, now time.Time) bool {
	closed := bitrixTaskClosed(t)
	switch filter {
	case "completed":
		return closed
	case "in_progress":
		return !closed && strings.TrimSpace(t.Status) == "3"
	case "overdue":
		if closed {
			return false
		}
		dl, ok := parseBitrixDeadline(t.Deadline)
		return ok && now.After(dl)
	case "open":
		return !closed
	default:
		return false
	}
}

func mapAdminBitrixTaskItems(items []bitrixclient.BitrixTaskBrief) []AdminBitrixTaskItemView {
	out := make([]AdminBitrixTaskItemView, 0, len(items))
	for _, task := range items {
		closed := bitrixTaskClosed(task)
		out = append(out, AdminBitrixTaskItemView{
			ID:          task.ID,
			Title:       task.Title,
			Status:      task.Status,
			StatusLabel: bitrixTaskStatusLabel(task.Status, closed),
			Deadline:    task.Deadline,
			ClosedDate:  task.ClosedDate,
		})
	}
	return out
}

// ListBitrixTasksForMobile — задачи ответственного: OAuth-сессия пользователя или вебхук (fallback).
func (s *Service) ListBitrixTasksForMobile(ctx context.Context, limit int, responsibleOverride int, oauthSessionID string, skipCache bool) (BitrixMobileTasksResponse, error) {
	var out BitrixMobileTasksResponse
	oauthSessionID = strings.TrimSpace(oauthSessionID)

	if oauthSessionID != "" && s.BitrixOAuthEnabled() {
		session, sessionErr := s.ensureActiveBitrixSession(ctx, oauthSessionID)
		if sessionErr != nil {
			return out, sessionErr
		}

		tokenClient := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)
		if session.BitrixUserID <= 0 {
			return out, errors.New("не удалось определить пользователя Bitrix для сессии")
		}
		rid := session.BitrixUserID
		if responsibleOverride > 0 {
			rid = responsibleOverride
		}

		cacheKey := cache.BitrixTasksKey(rid, limit) + ":oauth:v3:" + oauthSessionID
		if !skipCache && s.cache != nil {
			if payload, ok, err := s.cache.Get(ctx, cacheKey); err == nil && ok {
				if json.Unmarshal(payload, &out) == nil {
					return out, nil
				}
			}
		}

		items, err := tokenClient.ListTasksForUser(ctx, rid, limit)
		if err != nil {
			return out, bitrixclient.TasksListUserError(err)
		}
		out.ResponsibleUserID = rid
		out.Items = items
		out.Stats = computeBitrixTaskStats(items, s.now().In(time.Local))
		out.AuthMode = "oauth"

		if s.cache != nil {
			if payload, err := json.Marshal(out); err == nil {
				_ = s.cache.Set(ctx, cacheKey, payload, cacheBitrixTasksTTL)
			}
		}
		return out, nil
	}

	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		if s.BitrixOAuthEnabled() {
			return out, fmt.Errorf("войдите в Bitrix24 в приложении или настройте BITRIX_WEBHOOK_URL на сервере")
		}
		return out, fmt.Errorf("BITRIX_WEBHOOK_URL не настроен на сервере")
	}
	rid := responsibleOverride
	if rid <= 0 {
		var err error
		rid, err = s.bitrix.WebhookOwnerUserID()
		if err != nil {
			return out, fmt.Errorf("не удалось определить пользователя вебхука: %w; укажите query responsibleId=…", err)
		}
	}

	cacheKey := cache.BitrixTasksKey(rid, limit)
	if !skipCache && s.cache != nil {
		if payload, ok, err := s.cache.Get(ctx, cacheKey); err == nil && ok {
			if json.Unmarshal(payload, &out) == nil {
				return out, nil
			}
		}
	}

	items, err := s.bitrix.ListTasksForResponsible(ctx, rid, limit)
	if err != nil {
		return out, err
	}
	out.ResponsibleUserID = rid
	out.Items = items
	out.Stats = computeBitrixTaskStats(items, s.now().In(time.Local))
	out.AuthMode = "webhook"

	if s.cache != nil {
		if payload, err := json.Marshal(out); err == nil {
			_ = s.cache.Set(ctx, cacheKey, payload, cacheBitrixTasksTTL)
		}
	}

	return out, nil
}

func (s *Service) invalidateBitrixTaskListCache(ctx context.Context, rid, limit int, oauthSessionID string) {
	if s.cache == nil {
		return
	}
	oauthSessionID = strings.TrimSpace(oauthSessionID)
	if oauthSessionID != "" {
		_ = s.cache.Delete(ctx, cache.BitrixTasksKey(rid, limit)+":oauth:v3:"+oauthSessionID)
	}
	_ = s.cache.Delete(ctx, cache.BitrixTasksKey(rid, limit))
}

// GetBitrixTaskForMobile — карточка задачи (описание и поля).
func (s *Service) GetBitrixTaskForMobile(ctx context.Context, taskID, oauthSessionID string) (bitrixclient.BitrixTaskDetail, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return bitrixclient.BitrixTaskDetail{}, errors.New("task id is required")
	}
	oauthSessionID = strings.TrimSpace(oauthSessionID)

	if oauthSessionID != "" && s.BitrixOAuthEnabled() {
		session, err := s.ensureActiveBitrixSession(ctx, oauthSessionID)
		if err != nil {
			return bitrixclient.BitrixTaskDetail{}, err
		}
		client := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)
		detail, err := client.GetTask(ctx, taskID)
		if err != nil {
			return bitrixclient.BitrixTaskDetail{}, bitrixclient.TasksListUserError(err)
		}
		return detail, nil
	}

	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		if s.BitrixOAuthEnabled() {
			return bitrixclient.BitrixTaskDetail{}, fmt.Errorf("войдите в Bitrix24 в приложении или настройте BITRIX_WEBHOOK_URL на сервере")
		}
		return bitrixclient.BitrixTaskDetail{}, fmt.Errorf("BITRIX_WEBHOOK_URL не настроен на сервере")
	}

	detail, err := s.bitrix.GetTask(ctx, taskID)
	if err != nil {
		return bitrixclient.BitrixTaskDetail{}, err
	}
	return detail, nil
}

// UpdateBitrixTaskStatusForMobile — смена статуса задачи и возврат обновлённой карточки.
func (s *Service) UpdateBitrixTaskStatusForMobile(ctx context.Context, taskID string, status int, oauthSessionID string) (bitrixclient.BitrixTaskDetail, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return bitrixclient.BitrixTaskDetail{}, errors.New("task id is required")
	}
	if status <= 0 {
		return bitrixclient.BitrixTaskDetail{}, errors.New("status is required")
	}
	oauthSessionID = strings.TrimSpace(oauthSessionID)

	if oauthSessionID != "" && s.BitrixOAuthEnabled() {
		session, err := s.ensureActiveBitrixSession(ctx, oauthSessionID)
		if err != nil {
			return bitrixclient.BitrixTaskDetail{}, err
		}
		client := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)
		if err := client.UpdateTaskStatus(ctx, taskID, status); err != nil {
			return bitrixclient.BitrixTaskDetail{}, bitrixclient.TasksListUserError(err)
		}
		s.invalidateBitrixTaskListCache(ctx, session.BitrixUserID, 80, oauthSessionID)
		s.invalidateBitrixTaskListCache(ctx, session.BitrixUserID, 60, oauthSessionID)
		detail, err := client.GetTask(ctx, taskID)
		if err != nil {
			return bitrixclient.BitrixTaskDetail{}, bitrixclient.TasksListUserError(err)
		}
		return detail, nil
	}

	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		if s.BitrixOAuthEnabled() {
			return bitrixclient.BitrixTaskDetail{}, fmt.Errorf("войдите в Bitrix24 в приложении или настройте BITRIX_WEBHOOK_URL на сервере")
		}
		return bitrixclient.BitrixTaskDetail{}, fmt.Errorf("BITRIX_WEBHOOK_URL не настроен на сервере")
	}

	if err := s.bitrix.UpdateTaskStatus(ctx, taskID, status); err != nil {
		return bitrixclient.BitrixTaskDetail{}, err
	}
	rid, _ := s.bitrix.WebhookOwnerUserID()
	s.invalidateBitrixTaskListCache(ctx, rid, 80, "")
	s.invalidateBitrixTaskListCache(ctx, rid, 60, "")
	detail, err := s.bitrix.GetTask(ctx, taskID)
	if err != nil {
		return bitrixclient.BitrixTaskDetail{}, err
	}
	return detail, nil
}
