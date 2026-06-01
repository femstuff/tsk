package documentjob

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	bdomain "tsk/backend-api/internal/domain/bitrixoauth"
	domain "tsk/backend-api/internal/domain/documentjob"
	platform "tsk/backend-api/internal/domain/platform"
	"tsk/backend-api/internal/application/estimateintent"
	"tsk/backend-api/internal/infrastructure/cache"
	"tsk/backend-api/internal/infrastructure/whisper"
	"tsk/backend-api/internal/integrations/bitrixoauth"
	"tsk/backend-api/internal/integrations/bitrixclient"
)

type MetricsRecorder interface {
	RecordJobCreated()
	RecordJobProcessed(domain.Status, time.Duration)
	RecordError(string)
	SyncJobStatusCounts(map[domain.Status]int)
}

type StoredFile struct {
	StorageKey string
	FileName   string
	SizeBytes  int64
	MimeType   string
}

type FileStorage interface {
	Save(context.Context, string, string, []byte) (StoredFile, error)
	Resolve(string) (string, error)
}

type IntegrationsConfig struct {
	BitrixWebhookURL        string
	BitrixDealEstimateField string
	ApprovalEmail           string
	BitrixOAuth             bitrixoauth.Config
}

type CreateTemplateInput struct {
	Name        string
	Category    string
	Version     string
	Description string
	FileName    string
	MimeType    string
	Content     []byte
}

type CreateJobInput struct {
	TemplateID      string                 `json:"templateId"`
	SourceName      string                 `json:"sourceName"`
	RequestedBy     string                 `json:"requestedBy"`
	Payload         string                 `json:"payload"`
	DeliveryChannel domain.DeliveryChannel `json:"deliveryChannel"`
	DeliveryAddress string                 `json:"deliveryAddress"`
	Status          domain.Status          `json:"status"`
	BitrixDealID    *int                   `json:"bitrixDealId"`
	BitrixDealTitle string                 `json:"bitrixDealTitle"`
}

type CreateMobileVoiceRequestInput struct {
	TemplateID      string
	SourceName      string
	RequestedBy     string
	Payload         string
	DeliveryChannel domain.DeliveryChannel
	DeliveryAddress string
	TaskCommandText string
	TaskTarget      domain.TaskTargetSystem
	BitrixDealID    int
	BitrixDealTitle string
	OAuthSessionID  string
}

type CreateTaskCommandInput struct {
	JobID            *string                 `json:"jobId"`
	SourceDocumentID *string                 `json:"sourceDocumentId"`
	TargetSystem     domain.TaskTargetSystem `json:"targetSystem"`
	CommandText      string                  `json:"commandText"`
}

type MobileVoiceRequestResult struct {
	Job            domain.Job                    `json:"job"`
	SourceDocument domain.SourceDocument         `json:"sourceDocument"`
	TaskCommand    *domain.TaskCommand           `json:"taskCommand,omitempty"`
	Estimate       estimateintent.EstimatePreview `json:"estimate"`
	Transcript     string                        `json:"transcript"`
	GeneratedDocument *domain.GeneratedDocument  `json:"generatedDocument,omitempty"`
}

type UpdateJobStatusInput struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

type Service struct {
	templates       domain.TemplateRepository
	jobs            domain.JobRepository
	documents       domain.GeneratedDocumentRepository
	sourceDocuments domain.SourceDocumentRepository
	taskCommands    domain.TaskCommandRepository
	events          domain.ProcessingEventRepository
	platform        platform.Repository
	storage         FileStorage
	cache           cache.Cache
	storageRoot     string
	metrics         MetricsRecorder
	integrations    IntegrationsConfig
	httpClient      *http.Client
	whisper         *whisper.Client
	estimateLLM     *estimateintent.LLMEnricher
	bitrix          *bitrixclient.Client
	bitrixOAuth     *bitrixoauth.Client
	bitrixOAuthCfg  bitrixoauth.Config
	bitrixOAuthSessions bdomain.Repository
	now             func() time.Time
}

