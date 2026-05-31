package documentjob

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	domain "tsk/backend-api/internal/domain/documentjob"
	platform "tsk/backend-api/internal/domain/platform"
	"tsk/backend-api/internal/application/voiceintent"
	"tsk/backend-api/internal/integrations/bitrixclient"
)

// resolveDealIDForBitrix: номер из текста/формы или поиск по названию (точный, затем по подстроке).
func (s *Service) resolveDealIDForBitrix(ctx context.Context, intent voiceintent.Intent, in AdminVoiceBitrixInput) (dealID int, matchedTitle string, steps []string) {
	id := intent.DealID
	if in.DealIDOverride > 0 {
		id = in.DealIDOverride
		steps = append(steps, fmt.Sprintf("Используем ID из формы: %d", id))
	}
	title := strings.TrimSpace(intent.DealTitle)
	if title == "" {
		title = strings.TrimSpace(in.DealTitleOverride)
	}
	if id > 0 {
		return id, title, steps
	}
	if title == "" {
		return 0, "", steps
	}

	steps = append(steps, fmt.Sprintf("Поиск сделки по названию (точно): %q", title))
	found, err := s.bitrix.FindDealIDByTitle(ctx, title)
	if err == nil && found > 0 {
		steps = append(steps, fmt.Sprintf("Найдено по точному TITLE, id=%d", found))
		return found, title, steps
	}
	if err != nil {
		steps = append(steps, "Точное совпадение: "+err.Error())
	}

	found2, matched, err2 := s.bitrix.FindDealIDByTitleFuzzy(ctx, title)
	if err2 == nil && found2 > 0 {
		steps = append(steps, fmt.Sprintf("Найдено по подстроке в названии: id=%d, совпало с %q", found2, matched))
		return found2, matched, steps
	}
	if err2 != nil {
		steps = append(steps, "Поиск по подстроке: "+err2.Error())
	}
	return 0, title, steps
}

func applyAdminVoiceTextHints(intent *voiceintent.Intent, in AdminVoiceBitrixInput) {
	if sh := strings.TrimSpace(in.StageHintText); sh != "" && strings.TrimSpace(intent.StageHint) == "" {
		intent.StageHint = sh
	}
	dh := strings.TrimSpace(in.DealHintText)
	if dh == "" {
		return
	}
	if id, err := strconv.Atoi(dh); err == nil && id > 0 && intent.DealID == 0 && in.DealIDOverride == 0 {
		intent.DealID = id
		return
	}
	if strings.TrimSpace(intent.DealTitle) == "" && strings.TrimSpace(in.DealTitleOverride) == "" {
		intent.DealTitle = dh
	}
}

// AdminVoiceBitrixInput — загрузка голоса из админки для тестового контура.
type AdminVoiceBitrixInput struct {
	TemplateID        string
	SourceName        string
	DealIDOverride    int
	DealTitleOverride string // опционально, если в аудио плохо распозналось название
	DealHintText      string // опционально: какую сделку (номер или название), если не из голоса
	StageHintText     string // опционально: куда / целевая стадия
	FileName          string
	MimeType          string
	Audio             []byte
}

// AdminVoiceBitrixResult — что сделано после транскрипции и попытки Bitrix.
type AdminVoiceBitrixResult struct {
	Transcript       string                `json:"transcript"`
	Job              domain.Job            `json:"job"`
	SourceDocument   domain.SourceDocument `json:"sourceDocument"`
	ParsedAction     string                `json:"parsedAction"`
	ParsedDealID     int                   `json:"parsedDealId"`
	ParsedDealTitle  string                `json:"parsedDealTitle"`
	BitrixSteps      []string              `json:"bitrixSteps"`
	BitrixConfigured bool                  `json:"bitrixConfigured"`
}

