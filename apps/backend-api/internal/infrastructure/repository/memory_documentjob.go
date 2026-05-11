package repository

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	domain "tsk/backend-api/internal/domain/documentjob"
)

type InMemoryTemplateRepository struct {
	templates []domain.Template
}

func NewInMemoryTemplateRepository() *InMemoryTemplateRepository {
	now := time.Now().UTC()

	return &InMemoryTemplateRepository{
		templates: []domain.Template{
			{
				ID:          "tpl-service-brief",
				Name:        "Служебное резюме",
				Category:    "operations",
				Version:     "v1",
				Description: "Краткое операционное резюме по клиентскому запросу.",
				CreatedAt:   now,
			},
			{
				ID:          "tpl-sales-follow-up",
				Name:        "Повторное коммерческое предложение",
				Category:    "sales",
				Version:     "v1",
				Description: "Последующий документ для нового квалифицированного лида.",
				CreatedAt:   now,
			},
			{
				ID:          "tpl-onboarding-pack",
				Name:        "Пакет онбординга клиента",
				Category:    "customer-success",
				Version:     "v2",
				Description: "Стартовый комплект шаблонов для онбординга клиента.",
				CreatedAt:   now,
			},
		},
	}
}

func (r *InMemoryTemplateRepository) ListTemplates(context.Context) ([]domain.Template, error) {
	return slices.Clone(r.templates), nil
}

func (r *InMemoryTemplateRepository) GetTemplateByID(_ context.Context, id string) (domain.Template, error) {
	for _, template := range r.templates {
		if template.ID == id {
			return template, nil
		}
	}

	return domain.Template{}, domain.ErrTemplateNotFound
}

func (r *InMemoryTemplateRepository) CreateTemplate(_ context.Context, params domain.TemplateCreateParams) (domain.Template, error) {
	template := domain.Template{
		ID:          params.ID,
		Name:        params.Name,
		Category:    params.Category,
		Version:     params.Version,
		Description: params.Description,
		FileName:    params.FileName,
		MimeType:    params.MimeType,
		StorageKey:  params.StorageKey,
		SizeBytes:   params.SizeBytes,
		CreatedAt:   params.CreatedAt,
	}

	r.templates = append([]domain.Template{template}, r.templates...)
	return template, nil
}

type InMemoryJobRepository struct {
	mu   sync.RWMutex
	jobs []domain.Job
}

func NewInMemoryJobRepository() *InMemoryJobRepository {
	return &InMemoryJobRepository{
		jobs: make([]domain.Job, 0),
	}
}

func (r *InMemoryJobRepository) ListJobs(context.Context) ([]domain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return slices.Clone(r.jobs), nil
}

func (r *InMemoryJobRepository) GetJobByID(_ context.Context, id string) (domain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, job := range r.jobs {
		if job.ID == id {
			return job, nil
		}
	}

	return domain.Job{}, domain.ErrJobNotFound
}

func (r *InMemoryJobRepository) CreateJob(_ context.Context, params domain.JobCreateParams) (domain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	job := domain.Job{
		ID:              params.ID,
		TemplateID:      params.TemplateID,
		TemplateName:    params.TemplateName,
		SourceName:      params.SourceName,
		RequestedBy:     params.RequestedBy,
		Payload:         params.Payload,
		DeliveryChannel: params.DeliveryChannel,
		DeliveryAddress: params.DeliveryAddress,
		DispatchStatus:  params.DispatchStatus,
		Status:          params.Status,
		CreatedAt:       params.CreatedAt,
		UpdatedAt:       params.CreatedAt,
	}

	r.jobs = append([]domain.Job{job}, r.jobs...)
	return job, nil
}