func NewService(
	templates domain.TemplateRepository,
	jobs domain.JobRepository,
	documents domain.GeneratedDocumentRepository,
	sourceDocuments domain.SourceDocumentRepository,
	taskCommands domain.TaskCommandRepository,
	events domain.ProcessingEventRepository,
	platformRepo platform.Repository,
	storage FileStorage,
	cacheStore cache.Cache,
	storageRoot string,
	metrics MetricsRecorder,
	integrations IntegrationsConfig,
	whisperClient *whisper.Client,
	bitrixClient *bitrixclient.Client,
	bitrixOAuthSessions bdomain.Repository,
) *Service {
	oauthCfg := integrations.BitrixOAuth
	var oauthClient *bitrixoauth.Client
	if oauthCfg.Enabled() {
		oauthClient = bitrixoauth.NewClient(oauthCfg, &http.Client{Timeout: 30 * time.Second})
	}
	return &Service{
		templates:       templates,
		jobs:            jobs,
		documents:       documents,
		sourceDocuments: sourceDocuments,
		taskCommands:    taskCommands,
		events:          events,
		platform:        platformRepo,
		storage:         storage,
		cache:           cacheStore,
		storageRoot:     storageRoot,
		metrics:         metrics,
		integrations:    integrations,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		whisper:             whisperClient,
		bitrix:              bitrixClient,
		bitrixOAuth:         oauthClient,
		bitrixOAuthCfg:      oauthCfg,
		bitrixOAuthSessions: bitrixOAuthSessions,
		now:                 time.Now,
	}
}

// SetEstimateLLM включает пост-обработку полей сметы через LLM (если задан API key).
func (s *Service) SetEstimateLLM(enricher *estimateintent.LLMEnricher) {
	s.estimateLLM = enricher
}

func (s *Service) EnsureSeedData(ctx context.Context) error {
	if err := s.ensureEstimateTemplate(ctx); err != nil {
		return err
	}
	return s.syncJobStatusMetrics(ctx)
}

func (s *Service) ListTemplates(ctx context.Context) ([]domain.Template, error) {
	return s.listTemplatesCached(ctx)
}

func (s *Service) GetTemplate(ctx context.Context, id string) (domain.Template, error) {
	return s.templates.GetTemplateByID(ctx, id)
}

// resolveDefaultTemplateID подставляет первый доступный шаблон, если templateId не передан (голос из админки).
func (s *Service) resolveDefaultTemplateID(ctx context.Context, templateID string) (string, error) {
	if tid := strings.TrimSpace(templateID); tid != "" {
		if _, err := s.templates.GetTemplateByID(ctx, tid); err != nil {
			return "", err
		}
		return tid, nil
	}
	list, err := s.templates.ListTemplates(ctx)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", errors.New("нет шаблонов: загрузите шаблон или укажите templateId")
	}
	return list[0].ID, nil
}

func (s *Service) holdAdminVoiceJobForBitrix(ctx context.Context, jobID string) error {
	_, err := s.UpdateJobStatus(ctx, jobID, UpdateJobStatusInput{Status: string(domain.StatusRunning)})
	return err
}

func (s *Service) releaseAdminVoiceJobForDocumentQueue(ctx context.Context, jobID string) (domain.Job, error) {
	_, err := s.UpdateJobStatus(ctx, jobID, UpdateJobStatusInput{Status: string(domain.StatusQueued)})
	if err != nil {
		return domain.Job{}, err
	}
	return s.jobs.GetJobByID(ctx, jobID)
}

// failAdminVoiceBitrixJob помечает заявку failed после логической ошибки Bitrix (Grafana / tsk_document_jobs_by_status).
func (s *Service) failAdminVoiceBitrixJob(ctx context.Context, job domain.Job, started time.Time, err error) (domain.Job, error) {
	if s.metrics != nil {
		s.metrics.RecordError("bitrix")
	}
	if e := s.failJob(ctx, job, started, err); e != nil {
		return job, e
	}
	return s.jobs.GetJobByID(ctx, job.ID)
}

func (s *Service) CreateTemplate(ctx context.Context, input CreateTemplateInput) (domain.Template, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Template{}, errors.New("name is required")
	}

	if len(input.Content) == 0 {
		return domain.Template{}, errors.New("template file is required")
	}

	fileName := sanitizeFileName(input.FileName, "template.txt")
	mimeType := strings.TrimSpace(input.MimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	storedFile, err := s.storage.Save(ctx, "templates", fileName, input.Content)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordError("storage")
		}

		return domain.Template{}, err
	}

	template, err := s.templates.CreateTemplate(ctx, domain.TemplateCreateParams{
		ID:          "tpl-" + uuid.NewString(),
		Name:        name,
		Category:    fallbackString(input.Category, "general"),
		Version:     fallbackString(input.Version, "v1"),
		Description: strings.TrimSpace(input.Description),
		FileName:    storedFile.FileName,
		MimeType:    mimeType,
		StorageKey:  storedFile.StorageKey,
		SizeBytes:   storedFile.SizeBytes,
		CreatedAt:   s.now().UTC(),
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordError("database")
		}

		return domain.Template{}, err
	}

	if _, err := s.createEvent(ctx, nil, "info", "template.created", "Загружен шаблон документа "+template.Name, template.ID); err != nil {
		return domain.Template{}, err
	}

	s.registerStoredFile(ctx, "templates", storedFile, platform.EntityTypeTemplate, template.ID, input.Content)
	s.invalidateTemplatesCache(ctx)

	return template, nil
}

