package documentjob

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	domain "tsk/backend-api/internal/domain/documentjob"
	platform "tsk/backend-api/internal/domain/platform"
	"tsk/backend-api/internal/infrastructure/cache"
)

const (
	cacheTemplatesTTL   = 60 * time.Second
	cacheBitrixTasksTTL = 30 * time.Second
)

func (s *Service) EnsureRuntimeSettings(ctx context.Context) error {
	if s.platform == nil {
		return nil
	}

	now := s.now().UTC()
	settings := []platform.AppSettingUpsertParams{
		{
			Key:         "approval_email",
			Value:       fallbackString(s.integrations.ApprovalEmail, "ops@example.local"),
			Description: "Email для сценария согласования",
			UpdatedAt:   now,
		},
		{
			Key:         "bitrix_configured",
			Value:       boolString(s.bitrix != nil && s.bitrix.WebhookConfigured()),
			Description: "Настроен ли входящий вебхук Bitrix24",
			UpdatedAt:   now,
		},
		{
			Key:         "whisper_configured",
			Value:       boolString(s.whisper != nil),
			Description: "Доступен ли сервис Whisper для транскрипции",
			UpdatedAt:   now,
		},
		{
			Key:         "redis_cache_enabled",
			Value:       boolString(s.cache != nil && !isNoopCache(s.cache)),
			Description: "Используется ли Redis для кеширования read-моделей",
			UpdatedAt:   now,
		},
	}

	for _, setting := range settings {
		if err := s.platform.UpsertAppSetting(ctx, setting); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) RecoverStuckJobs(ctx context.Context, stuckAfter time.Duration) (int, error) {
	if s.platform == nil {
		return 0, nil
	}

	return s.platform.RecoverStuckJobs(ctx, stuckAfter)
}

func (s *Service) ListPublicAppSettings(ctx context.Context) ([]platform.AppSetting, error) {
	if s.platform == nil {
		return nil, nil
	}

	return s.platform.ListPublicAppSettings(ctx)
}

func (s *Service) invalidateTemplatesCache(ctx context.Context) {
	if s.cache == nil {
		return
	}

	_ = s.cache.Delete(ctx, cache.KeyTemplatesList)
}

func (s *Service) registerStoredFile(ctx context.Context, namespace string, stored StoredFile, entityType string, entityID string, content []byte) {
	if s.platform == nil {
		return
	}

	sum := sha256.Sum256(content)
	_ = s.platform.CreateStoredFile(ctx, platform.StoredFileCreateParams{
		ID:         "file-" + uuid.NewString(),
		Namespace:  namespace,
		StorageKey: stored.StorageKey,
		FileName:   stored.FileName,
		MimeType:   fallbackString(stored.MimeType, "application/octet-stream"),
		SizeBytes:  stored.SizeBytes,
		SHA256Hex:  hex.EncodeToString(sum[:]),
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  s.now().UTC(),
	})
}

func (s *Service) saveTranscription(ctx context.Context, source string, transcript string, jobID *string, sourceDocumentID *string) {
	if s.platform == nil || strings.TrimSpace(transcript) == "" {
		return
	}

	_ = s.platform.CreateTranscription(ctx, platform.TranscriptionCreateParams{
		ID:               "trc-" + uuid.NewString(),
		Source:           source,
		SourceDocumentID: sourceDocumentID,
		JobID:            jobID,
		Transcript:       transcript,
		WhisperModel:     "whisper-api",
		CreatedAt:        s.now().UTC(),
	})
}

func (s *Service) beginJobAttempt(ctx context.Context, jobID string) (string, error) {
	if s.platform == nil {
		return "", nil
	}

	attemptNo, err := s.platform.NextJobAttemptNo(ctx, jobID)
	if err != nil {
		return "", err
	}

	attemptID := "att-" + uuid.NewString()
	if err := s.platform.CreateJobProcessingAttempt(ctx, platform.JobAttemptCreateParams{
		ID:        attemptID,
		JobID:     jobID,
		AttemptNo: attemptNo,
		Status:    platform.AttemptStatusStarted,
		StartedAt: s.now().UTC(),
	}); err != nil {
		return "", err
	}

	return attemptID, nil
}

func (s *Service) finishJobAttempt(ctx context.Context, attemptID string, status string, err error) {
	if s.platform == nil || attemptID == "" {
		return
	}

	message := ""
	if err != nil {
		message = err.Error()
	}

	_ = s.platform.FinishJobProcessingAttempt(ctx, attemptID, platform.JobAttemptFinishParams{
		Status:       status,
		ErrorMessage: message,
		FinishedAt:   s.now().UTC(),
	})
}

func boolString(value bool) string {
	if value {
		return "true"
	}

	return "false"
}

func isNoopCache(c cache.Cache) bool {
	_, ok := c.(*cache.NoopCache)
	return ok
}

func (s *Service) listTemplatesCached(ctx context.Context) ([]domain.Template, error) {
	if s.cache != nil {
		if payload, ok, err := s.cache.Get(ctx, cache.KeyTemplatesList); err == nil && ok {
			var items []domain.Template
			if json.Unmarshal(payload, &items) == nil {
				return items, nil
			}
		}
	}

	items, err := s.templates.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		if payload, err := json.Marshal(items); err == nil {
			_ = s.cache.Set(ctx, cache.KeyTemplatesList, payload, cacheTemplatesTTL)
		}
	}

	return items, nil
}