// RunAdminVoiceBitrixPipeline: Whisper → заявка + исходник → разбор текста → Bitrix (сделка/задача).
func (s *Service) RunAdminVoiceBitrixPipeline(ctx context.Context, in AdminVoiceBitrixInput) (AdminVoiceBitrixResult, error) {
	if s.whisper == nil {
		return AdminVoiceBitrixResult{}, fmt.Errorf("whisper is not configured (set WHISPER_BASE_URL)")
	}
	if len(in.Audio) == 0 {
		return AdminVoiceBitrixResult{}, fmt.Errorf("audio file is required")
	}

	fn := sanitizeFileName(in.FileName, "admin-voice.m4a")
	mt := fallbackString(strings.TrimSpace(in.MimeType), "audio/mp4")

	transcript, err := s.whisper.Transcribe(ctx, fn, mt, in.Audio)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordError("whisper_transcribe")
		}
		return AdminVoiceBitrixResult{}, fmt.Errorf("transcription: %w", err)
	}

	templateID, err := s.resolveDefaultTemplateID(ctx, in.TemplateID)
	if err != nil {
		return AdminVoiceBitrixResult{}, err
	}

	sourceName := strings.TrimSpace(in.SourceName)
	if sourceName == "" {
		sourceName = "Тест голоса из админки"
	}

	job, err := s.CreateJob(ctx, CreateJobInput{
		TemplateID:      templateID,
		SourceName:      sourceName,
		RequestedBy:     "admin-web",
		Payload:         transcript,
		DeliveryChannel: domain.DeliveryChannelInternal,
		DeliveryAddress: "",
	})
	if err != nil {
		return AdminVoiceBitrixResult{}, err
	}

	pipelineStarted := time.Now()

	if err := s.holdAdminVoiceJobForBitrix(ctx, job.ID); err != nil {
		return AdminVoiceBitrixResult{}, err
	}
	jobHeld, err := s.jobs.GetJobByID(ctx, job.ID)
	if err != nil {
		return AdminVoiceBitrixResult{}, err
	}
	job = jobHeld

	storedFile, err := s.storage.Save(ctx, "source", fn, in.Audio)
	if err != nil {
		_ = s.failJob(ctx, job, pipelineStarted, fmt.Errorf("сохранение аудио: %w", err))
		return AdminVoiceBitrixResult{}, err
	}

	src, err := s.sourceDocuments.CreateSourceDocument(ctx, domain.SourceDocumentCreateParams{
		ID:         "src-" + uuid.NewString(),
		JobID:      &job.ID,
		TemplateID: job.TemplateID,
		Kind:       domain.SourceDocumentKindVoiceRecording,
		Origin:     "admin-web",
		FileName:   storedFile.FileName,
		MimeType:   mt,
		StorageKey: storedFile.StorageKey,
		SizeBytes:  storedFile.SizeBytes,
		CreatedAt:  s.now().UTC(),
	})
	if err != nil {
		_ = s.failJob(ctx, job, pipelineStarted, fmt.Errorf("исходный документ: %w", err))
		return AdminVoiceBitrixResult{}, err
	}

	s.registerStoredFile(ctx, "source", storedFile, platform.EntityTypeSourceDocument, src.ID, in.Audio)

	if _, err := s.createEvent(ctx, &job.ID, "info", "voice.transcribed", "Транскрипция голоса завершена", truncateForEvent(transcript, 2000)); err != nil {
		return AdminVoiceBitrixResult{}, err
	}

	s.saveTranscription(ctx, platform.TranscriptionSourceAdminVoice, transcript, &job.ID, &src.ID)

	intent := voiceintent.Parse(transcript, in.DealIDOverride)
	if strings.TrimSpace(in.DealTitleOverride) != "" && strings.TrimSpace(intent.DealTitle) == "" {
		intent.DealTitle = strings.TrimSpace(in.DealTitleOverride)
	}
	applyAdminVoiceTextHints(&intent, in)

	res := AdminVoiceBitrixResult{
		Transcript:       transcript,
		Job:              job,
		SourceDocument:   src,
		ParsedAction:     intent.Action,
		ParsedDealID:     intent.DealID,
		ParsedDealTitle:  strings.TrimSpace(intent.DealTitle),
		BitrixSteps:      []string{},
		BitrixConfigured: s.bitrix != nil && s.bitrix.WebhookConfigured(),
	}

	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		if _, err := s.createEvent(ctx, &job.ID, "warning", "voice.bitrix_skipped", "Вебхук Bitrix не настроен — действия по сделке/задаче не выполнялись", ""); err != nil {
			return res, err
		}
		res.BitrixSteps = append(res.BitrixSteps, "Bitrix: вебхук не задан, шаги пропущены")
		jq, err := s.releaseAdminVoiceJobForDocumentQueue(ctx, job.ID)
		if err != nil {
			return res, err
		}
		res.Job = jq
		return res, nil
	}

	switch intent.Action {
	case voiceintent.ActionCreateTask:
		taskID, err := s.bitrix.AddTask(ctx, intent.TaskTitle, "TSK: голосовой запрос из админки\n\n"+transcript)
		if err != nil {
			res.BitrixSteps = append(res.BitrixSteps, "Задача: ошибка — "+err.Error())
			if _, e := s.createEvent(ctx, &job.ID, "error", "voice.bitrix_task_failed", "Не удалось создать задачу в Bitrix", err.Error()); e != nil {
				return res, e
			}
			jf, ferr := s.failAdminVoiceBitrixJob(ctx, job, pipelineStarted, fmt.Errorf("создание задачи Bitrix: %w", err))
			if ferr != nil {
				return res, ferr
			}
			res.Job = jf
			return res, nil
		}
		res.BitrixSteps = append(res.BitrixSteps, fmt.Sprintf("Задача создана, id=%s", taskID))
		if _, err := s.createEvent(ctx, &job.ID, "info", "voice.bitrix_task_created", "Создана задача в Bitrix24", taskID); err != nil {
			return res, err
		}
		jq, err := s.releaseAdminVoiceJobForDocumentQueue(ctx, job.ID)
		if err != nil {
			return res, err
		}
		res.Job = jq
		return res, nil

	case voiceintent.ActionMoveNext, voiceintent.ActionMovePrev, voiceintent.ActionMoveStage:
		dealID, matchedTitle, resolveSteps := s.resolveDealIDForBitrix(ctx, intent, in)
		res.BitrixSteps = append(res.BitrixSteps, resolveSteps...)
		if matchedTitle != "" {
			res.ParsedDealTitle = matchedTitle
		}
		if dealID <= 0 {
			res.BitrixSteps = append(res.BitrixSteps,
				"Сделка: не найден ID — укажите dealId в форме, поле «название сделки», или в голосе: «сделку «Название» следующий этап» / номер сделки")
			if _, err := s.createEvent(ctx, &job.ID, "warning", "voice.bitrix_no_deal", "Не выполнено: не найдена сделка по ID или названию", ""); err != nil {
				return res, err
			}
			jf, ferr := s.failAdminVoiceBitrixJob(ctx, job, pipelineStarted, errors.New("сделка не найдена по ID или названию"))
			if ferr != nil {
				return res, ferr
			}
			res.Job = jf
			return res, nil
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
			resolved, err := s.bitrix.ResolveStageID(ctx, dealID, intent.StageHint)
			if err != nil {
				res.BitrixSteps = append(res.BitrixSteps, "Сделка: не удалось определить стадию — "+err.Error())
				if _, e := s.createEvent(ctx, &job.ID, "error", "voice.bitrix_stage_resolve_failed", err.Error(), ""); e != nil {
					return res, e
				}
				jf, ferr := s.failAdminVoiceBitrixJob(ctx, job, pipelineStarted, fmt.Errorf("стадия Bitrix: %w", err))
				if ferr != nil {
					return res, ferr
				}
				res.Job = jf
				return res, nil
			}
			stageArg = resolved
		}

		newDealID, newStage, err := s.bitrix.MoveDealToStage(ctx, dealID, stageArg, dir)
		if err != nil {
			res.BitrixSteps = append(res.BitrixSteps, "Сделка: ошибка — "+err.Error())
			if _, e := s.createEvent(ctx, &job.ID, "error", "voice.bitrix_deal_move_failed", "Не удалось переместить сделку", err.Error()); e != nil {
				return res, e
			}
			jf, ferr := s.failAdminVoiceBitrixJob(ctx, job, pipelineStarted, fmt.Errorf("перенос сделки Bitrix: %w", err))
			if ferr != nil {
				return res, ferr
			}
			res.Job = jf
			return res, nil
		}
		res.BitrixSteps = append(res.BitrixSteps, fmt.Sprintf("Сделка #%d переведена на стадию %s", newDealID, newStage))
		if _, err := s.createEvent(ctx, &job.ID, "info", "voice.bitrix_deal_moved", "Сделка обновлена в Bitrix24", newStage); err != nil {
			return res, err
		}
		jq, err := s.releaseAdminVoiceJobForDocumentQueue(ctx, job.ID)
		if err != nil {
			return res, err
		}
		res.Job = jq
		return res, nil

	default:
		res.BitrixSteps = append(res.BitrixSteps, "Действие Bitrix не распознано по тексту (ожидаются: следующий этап, назад, стадия …, создай задачу …)")
		if _, err := s.createEvent(ctx, &job.ID, "warning", "voice.bitrix_no_action", "Транскрипт сохранён, явного действия Bitrix не найдено", ""); err != nil {
			return res, err
		}
		jf, ferr := s.failAdminVoiceBitrixJob(ctx, job, pipelineStarted, errors.New("действие Bitrix не распознано по тексту транскрипта"))
		if ferr != nil {
			return res, ferr
		}
		res.Job = jf
		return res, nil
	}
}

func truncateForEvent(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