func (s *Service) DeleteTemplate(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("template id is required")
	}

	if _, err := s.templates.GetTemplateByID(ctx, id); err != nil {
		return err
	}

	count, err := s.templates.CountTemplateReferences(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrTemplateInUse
	}

	if err := s.templates.DeleteTemplate(ctx, id); err != nil {
		return err
	}

	if _, err := s.createEvent(ctx, nil, "info", "template.deleted", "Удалён шаблон документа", id); err != nil {
		return err
	}

	s.invalidateTemplatesCache(ctx)
	return nil
}

func (s *Service) ListJobs(ctx context.Context) ([]domain.Job, error) {
	return s.jobs.ListJobs(ctx)
}

func (s *Service) CreateJob(ctx context.Context, input CreateJobInput) (domain.Job, error) {
	template, requestedBy, deliveryChannel, err := s.validateJobInput(ctx, input.TemplateID, input.SourceName, input.RequestedBy, input.DeliveryChannel)
	if err != nil {
		return domain.Job{}, err
	}

	status := input.Status
	if status == "" {
		status = domain.StatusQueued
	}
	if !status.IsValid() {
		return domain.Job{}, errors.New("status is invalid")
	}

	var bitrixDealID *int
	if input.BitrixDealID != nil && *input.BitrixDealID > 0 {
		bitrixDealID = input.BitrixDealID
	}

	job, err := s.jobs.CreateJob(ctx, domain.JobCreateParams{
		ID:              "job-" + uuid.NewString(),
		TemplateID:      template.ID,
		TemplateName:    template.Name,
		SourceName:      strings.TrimSpace(input.SourceName),
		RequestedBy:     requestedBy,
		Payload:         strings.TrimSpace(input.Payload),
		DeliveryChannel: deliveryChannel,
		DeliveryAddress: strings.TrimSpace(input.DeliveryAddress),
		DispatchStatus:  dispatchStatusForChannel(deliveryChannel),
		Status:          status,
		BitrixDealID:    bitrixDealID,
		BitrixDealTitle: strings.TrimSpace(input.BitrixDealTitle),
		CreatedAt:       s.now().UTC(),
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordError("database")
		}

		return domain.Job{}, err
	}

	queueMsg := "Заявка на документ поставлена в очередь на обработку"
	queueType := "job.queued"
	if status == domain.StatusAwaitingReview {
		queueMsg = "Заявка создана — ожидает проверки полей"
		queueType = "job.awaiting_review"
	}
	if _, err := s.createEvent(ctx, &job.ID, "info", queueType, queueMsg, job.SourceName); err != nil {
		return domain.Job{}, err
	}

	if s.metrics != nil {
		s.metrics.RecordJobCreated()
	}

	if err := s.syncJobStatusMetrics(ctx); err != nil {
		return domain.Job{}, err
	}

	return job, nil
}

