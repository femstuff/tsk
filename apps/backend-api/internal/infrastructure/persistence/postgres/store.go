package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "tsk/backend-api/internal/domain/documentjob"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

func (s *Store) ListTemplates(ctx context.Context) ([]domain.Template, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, category, version, description, file_name, mime_type, storage_key, size_bytes, created_at
		FROM document_templates
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Template, 0)
	for rows.Next() {
		template, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, template)
	}

	return items, rows.Err()
}

func (s *Store) GetTemplateByID(ctx context.Context, id string) (domain.Template, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, name, category, version, description, file_name, mime_type, storage_key, size_bytes, created_at
		FROM document_templates
		WHERE id = $1
	`, id)

	template, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Template{}, domain.ErrTemplateNotFound
		}

		return domain.Template{}, err
	}

	return template, nil
}

func (s *Store) CreateTemplate(ctx context.Context, params domain.TemplateCreateParams) (domain.Template, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO document_templates (
			id, name, category, version, description, file_name, mime_type, storage_key, size_bytes, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, name, category, version, description, file_name, mime_type, storage_key, size_bytes, created_at
	`, params.ID, params.Name, params.Category, params.Version, params.Description, params.FileName, params.MimeType, params.StorageKey, params.SizeBytes, params.CreatedAt)

	return scanTemplate(row)
}

func (s *Store) ListJobs(ctx context.Context) ([]domain.Job, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, template_id, template_name, source_name, requested_by, payload, delivery_channel, delivery_address,
		       dispatch_status, status, error_message, result_document_id, created_at, updated_at, started_at, completed_at
		FROM document_jobs
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Job, 0)
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, job)
	}

	return items, rows.Err()
}

func (s *Store) GetJobByID(ctx context.Context, id string) (domain.Job, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, template_id, template_name, source_name, requested_by, payload, delivery_channel, delivery_address,
		       dispatch_status, status, error_message, result_document_id, created_at, updated_at, started_at, completed_at
		FROM document_jobs
		WHERE id = $1
	`, id)

	job, err := scanJob(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Job{}, domain.ErrJobNotFound
		}

		return domain.Job{}, err
	}

	return job, nil
}

func (s *Store) CreateJob(ctx context.Context, params domain.JobCreateParams) (domain.Job, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO document_jobs (
			id, template_id, template_name, source_name, requested_by, payload, delivery_channel, delivery_address,
			dispatch_status, status, error_message, result_document_id, created_at, updated_at, started_at, completed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, '', NULL, $11, $11, NULL, NULL)
		RETURNING id, template_id, template_name, source_name, requested_by, payload, delivery_channel, delivery_address,
		          dispatch_status, status, error_message, result_document_id, created_at, updated_at, started_at, completed_at
	`, params.ID, params.TemplateID, params.TemplateName, params.SourceName, params.RequestedBy, params.Payload, params.DeliveryChannel, params.DeliveryAddress, params.DispatchStatus, params.Status, params.CreatedAt)

	return scanJob(row)
}

func (s *Store) ClaimNextQueuedJob(ctx context.Context) (domain.Job, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Job{}, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		WITH picked AS (
			SELECT id
			FROM document_jobs
			WHERE status = 'queued'
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE document_jobs AS jobs
		SET status = 'running',
		    updated_at = NOW(),
		    started_at = COALESCE(jobs.started_at, NOW()),
		    error_message = ''
		FROM picked
		WHERE jobs.id = picked.id
		RETURNING jobs.id, jobs.template_id, jobs.template_name, jobs.source_name, jobs.requested_by, jobs.payload,
		          jobs.delivery_channel, jobs.delivery_address, jobs.dispatch_status, jobs.status, jobs.error_message,
		          jobs.result_document_id, jobs.created_at, jobs.updated_at, jobs.started_at, jobs.completed_at
	`)

	job, err := scanJob(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Job{}, domain.ErrJobNotFound
		}

		return domain.Job{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Job{}, err
	}

	return job, nil
}

func (s *Store) UpdateJobStatus(ctx context.Context, id string, params domain.JobStatusUpdateParams) (domain.Job, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE document_jobs
		SET status = $2,
		    dispatch_status = $3,
		    error_message = $4,
		    result_document_id = $5,
		    started_at = $6,
		    completed_at = $7,
		    updated_at = $8
		WHERE id = $1
		RETURNING id, template_id, template_name, source_name, requested_by, payload, delivery_channel, delivery_address,
		          dispatch_status, status, error_message, result_document_id, created_at, updated_at, started_at, completed_at
	`, id, params.Status, params.DispatchStatus, params.ErrorMessage, params.ResultDocumentID, params.StartedAt, params.CompletedAt, params.UpdatedAt)

	job, err := scanJob(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Job{}, domain.ErrJobNotFound
		}

		return domain.Job{}, err
	}

	return job, nil
}