func (r *InMemoryJobRepository) ClaimNextQueuedJob(_ context.Context) (domain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index, job := range r.jobs {
		if job.Status == domain.StatusQueued {
			now := time.Now().UTC()
			job.Status = domain.StatusRunning
			job.UpdatedAt = now
			job.StartedAt = &now
			r.jobs[index] = job
			return job, nil
		}
	}

	return domain.Job{}, domain.ErrJobNotFound
}

func (r *InMemoryJobRepository) UpdateJobStatus(_ context.Context, id string, params domain.JobStatusUpdateParams) (domain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index, job := range r.jobs {
		if job.ID == id {
			job.Status = params.Status
			job.DispatchStatus = params.DispatchStatus
			job.ErrorMessage = params.ErrorMessage
			job.ResultDocumentID = params.ResultDocumentID
			job.StartedAt = params.StartedAt
			job.CompletedAt = params.CompletedAt
			job.UpdatedAt = params.UpdatedAt
			r.jobs[index] = job
			return job, nil
		}
	}

	return domain.Job{}, domain.ErrJobNotFound
}

func (r *InMemoryJobRepository) CountJobsByStatus(_ context.Context) (map[domain.Status]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	counts := make(map[domain.Status]int)
	for _, status := range domain.ValidStatuses() {
		counts[status] = 0
	}

	for _, job := range r.jobs {
		counts[job.Status]++
	}

	return counts, nil
}

type InMemoryGeneratedDocumentRepository struct {
	mu        sync.RWMutex
	documents []domain.GeneratedDocument
}

func (r *InMemoryGeneratedDocumentRepository) ListGeneratedDocuments(_ context.Context, jobID string) ([]domain.GeneratedDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if jobID == "" {
		return slices.Clone(r.documents), nil
	}

	filtered := make([]domain.GeneratedDocument, 0)
	for _, document := range r.documents {
		if document.JobID == jobID {
			filtered = append(filtered, document)
		}
	}

	return filtered, nil
}

func (r *InMemoryGeneratedDocumentRepository) GetGeneratedDocumentByID(_ context.Context, id string) (domain.GeneratedDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, document := range r.documents {
		if document.ID == id {
			return document, nil
		}
	}

	return domain.GeneratedDocument{}, domain.ErrGeneratedDocumentNotFound
}

func (r *InMemoryGeneratedDocumentRepository) CreateGeneratedDocument(_ context.Context, params domain.GeneratedDocumentCreateParams) (domain.GeneratedDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	document := domain.GeneratedDocument{
		ID:           params.ID,
		JobID:        params.JobID,
		TemplateID:   params.TemplateID,
		TemplateName: params.TemplateName,
		FileName:     params.FileName,
		MimeType:     params.MimeType,
		StorageKey:   params.StorageKey,
		SizeBytes:    params.SizeBytes,
		CreatedAt:    params.CreatedAt,
	}

	r.documents = append([]domain.GeneratedDocument{document}, r.documents...)
	return document, nil
}

type InMemoryProcessingEventRepository struct {
	mu     sync.RWMutex
	events []domain.ProcessingEvent
}

func (r *InMemoryProcessingEventRepository) ListProcessingEvents(_ context.Context, jobID string, limit int) ([]domain.ProcessingEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]domain.ProcessingEvent, 0, len(r.events))
	for _, event := range r.events {
		if jobID == "" || (event.JobID != nil && *event.JobID == jobID) {
			items = append(items, event)
		}
		if limit > 0 && len(items) >= limit {
			break
		}
	}

	return items, nil
}