func (s *Service) CreateMobileVoiceRequest(ctx context.Context, input CreateMobileVoiceRequestInput, fileName string, mimeType string, content []byte) (MobileVoiceRequestResult, error) {
	if len(content) == 0 {
		return MobileVoiceRequestResult{}, errors.New("voice recording is required")
	}

	template, err := s.resolveEstimateTemplateID(ctx, input.TemplateID)
	if err != nil {
		return MobileVoiceRequestResult{}, err
	}

	fileName = sanitizeFileName(fileName, "voice-note.m4a")
	mimeType = fallbackString(mimeType, "audio/mp4")

	transcript, err := s.transcribeVoice(ctx, fileName, mimeType, content)
	if err != nil {
		return MobileVoiceRequestResult{}, err
	}
	if manual := strings.TrimSpace(input.Payload); manual != "" {
		transcript = strings.TrimSpace(transcript + "\n" + manual)
	}

	payload, estimate, err := s.buildEstimatePayload(ctx, transcript)
	if err != nil {
		return MobileVoiceRequestResult{}, err
	}

	dealID, dealTitle, err := s.resolveBitrixDealMeta(ctx, input.BitrixDealID, input.BitrixDealTitle, input.OAuthSessionID)
	if err != nil {
		return MobileVoiceRequestResult{}, err
	}
	var dealPtr *int
	if dealID > 0 {
		dealPtr = &dealID
	}

	job, err := s.CreateJob(ctx, CreateJobInput{
		TemplateID:      template.ID,
		SourceName:      estimateSourceName(input.SourceName, estimate),
		RequestedBy:     fallbackString(input.RequestedBy, mobileAppRequestedBy),
		Payload:         payload,
		DeliveryChannel: input.DeliveryChannel,
		DeliveryAddress: input.DeliveryAddress,
		Status:          domain.StatusAwaitingReview,
		BitrixDealID:    dealPtr,
		BitrixDealTitle: dealTitle,
	})
	if err != nil {
		return MobileVoiceRequestResult{}, err
	}

	storedFile, err := s.storage.Save(ctx, "source", fileName, content)
	if err != nil {
		return MobileVoiceRequestResult{}, err
	}

	sourceDocument, err := s.sourceDocuments.CreateSourceDocument(ctx, domain.SourceDocumentCreateParams{
		ID:         "src-" + uuid.NewString(),
		JobID:      &job.ID,
		TemplateID: job.TemplateID,
		Kind:       domain.SourceDocumentKindVoiceRecording,
		Origin:     "mobile-app",
		FileName:   storedFile.FileName,
		MimeType:   mimeType,
		StorageKey: storedFile.StorageKey,
		SizeBytes:  storedFile.SizeBytes,
		CreatedAt:  s.now().UTC(),
	})
	if err != nil {
		return MobileVoiceRequestResult{}, err
	}

	if _, err := s.createEvent(ctx, &job.ID, "info", "source_document.uploaded", "Сохранена голосовая запись из мобильного приложения", sourceDocument.FileName); err != nil {
		return MobileVoiceRequestResult{}, err
	}
	if _, err := s.createEvent(ctx, &job.ID, "info", "voice.transcribed", "Голос распознан — проверьте поля перед подтверждением", previewEstimateFields(estimate)); err != nil {
		return MobileVoiceRequestResult{}, err
	}
	if _, err := s.createEvent(ctx, &job.ID, "info", "job.awaiting_review", "Ожидает проверки полей в приложении", dealTitle); err != nil {
		return MobileVoiceRequestResult{}, err
	}

	s.registerStoredFile(ctx, "source", storedFile, platform.EntityTypeSourceDocument, sourceDocument.ID, content)
	s.saveVoiceTranscription(ctx, platform.TranscriptionSourceMobileVoice, transcript, &job.ID, &sourceDocument.ID)

	result := MobileVoiceRequestResult{
		Job:            job,
		SourceDocument: sourceDocument,
		Estimate:       estimateintent.ToPreview(estimate),
		Transcript:     transcript,
	}

	commandText := strings.TrimSpace(input.TaskCommandText)
	if commandText != "" {
		target := input.TaskTarget
		if target == "" {
			target = defaultTaskTarget(job.DeliveryChannel)
		}

		command, err := s.createAndDispatchTaskCommand(ctx, CreateTaskCommandInput{
			JobID:            &job.ID,
			SourceDocumentID: &sourceDocument.ID,
			TargetSystem:     target,
			CommandText:      commandText,
		})
		if err != nil {
			return MobileVoiceRequestResult{}, err
		}

		result.TaskCommand = &command
	}

	return result, nil
}

func (s *Service) CreateTaskCommand(ctx context.Context, input CreateTaskCommandInput) (domain.TaskCommand, error) {
	return s.createAndDispatchTaskCommand(ctx, input)
}