func (s *Store) CountJobsByStatus(ctx context.Context) (map[domain.Status]int, error) {
	counts := make(map[domain.Status]int, len(domain.ValidStatuses()))
	for _, status := range domain.ValidStatuses() {
		counts[status] = 0
	}

	rows, err := s.pool.Query(ctx, `
		SELECT status, COUNT(*)
		FROM document_jobs
		GROUP BY status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}

		counts[domain.Status(status)] = count
	}

	return counts, rows.Err()
}

func (s *Store) ListGeneratedDocuments(ctx context.Context, jobID string) ([]domain.GeneratedDocument, error) {
	baseQuery := `
		SELECT id, job_id, template_id, template_name, file_name, mime_type, storage_key, size_bytes, created_at
		FROM generated_documents
	`
	args := []any{}
	if jobID != "" {
		baseQuery += ` WHERE job_id = $1`
		args = append(args, jobID)
	}
	baseQuery += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.GeneratedDocument, 0)
	for rows.Next() {
		document, err := scanGeneratedDocument(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, document)
	}

	return items, rows.Err()
}

func (s *Store) ListSourceDocuments(ctx context.Context, jobID string) ([]domain.SourceDocument, error) {
	baseQuery := `
		SELECT id, job_id, template_id, kind, origin, file_name, mime_type, storage_key, size_bytes, created_at
		FROM source_documents
	`
	args := []any{}
	if jobID != "" {
		baseQuery += ` WHERE job_id = $1`
		args = append(args, jobID)
	}
	baseQuery += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.SourceDocument, 0)
	for rows.Next() {
		document, err := scanSourceDocument(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, document)
	}

	return items, rows.Err()
}

func (s *Store) GetSourceDocumentByID(ctx context.Context, id string) (domain.SourceDocument, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, job_id, template_id, kind, origin, file_name, mime_type, storage_key, size_bytes, created_at
		FROM source_documents
		WHERE id = $1
	`, id)

	document, err := scanSourceDocument(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.SourceDocument{}, domain.ErrSourceDocumentNotFound
		}

		return domain.SourceDocument{}, err
	}

	return document, nil
}

func (s *Store) CreateSourceDocument(ctx context.Context, params domain.SourceDocumentCreateParams) (domain.SourceDocument, error) {
	var templateID any
	if strings.TrimSpace(params.TemplateID) != "" {
		templateID = params.TemplateID
	}

	row := s.pool.QueryRow(ctx, `
		INSERT INTO source_documents (
			id, job_id, template_id, kind, origin, file_name, mime_type, storage_key, size_bytes, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, job_id, template_id, kind, origin, file_name, mime_type, storage_key, size_bytes, created_at
	`, params.ID, params.JobID, templateID, params.Kind, params.Origin, params.FileName, params.MimeType, params.StorageKey, params.SizeBytes, params.CreatedAt)

	return scanSourceDocument(row)
}

