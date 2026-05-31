package postgres

import (
	"context"
	"fmt"
	"time"

	domain "tsk/backend-api/internal/domain/platform"
)

func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("postgres store is not initialized")
	}

	return s.pool.Ping(ctx)
}

func (s *Store) UpsertAppSetting(ctx context.Context, params domain.AppSettingUpsertParams) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value, description, is_secret, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			description = EXCLUDED.description,
			is_secret = EXCLUDED.is_secret,
			updated_at = EXCLUDED.updated_at
	`, params.Key, params.Value, params.Description, params.IsSecret, params.UpdatedAt)
	return err
}

func (s *Store) ListPublicAppSettings(ctx context.Context) ([]domain.AppSetting, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT key, value, description, is_secret, updated_at
		FROM app_settings
		WHERE is_secret = FALSE
		ORDER BY key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.AppSetting, 0)
	for rows.Next() {
		var item domain.AppSetting
		if err := rows.Scan(&item.Key, &item.Value, &item.Description, &item.IsSecret, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *Store) CreateStoredFile(ctx context.Context, params domain.StoredFileCreateParams) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO stored_files (
			id, namespace, storage_key, file_name, mime_type, size_bytes,
			sha256_hex, entity_type, entity_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (storage_key) DO UPDATE SET
			entity_type = EXCLUDED.entity_type,
			entity_id = EXCLUDED.entity_id,
			size_bytes = EXCLUDED.size_bytes,
			sha256_hex = EXCLUDED.sha256_hex
	`, params.ID, params.Namespace, params.StorageKey, params.FileName, params.MimeType,
		params.SizeBytes, params.SHA256Hex, params.EntityType, params.EntityID, params.CreatedAt)
	return err
}

func (s *Store) CreateTranscription(ctx context.Context, params domain.TranscriptionCreateParams) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO transcriptions (
			id, source, source_document_id, job_id, transcript,
			language, whisper_model, duration_ms, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, params.ID, params.Source, params.SourceDocumentID, params.JobID, params.Transcript,
		params.Language, params.WhisperModel, params.DurationMS, params.CreatedAt)
	return err
}

func (s *Store) NextJobAttemptNo(ctx context.Context, jobID string) (int, error) {
	var next int
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(attempt_no), 0) + 1
		FROM job_processing_attempts
		WHERE job_id = $1
	`, jobID).Scan(&next)
	return next, err
}

func (s *Store) CreateJobProcessingAttempt(ctx context.Context, params domain.JobAttemptCreateParams) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO job_processing_attempts (id, job_id, attempt_no, status, started_at)
		VALUES ($1, $2, $3, $4, $5)
	`, params.ID, params.JobID, params.AttemptNo, params.Status, params.StartedAt)
	return err
}

func (s *Store) FinishJobProcessingAttempt(ctx context.Context, attemptID string, params domain.JobAttemptFinishParams) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE job_processing_attempts
		SET status = $2, error_message = $3, finished_at = $4
		WHERE id = $1
	`, attemptID, params.Status, params.ErrorMessage, params.FinishedAt)
	return err
}

func (s *Store) RecoverStuckJobs(ctx context.Context, stuckAfter time.Duration) (int, error) {
	if stuckAfter <= 0 {
		stuckAfter = 10 * time.Minute
	}

	tag, err := s.pool.Exec(ctx, `
		UPDATE document_jobs
		SET status = 'queued',
		    error_message = 'recovered after worker restart',
		    started_at = NULL,
		    updated_at = NOW()
		WHERE status = 'running'
		  AND started_at IS NOT NULL
		  AND started_at < NOW() - ($1::bigint * INTERVAL '1 millisecond')
	`, stuckAfter.Milliseconds())
	if err != nil {
		return 0, err
	}

	return int(tag.RowsAffected()), nil
}