func (s *Service) UpdateJobStatus(ctx context.Context, id string, input UpdateJobStatusInput) (domain.Job, error) {
	job, err := s.jobs.GetJobByID(ctx, id)
	if err != nil {
		return domain.Job{}, err
	}

	status := domain.Status(strings.TrimSpace(input.Status))
	if !status.IsValid() {
		return domain.Job{}, errors.New("status is invalid")
	}

	now := s.now().UTC()
	startedAt := job.StartedAt
	completedAt := job.CompletedAt
	resultDocumentID := job.ResultDocumentID
	errorMessage := job.ErrorMessage
	dispatchStatus := job.DispatchStatus

	switch status {
	case domain.StatusQueued:
		startedAt = nil
		completedAt = nil
		resultDocumentID = nil
		errorMessage = ""
		dispatchStatus = dispatchStatusForChannel(job.DeliveryChannel)
	case domain.StatusRunning:
		if startedAt == nil {
			startedAt = &now
		}
		completedAt = nil
		errorMessage = ""
	case domain.StatusCompleted:
		if startedAt == nil {
			startedAt = &now
		}
		completedAt = &now
		errorMessage = ""
	case domain.StatusFailed:
		completedAt = &now
		errorMessage = fallbackString(input.Note, "Статус изменён на «Ошибка» из админ-панели")
		dispatchStatus = domain.DispatchStatusFailed
	case domain.StatusCancelled:
		completedAt = &now
		errorMessage = fallbackString(input.Note, "Заявка отменена из админ-панели")
	}

	updatedJob, err := s.jobs.UpdateJobStatus(ctx, id, domain.JobStatusUpdateParams{
		Status:           status,
		DispatchStatus:   dispatchStatus,
		ErrorMessage:     errorMessage,
		ResultDocumentID: resultDocumentID,
		StartedAt:        startedAt,
		CompletedAt:      completedAt,
		UpdatedAt:        now,
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordError("database")
		}

		return domain.Job{}, err
	}

	if _, err := s.createEvent(ctx, &updatedJob.ID, "info", "job.status.updated", "Статус заявки изменён: "+string(status), strings.TrimSpace(input.Note)); err != nil {
		return domain.Job{}, err
	}

	if err := s.syncJobStatusMetrics(ctx); err != nil {
		return domain.Job{}, err
	}

	return updatedJob, nil
}

func (s *Service) ListGeneratedDocuments(ctx context.Context, jobID string) ([]domain.GeneratedDocument, error) {
	return s.documents.ListGeneratedDocuments(ctx, strings.TrimSpace(jobID))
}

func (s *Service) GetGeneratedDocument(ctx context.Context, id string) (domain.GeneratedDocument, error) {
	return s.documents.GetGeneratedDocumentByID(ctx, id)
}

func (s *Service) ListSourceDocuments(ctx context.Context, jobID string) ([]domain.SourceDocument, error) {
	return s.sourceDocuments.ListSourceDocuments(ctx, strings.TrimSpace(jobID))
}

func (s *Service) GetSourceDocument(ctx context.Context, id string) (domain.SourceDocument, error) {
	return s.sourceDocuments.GetSourceDocumentByID(ctx, id)
}

func (s *Service) ListTaskCommands(ctx context.Context, jobID string) ([]domain.TaskCommand, error) {
	return s.taskCommands.ListTaskCommands(ctx, strings.TrimSpace(jobID))
}

func (s *Service) ListProcessingEvents(ctx context.Context, jobID string, limit int) ([]domain.ProcessingEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	return s.events.ListProcessingEvents(ctx, strings.TrimSpace(jobID), limit)
}

func (s *Service) ResolveStoragePath(storageKey string) (string, error) {
	return s.storage.Resolve(storageKey)
}

func (s *Service) ProcessNextQueuedJob(ctx context.Context) (bool, error) {
	job, err := s.jobs.ClaimNextQueuedJob(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			return false, nil
		}

		if s.metrics != nil {
			s.metrics.RecordError("database")
		}

		return false, err
	}

	processingStartedAt := time.Now()
	attemptID, err := s.beginJobAttempt(ctx, job.ID)
	if err != nil {
		return true, err
	}

	if _, err := s.createEvent(ctx, &job.ID, "info", "job.started", "Заявка из очереди взята в обработку", ""); err != nil {
		s.finishJobAttempt(ctx, attemptID, platform.AttemptStatusFailed, err)
		return true, err
	}

	_, _, err = s.generateAndCompleteJob(ctx, job)
	if err != nil {
		s.finishJobAttempt(ctx, attemptID, platform.AttemptStatusFailed, err)
		return true, s.failJob(ctx, job, processingStartedAt, err)
	}

	s.finishJobAttempt(ctx, attemptID, platform.AttemptStatusCompleted, nil)

	if s.metrics != nil {
		s.metrics.RecordJobProcessed(domain.StatusCompleted, time.Since(processingStartedAt))
	}

	if err := s.syncJobStatusMetrics(ctx); err != nil {
		return true, err
	}

	return true, nil
}