func (s *Store) GetGeneratedDocumentByID(ctx context.Context, id string) (domain.GeneratedDocument, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, job_id, template_id, template_name, file_name, mime_type, storage_key, size_bytes, created_at
		FROM generated_documents
		WHERE id = $1
	`, id)

	document, err := scanGeneratedDocument(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.GeneratedDocument{}, domain.ErrGeneratedDocumentNotFound
		}

		return domain.GeneratedDocument{}, err
	}

	return document, nil
}

func (s *Store) CreateGeneratedDocument(ctx context.Context, params domain.GeneratedDocumentCreateParams) (domain.GeneratedDocument, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO generated_documents (
			id, job_id, template_id, template_name, file_name, mime_type, storage_key, size_bytes, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, job_id, template_id, template_name, file_name, mime_type, storage_key, size_bytes, created_at
	`, params.ID, params.JobID, params.TemplateID, params.TemplateName, params.FileName, params.MimeType, params.StorageKey, params.SizeBytes, params.CreatedAt)

	return scanGeneratedDocument(row)
}

func (s *Store) ListProcessingEvents(ctx context.Context, jobID string, limit int) ([]domain.ProcessingEvent, error) {
	query := `
		SELECT id, job_id, level, event_type, message, details, created_at
		FROM processing_events
	`
	args := []any{}
	if jobID != "" {
		query += ` WHERE job_id = $1`
		args = append(args, jobID)
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d", limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.ProcessingEvent, 0)
	for rows.Next() {
		event, err := scanProcessingEvent(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, event)
	}

	return items, rows.Err()
}

func (s *Store) CreateProcessingEvent(ctx context.Context, params domain.ProcessingEventCreateParams) (domain.ProcessingEvent, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO processing_events (id, job_id, level, event_type, message, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, job_id, level, event_type, message, details, created_at
	`, params.ID, params.JobID, params.Level, params.EventType, params.Message, params.Details, params.CreatedAt)

	return scanProcessingEvent(row)
}

func (s *Store) CountVoiceEventsSince(ctx context.Context, since time.Time) (int, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM processing_events
		WHERE created_at >= $1
		  AND (
		    event_type LIKE 'voice.%'
		    OR event_type LIKE 'mobile.bitrix_intent.%'
		    OR (event_type = 'source_document.uploaded' AND message ILIKE '%голос%')
		  )
	`, since)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) ListTaskCommands(ctx context.Context, jobID string) ([]domain.TaskCommand, error) {
	baseQuery := `
		SELECT id, job_id, source_document_id, target_system, command_text, status, integration_mode,
		       external_reference, result_message, created_at, updated_at
		FROM task_commands
	`
	args := []any{}
	if jobID != "" {
		baseQuery += ` WHERE job_id = $1`
		args = append(args, jobID)
	}
	baseQuery += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.TaskCommand, 0)
	for rows.Next() {
		command, err := scanTaskCommand(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, command)
	}

	return items, rows.Err()
}

func (s *Store) GetTaskCommandByID(ctx context.Context, id string) (domain.TaskCommand, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, job_id, source_document_id, target_system, command_text, status, integration_mode,
		       external_reference, result_message, created_at, updated_at
		FROM task_commands
		WHERE id = $1
	`, id)

	command, err := scanTaskCommand(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.TaskCommand{}, domain.ErrTaskCommandNotFound
		}

		return domain.TaskCommand{}, err
	}

	return command, nil
}

func (s *Store) CreateTaskCommand(ctx context.Context, params domain.TaskCommandCreateParams) (domain.TaskCommand, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO task_commands (
			id, job_id, source_document_id, target_system, command_text, status, integration_mode,
			external_reference, result_message, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		RETURNING id, job_id, source_document_id, target_system, command_text, status, integration_mode,
		          external_reference, result_message, created_at, updated_at
	`, params.ID, params.JobID, params.SourceDocumentID, params.TargetSystem, params.CommandText, params.Status, params.IntegrationMode, params.ExternalReference, params.ResultMessage, params.CreatedAt)

	return scanTaskCommand(row)
}