func (r *InMemoryProcessingEventRepository) CreateProcessingEvent(_ context.Context, params domain.ProcessingEventCreateParams) (domain.ProcessingEvent, error) {
	if strings.TrimSpace(params.Level) == "" {
		return domain.ProcessingEvent{}, errors.New("level is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	event := domain.ProcessingEvent{
		ID:        params.ID,
		JobID:     params.JobID,
		Level:     params.Level,
		EventType: params.EventType,
		Message:   params.Message,
		Details:   params.Details,
		CreatedAt: params.CreatedAt,
	}

	r.events = append([]domain.ProcessingEvent{event}, r.events...)
	return event, nil
}

type InMemorySourceDocumentRepository struct {
	mu        sync.RWMutex
	documents []domain.SourceDocument
}

func (r *InMemorySourceDocumentRepository) ListSourceDocuments(_ context.Context, jobID string) ([]domain.SourceDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if jobID == "" {
		return slices.Clone(r.documents), nil
	}

	filtered := make([]domain.SourceDocument, 0)
	for _, document := range r.documents {
		if document.JobID != nil && *document.JobID == jobID {
			filtered = append(filtered, document)
		}
	}

	return filtered, nil
}

func (r *InMemorySourceDocumentRepository) GetSourceDocumentByID(_ context.Context, id string) (domain.SourceDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, document := range r.documents {
		if document.ID == id {
			return document, nil
		}
	}

	return domain.SourceDocument{}, domain.ErrSourceDocumentNotFound
}

func (r *InMemorySourceDocumentRepository) CreateSourceDocument(_ context.Context, params domain.SourceDocumentCreateParams) (domain.SourceDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	document := domain.SourceDocument{
		ID:         params.ID,
		JobID:      params.JobID,
		TemplateID: params.TemplateID,
		Kind:       params.Kind,
		Origin:     params.Origin,
		FileName:   params.FileName,
		MimeType:   params.MimeType,
		StorageKey: params.StorageKey,
		SizeBytes:  params.SizeBytes,
		CreatedAt:  params.CreatedAt,
	}

	r.documents = append([]domain.SourceDocument{document}, r.documents...)
	return document, nil
}

type InMemoryTaskCommandRepository struct {
	mu       sync.RWMutex
	commands []domain.TaskCommand
}

func (r *InMemoryTaskCommandRepository) ListTaskCommands(_ context.Context, jobID string) ([]domain.TaskCommand, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if jobID == "" {
		return slices.Clone(r.commands), nil
	}

	filtered := make([]domain.TaskCommand, 0)
	for _, command := range r.commands {
		if command.JobID != nil && *command.JobID == jobID {
			filtered = append(filtered, command)
		}
	}

	return filtered, nil
}

func (r *InMemoryTaskCommandRepository) GetTaskCommandByID(_ context.Context, id string) (domain.TaskCommand, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, command := range r.commands {
		if command.ID == id {
			return command, nil
		}
	}

	return domain.TaskCommand{}, domain.ErrTaskCommandNotFound
}

func (r *InMemoryTaskCommandRepository) CreateTaskCommand(_ context.Context, params domain.TaskCommandCreateParams) (domain.TaskCommand, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	command := domain.TaskCommand{
		ID:                params.ID,
		JobID:             params.JobID,
		SourceDocumentID:  params.SourceDocumentID,
		TargetSystem:      params.TargetSystem,
		CommandText:       params.CommandText,
		Status:            params.Status,
		IntegrationMode:   params.IntegrationMode,
		ExternalReference: params.ExternalReference,
		ResultMessage:     params.ResultMessage,
		CreatedAt:         params.CreatedAt,
		UpdatedAt:         params.CreatedAt,
	}

	r.commands = append([]domain.TaskCommand{command}, r.commands...)
	return command, nil
}

func (r *InMemoryTaskCommandRepository) UpdateTaskCommandStatus(_ context.Context, id string, params domain.TaskCommandStatusUpdateParams) (domain.TaskCommand, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index, command := range r.commands {
		if command.ID == id {
			command.Status = params.Status
			command.IntegrationMode = params.IntegrationMode
			command.ExternalReference = params.ExternalReference
			command.ResultMessage = params.ResultMessage
			command.UpdatedAt = params.UpdatedAt
			r.commands[index] = command
			return command, nil
		}
	}

	return domain.TaskCommand{}, domain.ErrTaskCommandNotFound
}