func (s *Service) validateJobInput(ctx context.Context, templateID string, sourceName string, requestedBy string, deliveryChannel domain.DeliveryChannel) (domain.Template, string, domain.DeliveryChannel, error) {
	if strings.TrimSpace(templateID) == "" {
		return domain.Template{}, "", "", errors.New("templateId is required")
	}

	if strings.TrimSpace(sourceName) == "" {
		return domain.Template{}, "", "", errors.New("sourceName is required")
	}

	template, err := s.templates.GetTemplateByID(ctx, templateID)
	if err != nil {
		return domain.Template{}, "", "", err
	}

	requestedBy = fallbackString(requestedBy, "admin-web")
	if deliveryChannel == "" {
		deliveryChannel = domain.DeliveryChannelInternal
	}

	if !deliveryChannel.IsValid() {
		return domain.Template{}, "", "", errors.New("deliveryChannel is invalid")
	}

	return template, requestedBy, deliveryChannel, nil
}

func (s *Service) routeCompletedJob(ctx context.Context, job domain.Job, document domain.GeneratedDocument) (domain.Job, error) {
	switch job.DeliveryChannel {
	case domain.DeliveryChannelInternal:
		return job, nil
	case domain.DeliveryChannelEmail:
		approvalAddress := fallbackString(job.DeliveryAddress, s.integrations.ApprovalEmail)
		command, err := s.createAndDispatchTaskCommand(ctx, CreateTaskCommandInput{
			JobID:        &job.ID,
			TargetSystem: domain.TaskTargetEmailApproval,
			CommandText:  fmt.Sprintf("Согласовать сформированный документ %s для %s", document.FileName, fallbackString(approvalAddress, "ops-review")),
		})
		if err != nil {
			return job, err
		}

		if _, err := s.createEvent(ctx, &job.ID, "info", "dispatch.email_approval", "Сценарий email-согласования зарегистрирован", command.ResultMessage); err != nil {
			return job, err
		}

		return job, nil
	case domain.DeliveryChannelBitrix:
		if strings.TrimSpace(s.integrations.BitrixWebhookURL) == "" {
			approvalAddress := fallbackString(job.DeliveryAddress, s.integrations.ApprovalEmail)
			command, err := s.createAndDispatchTaskCommand(ctx, CreateTaskCommandInput{
				JobID:        &job.ID,
				TargetSystem: domain.TaskTargetEmailApproval,
				CommandText:  fmt.Sprintf("Резервное согласование для отправки в Битрикс: %s -> %s", document.FileName, fallbackString(approvalAddress, "ops-review")),
			})
			if err != nil {
				return job, err
			}

			if _, err := s.createEvent(ctx, &job.ID, "warning", "dispatch.bitrix_fallback", "Вебхук Битрикс не настроен, зарегистрирован резервный сценарий согласования", command.ResultMessage); err != nil {
				return job, err
			}

			return job, nil
		}

		reference, err := s.sendToBitrix(ctx, job, document)
		finalStatus := domain.DispatchStatusSent
		message := "Передача в Битрикс завершена"
		if err != nil {
			finalStatus = domain.DispatchStatusFailed
			message = err.Error()
			if _, eventErr := s.createEvent(ctx, &job.ID, "error", "dispatch.bitrix_failed", "Ошибка отправки в Битрикс", err.Error()); eventErr != nil {
				return job, eventErr
			}
		} else {
			if _, eventErr := s.createEvent(ctx, &job.ID, "info", "dispatch.bitrix_sent", "Передача в Битрикс завершена", reference); eventErr != nil {
				return job, eventErr
			}
		}

		updatedJob, updateErr := s.jobs.UpdateJobStatus(ctx, job.ID, domain.JobStatusUpdateParams{
			Status:           job.Status,
			DispatchStatus:   finalStatus,
			ErrorMessage:     job.ErrorMessage,
			ResultDocumentID: job.ResultDocumentID,
			StartedAt:        job.StartedAt,
			CompletedAt:      job.CompletedAt,
			UpdatedAt:        s.now().UTC(),
		})
		if updateErr != nil {
			return job, updateErr
		}

		if finalStatus == domain.DispatchStatusFailed && s.metrics != nil {
			s.metrics.RecordError("bitrix")
		}

		_ = message
		return updatedJob, nil
	default:
		return job, nil
	}
}