func (s *Store) UpdateTaskCommandStatus(ctx context.Context, id string, params domain.TaskCommandStatusUpdateParams) (domain.TaskCommand, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE task_commands
		SET status = $2,
		    integration_mode = $3,
		    external_reference = $4,
		    result_message = $5,
		    updated_at = $6
		WHERE id = $1
		RETURNING id, job_id, source_document_id, target_system, command_text, status, integration_mode,
		          external_reference, result_message, created_at, updated_at
	`, id, params.Status, params.IntegrationMode, params.ExternalReference, params.ResultMessage, params.UpdatedAt)

	command, err := scanTaskCommand(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.TaskCommand{}, domain.ErrTaskCommandNotFound
		}

		return domain.TaskCommand{}, err
	}

	return command, nil
}

func scanTemplate(row interface {
	Scan(...any) error
}) (domain.Template, error) {
	var template domain.Template
	err := row.Scan(
		&template.ID,
		&template.Name,
		&template.Category,
		&template.Version,
		&template.Description,
		&template.FileName,
		&template.MimeType,
		&template.StorageKey,
		&template.SizeBytes,
		&template.CreatedAt,
	)
	return template, err
}

func scanJob(row interface {
	Scan(...any) error
}) (domain.Job, error) {
	var job domain.Job
	var resultDocumentID *string
	var startedAt *time.Time
	var completedAt *time.Time
	err := row.Scan(
		&job.ID,
		&job.TemplateID,
		&job.TemplateName,
		&job.SourceName,
		&job.RequestedBy,
		&job.Payload,
		&job.DeliveryChannel,
		&job.DeliveryAddress,
		&job.DispatchStatus,
		&job.Status,
		&job.ErrorMessage,
		&resultDocumentID,
		&job.CreatedAt,
		&job.UpdatedAt,
		&startedAt,
		&completedAt,
	)
	job.ResultDocumentID = resultDocumentID
	job.StartedAt = startedAt
	job.CompletedAt = completedAt
	return job, err
}

func scanGeneratedDocument(row interface {
	Scan(...any) error
}) (domain.GeneratedDocument, error) {
	var document domain.GeneratedDocument
	err := row.Scan(
		&document.ID,
		&document.JobID,
		&document.TemplateID,
		&document.TemplateName,
		&document.FileName,
		&document.MimeType,
		&document.StorageKey,
		&document.SizeBytes,
		&document.CreatedAt,
	)
	return document, err
}

func scanProcessingEvent(row interface {
	Scan(...any) error
}) (domain.ProcessingEvent, error) {
	var event domain.ProcessingEvent
	var jobID *string
	err := row.Scan(
		&event.ID,
		&jobID,
		&event.Level,
		&event.EventType,
		&event.Message,
		&event.Details,
		&event.CreatedAt,
	)
	event.JobID = jobID
	return event, err
}

func scanSourceDocument(row interface {
	Scan(...any) error
}) (domain.SourceDocument, error) {
	var document domain.SourceDocument
	var jobID *string
	var templateID *string
	err := row.Scan(
		&document.ID,
		&jobID,
		&templateID,
		&document.Kind,
		&document.Origin,
		&document.FileName,
		&document.MimeType,
		&document.StorageKey,
		&document.SizeBytes,
		&document.CreatedAt,
	)
	document.JobID = jobID
	if templateID != nil {
		document.TemplateID = *templateID
	}
	return document, err
}

func scanTaskCommand(row interface {
	Scan(...any) error
}) (domain.TaskCommand, error) {
	var command domain.TaskCommand
	var jobID *string
	var sourceDocumentID *string
	err := row.Scan(
		&command.ID,
		&jobID,
		&sourceDocumentID,
		&command.TargetSystem,
		&command.CommandText,
		&command.Status,
		&command.IntegrationMode,
		&command.ExternalReference,
		&command.ResultMessage,
		&command.CreatedAt,
		&command.UpdatedAt,
	)
	command.JobID = jobID
	command.SourceDocumentID = sourceDocumentID
	return command, err
}