func (s *Service) sendToBitrix(ctx context.Context, job domain.Job, document domain.GeneratedDocument) (string, error) {
	payload := fmt.Sprintf(
		`{"jobId":"%s","templateId":"%s","templateName":"%s","documentId":"%s","fileName":"%s","storageKey":"%s"}`,
		escapeJSON(job.ID),
		escapeJSON(job.TemplateID),
		escapeJSON(job.TemplateName),
		escapeJSON(document.ID),
		escapeJSON(document.FileName),
		escapeJSON(document.StorageKey),
	)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.integrations.BitrixWebhookURL, bytes.NewBufferString(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("вебхук Битрикс вернул статус %d", response.StatusCode)
	}

	return fmt.Sprintf("http-%d", response.StatusCode), nil
}

func (s *Service) createAndDispatchTaskCommand(ctx context.Context, input CreateTaskCommandInput) (domain.TaskCommand, error) {
	commandText := strings.TrimSpace(input.CommandText)
	if commandText == "" {
		return domain.TaskCommand{}, errors.New("commandText is required")
	}

	target := input.TargetSystem
	if target == "" {
		target = domain.TaskTargetEmailApproval
	}

	command, err := s.taskCommands.CreateTaskCommand(ctx, domain.TaskCommandCreateParams{
		ID:                "task-" + uuid.NewString(),
		JobID:             input.JobID,
		SourceDocumentID:  input.SourceDocumentID,
		TargetSystem:      target,
		CommandText:       commandText,
		Status:            domain.TaskCommandStatusRecorded,
		IntegrationMode:   "stub",
		ExternalReference: "",
		ResultMessage:     "Команда зарегистрирована",
		CreatedAt:         s.now().UTC(),
	})
	if err != nil {
		return domain.TaskCommand{}, err
	}

	finalStatus := domain.TaskCommandStatusRecorded
	integrationMode := "stub"
	externalReference := ""
	resultMessage := "Команда только зарегистрирована"

	switch target {
	case domain.TaskTargetBitrix24:
		if strings.TrimSpace(s.integrations.BitrixWebhookURL) != "" {
			reference, dispatchErr := s.sendTaskCommandToBitrix(ctx, command)
			if dispatchErr != nil {
				finalStatus = domain.TaskCommandStatusFailed
				integrationMode = "webhook"
				resultMessage = dispatchErr.Error()
			} else {
				finalStatus = domain.TaskCommandStatusSent
				integrationMode = "webhook"
				externalReference = reference
				resultMessage = "Команда отправлена в Битрикс"
			}
		} else {
			resultMessage = "Вебхук Битрикс не настроен; команда только зарегистрирована"
		}
	case domain.TaskTargetEmailApproval:
		finalStatus = domain.TaskCommandStatusPending
		resultMessage = fmt.Sprintf("Сценарий email-согласования зарегистрирован для %s", fallbackString(s.integrations.ApprovalEmail, "manual-review"))
	}

	command, err = s.taskCommands.UpdateTaskCommandStatus(ctx, command.ID, domain.TaskCommandStatusUpdateParams{
		Status:            finalStatus,
		IntegrationMode:   integrationMode,
		ExternalReference: externalReference,
		ResultMessage:     resultMessage,
		UpdatedAt:         s.now().UTC(),
	})
	if err != nil {
		return domain.TaskCommand{}, err
	}

	if _, err := s.createEvent(ctx, input.JobID, "info", "task_command.recorded", "Команда задачи сохранена", resultMessage); err != nil {
		return domain.TaskCommand{}, err
	}

	if finalStatus == domain.TaskCommandStatusFailed && s.metrics != nil {
		s.metrics.RecordError("task_command")
	}

	return command, nil
}

func (s *Service) sendTaskCommandToBitrix(ctx context.Context, command domain.TaskCommand) (string, error) {
	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		return "", errors.New("вебхук Bitrix не настроен (BITRIX_WEBHOOK_URL)")
	}
	// Раньше сюда слали JSON на корень вебхука — у Bitrix это 404. Нужен вызов REST, как в bitrixclient.
	title := strings.TrimSpace(command.CommandText)
	if len(title) > 200 {
		title = title[:200] + "…"
	}
	if title == "" {
		title = "TSK: команда без текста"
	}
	body := fmt.Sprintf("TSK: команда из мобильного приложения\nid=%s\ntarget=%s\n\n%s",
		command.ID, command.TargetSystem, strings.TrimSpace(command.CommandText))
	return s.bitrix.AddTask(ctx, title, body)
}

func (s *Service) failJob(ctx context.Context, job domain.Job, started time.Time, err error) error {
	failedAt := s.now().UTC()
	startedAt := job.StartedAt
	if startedAt == nil {
		startedAt = &failedAt
	}

	_, updateErr := s.jobs.UpdateJobStatus(ctx, job.ID, domain.JobStatusUpdateParams{
		Status:           domain.StatusFailed,
		DispatchStatus:   domain.DispatchStatusFailed,
		ErrorMessage:     err.Error(),
		ResultDocumentID: job.ResultDocumentID,
		StartedAt:        startedAt,
		CompletedAt:      &failedAt,
		UpdatedAt:        failedAt,
	})
	if updateErr != nil {
		return updateErr
	}

	if _, eventErr := s.createEvent(ctx, &job.ID, "error", "job.failed", "Ошибка генерации документа", err.Error()); eventErr != nil {
		return eventErr
	}

	if s.metrics != nil {
		s.metrics.RecordError("job_processing")
		s.metrics.RecordJobProcessed(domain.StatusFailed, time.Since(started))
	}

	return s.syncJobStatusMetrics(ctx)
}

func (s *Service) createEvent(ctx context.Context, jobID *string, level string, eventType string, message string, details string) (domain.ProcessingEvent, error) {
	return s.events.CreateProcessingEvent(ctx, domain.ProcessingEventCreateParams{
		ID:        "evt-" + uuid.NewString(),
		JobID:     jobID,
		Level:     strings.TrimSpace(level),
		EventType: strings.TrimSpace(eventType),
		Message:   strings.TrimSpace(message),
		Details:   strings.TrimSpace(details),
		CreatedAt: s.now().UTC(),
	})
}

func (s *Service) syncJobStatusMetrics(ctx context.Context) error {
	if s.metrics == nil {
		return nil
	}

	counts, err := s.jobs.CountJobsByStatus(ctx)
	if err != nil {
		return err
	}

	s.metrics.SyncJobStatusCounts(counts)
	return nil
}

func (s *Service) renderDocumentForTemplate(job domain.Job, template domain.Template) (renderedDocument, error) {
	if estimateintent.IsEstimateCategory(template.Category) {
		return s.renderEstimateDocument(job, template)
	}
	return renderedDocument{
		Content:  []byte(s.renderDocument(job, template)),
		FileName: "generated-document.txt",
		MimeType: "text/plain; charset=utf-8",
	}, nil
}

func (s *Service) renderDocument(job domain.Job, template domain.Template) string {
	return strings.TrimSpace(fmt.Sprintf(`
Сформированный документ TSK
===========================

Шаблон: %s
Версия шаблона: %s
Файл шаблона: %s
ID заявки: %s
Источник: %s
Инициатор: %s
Канал доставки: %s
Адрес доставки: %s
Создано: %s

Параметры
---------
%s

Примечания
----------
Этот MVP-генератор хранит шаблоны и сформированные файлы постоянно. Голосовые
входные файлы и команды задач фиксируются отдельно, чтобы админка и мобильное
приложение могли отслеживать весь процесс без имитации полной внешней автоматизации.
`, template.Name, template.Version, template.FileName, job.ID, job.SourceName, job.RequestedBy, job.DeliveryChannel, fallbackString(job.DeliveryAddress, "-"), job.CreatedAt.Format(time.RFC3339), fallbackString(job.Payload, "(пусто)")))
}

func fallbackString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}

	return trimmed
}

func sanitizeFileName(value string, fallback string) string {
	base := strings.TrimSpace(filepath.Base(value))
	base = strings.ReplaceAll(base, " ", "-")
	if base == "." || base == "/" || base == `\` || base == "" {
		return fallback
	}

	return base
}

func dispatchStatusForChannel(channel domain.DeliveryChannel) domain.DispatchStatus {
	if channel == domain.DeliveryChannelInternal {
		return domain.DispatchStatusNotRequired
	}

	return domain.DispatchStatusPending
}

func defaultTaskTarget(channel domain.DeliveryChannel) domain.TaskTargetSystem {
	if channel == domain.DeliveryChannelBitrix {
		return domain.TaskTargetBitrix24
	}

	return domain.TaskTargetEmailApproval
}

func escapeJSON(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return replacer.Replace(value)
}
