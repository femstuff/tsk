package documentjob

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	domain "tsk/backend-api/internal/domain/documentjob"
	platform "tsk/backend-api/internal/domain/platform"
	"tsk/backend-api/internal/application/estimateintent"
	"tsk/backend-api/internal/integrations/bitrixclient"

	"github.com/google/uuid"
)

// generateJobDocument формирует файл и сохраняет result_document_id (без финального статуса).
func (s *Service) generateJobDocument(ctx context.Context, job domain.Job) (domain.Job, domain.GeneratedDocument, error) {
	template, err := s.templates.GetTemplateByID(ctx, job.TemplateID)
	if err != nil {
		return domain.Job{}, domain.GeneratedDocument{}, fmt.Errorf("load template: %w", err)
	}

	rendered, err := s.renderDocumentForTemplate(job, template)
	if err != nil {
		return domain.Job{}, domain.GeneratedDocument{}, fmt.Errorf("render document: %w", err)
	}
	fileName := sanitizeFileName(
		fmt.Sprintf("%s-%s-%s", job.TemplateName, job.ID, rendered.FileName),
		rendered.FileName,
	)
	storedFile, err := s.storage.Save(ctx, "generated", fileName, rendered.Content)
	if err != nil {
		return domain.Job{}, domain.GeneratedDocument{}, fmt.Errorf("store generated document: %w", err)
	}

	document, err := s.documents.CreateGeneratedDocument(ctx, domain.GeneratedDocumentCreateParams{
		ID:           "doc-" + uuid.NewString(),
		JobID:        job.ID,
		TemplateID:   job.TemplateID,
		TemplateName: job.TemplateName,
		FileName:     storedFile.FileName,
		MimeType:     rendered.MimeType,
		StorageKey:   storedFile.StorageKey,
		SizeBytes:    storedFile.SizeBytes,
		CreatedAt:    s.now().UTC(),
	})
	if err != nil {
		return domain.Job{}, domain.GeneratedDocument{}, fmt.Errorf("insert generated document: %w", err)
	}

	s.registerStoredFile(ctx, "generated", storedFile, platform.EntityTypeGeneratedDocument, document.ID, rendered.Content)

	now := s.now().UTC()
	startedAt := job.StartedAt
	if startedAt == nil {
		startedAt = &now
	}

	updatedJob, err := s.jobs.UpdateJobStatus(ctx, job.ID, domain.JobStatusUpdateParams{
		Status:           job.Status,
		DispatchStatus:   job.DispatchStatus,
		ErrorMessage:     "",
		ResultDocumentID: &document.ID,
		StartedAt:        startedAt,
		CompletedAt:      job.CompletedAt,
		UpdatedAt:        now,
	})
	if err != nil {
		return domain.Job{}, domain.GeneratedDocument{}, err
	}

	if _, err := s.createEvent(ctx, &updatedJob.ID, "info", "document.generated", "Сформирован итоговый документ", document.FileName); err != nil {
		return domain.Job{}, domain.GeneratedDocument{}, err
	}

	return updatedJob, document, nil
}

func (s *Service) mobileJobNeedsBitrixAttach(job domain.Job) bool {
	return job.RequestedBy == mobileAppRequestedBy &&
		job.BitrixDealID != nil && *job.BitrixDealID > 0
}

func (s *Service) markJobAwaitingBitrixAttach(ctx context.Context, job domain.Job, attachErr error) (domain.Job, error) {
	now := s.now().UTC()
	msg := ""
	dispatch := job.DispatchStatus
	if attachErr != nil {
		msg = attachErr.Error()
		dispatch = domain.DispatchStatusFailed
	}
	return s.jobs.UpdateJobStatus(ctx, job.ID, domain.JobStatusUpdateParams{
		Status:           domain.StatusAwaitingBitrixAttach,
		DispatchStatus:   dispatch,
		ErrorMessage:     msg,
		ResultDocumentID: job.ResultDocumentID,
		StartedAt:        job.StartedAt,
		CompletedAt:      nil,
		UpdatedAt:        now,
	})
}

func (s *Service) markJobCompletedAfterDocument(ctx context.Context, job domain.Job, document domain.GeneratedDocument) (domain.Job, error) {
	now := s.now().UTC()
	updatedJob, err := s.jobs.UpdateJobStatus(ctx, job.ID, domain.JobStatusUpdateParams{
		Status:           domain.StatusCompleted,
		DispatchStatus:   dispatchStatusForChannel(job.DeliveryChannel),
		ErrorMessage:     "",
		ResultDocumentID: job.ResultDocumentID,
		StartedAt:        job.StartedAt,
		CompletedAt:      &now,
		UpdatedAt:        now,
	})
	if err != nil {
		return domain.Job{}, err
	}
	return s.routeCompletedJob(ctx, updatedJob, document)
}

// generateAndCompleteJob — для админки/очереди без привязки к Bitrix.
func (s *Service) generateAndCompleteJob(ctx context.Context, job domain.Job) (domain.Job, domain.GeneratedDocument, error) {
	job, document, err := s.generateJobDocument(ctx, job)
	if err != nil {
		return domain.Job{}, domain.GeneratedDocument{}, err
	}
	updatedJob, err := s.markJobCompletedAfterDocument(ctx, job, document)
	if err != nil {
		return domain.Job{}, domain.GeneratedDocument{}, err
	}
	return updatedJob, document, nil
}

func (s *Service) processJobByID(ctx context.Context, jobID string) (domain.Job, domain.GeneratedDocument, error) {
	job, err := s.jobs.GetJobByID(ctx, jobID)
	if err != nil {
		return domain.Job{}, domain.GeneratedDocument{}, err
	}

	processingStartedAt := time.Now()
	attemptID, err := s.beginJobAttempt(ctx, job.ID)
	if err != nil {
		return domain.Job{}, domain.GeneratedDocument{}, err
	}

	now := s.now().UTC()
	if job.Status == domain.StatusAwaitingReview {
		running, err := s.jobs.UpdateJobStatus(ctx, job.ID, domain.JobStatusUpdateParams{
			Status:           domain.StatusRunning,
			DispatchStatus:   job.DispatchStatus,
			ErrorMessage:     "",
			ResultDocumentID: job.ResultDocumentID,
			StartedAt:        &now,
			CompletedAt:      nil,
			UpdatedAt:        now,
		})
		if err != nil {
			s.finishJobAttempt(ctx, attemptID, platform.AttemptStatusFailed, err)
			return domain.Job{}, domain.GeneratedDocument{}, err
		}
		job = running
	}

	if _, err := s.createEvent(ctx, &job.ID, "info", "job.started", "Заявка взята в обработку", ""); err != nil {
		s.finishJobAttempt(ctx, attemptID, platform.AttemptStatusFailed, err)
		return domain.Job{}, domain.GeneratedDocument{}, err
	}

	job, document, err := s.generateJobDocument(ctx, job)
	if err != nil {
		s.finishJobAttempt(ctx, attemptID, platform.AttemptStatusFailed, err)
		_ = s.failJob(ctx, job, processingStartedAt, err)
		return domain.Job{}, domain.GeneratedDocument{}, err
	}

	if s.mobileJobNeedsBitrixAttach(job) {
		job, err = s.markJobAwaitingBitrixAttach(ctx, job, nil)
		if err != nil {
			s.finishJobAttempt(ctx, attemptID, platform.AttemptStatusFailed, err)
			return domain.Job{}, domain.GeneratedDocument{}, err
		}
		if _, e := s.createEvent(ctx, &job.ID, "info", "job.awaiting_bitrix_attach", "Документ готов — ожидает прикрепления к сделке", ""); e != nil {
			s.finishJobAttempt(ctx, attemptID, platform.AttemptStatusFailed, e)
			return domain.Job{}, domain.GeneratedDocument{}, e
		}
		s.finishJobAttempt(ctx, attemptID, platform.AttemptStatusCompleted, nil)
		_ = s.syncJobStatusMetrics(ctx)
		return job, document, nil
	}

	updatedJob, err := s.markJobCompletedAfterDocument(ctx, job, document)
	if err != nil {
		s.finishJobAttempt(ctx, attemptID, platform.AttemptStatusFailed, err)
		_ = s.failJob(ctx, job, processingStartedAt, err)
		return domain.Job{}, domain.GeneratedDocument{}, err
	}

	s.finishJobAttempt(ctx, attemptID, platform.AttemptStatusCompleted, nil)
	if s.metrics != nil {
		s.metrics.RecordJobProcessed(domain.StatusCompleted, time.Since(processingStartedAt))
	}
	_ = s.syncJobStatusMetrics(ctx)

	return updatedJob, document, nil
}

func (s *Service) attachGeneratedDocumentToBitrixDeal(ctx context.Context, oauthSessionID string, job domain.Job, document domain.GeneratedDocument) error {
	if job.BitrixDealID == nil || *job.BitrixDealID <= 0 {
		return nil
	}

	path, err := s.storage.Resolve(document.StorageKey)
	if err != nil {
		return err
	}
	content, err := readFileBytes(path)
	if err != nil {
		return err
	}

	dealID := *job.BitrixDealID
	bitrixFileName := bitrixAttachFileName(job, document)
	attachParams := s.bitrixEstimateAttachParams(job)

	var attachErr error
	attachMode := ""

	if s.bitrix != nil && s.bitrix.WebhookConfigured() {
		attachMode = "webhook"
		attachErr = s.bitrix.AttachEstimateToDeal(ctx, dealID, bitrixFileName, content, attachParams)
	}

	if attachErr != nil && strings.TrimSpace(oauthSessionID) != "" && s.BitrixOAuthEnabled() {
		session, err := s.ensureActiveBitrixSession(ctx, oauthSessionID)
		if err != nil {
			if attachMode == "" {
				return err
			}
			return fmt.Errorf("%s: %w; oauth: %w", attachMode, attachErr, err)
		}
		rest := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)
		attachMode = "oauth"
		attachErr = rest.AttachEstimateToDeal(ctx, dealID, bitrixFileName, content, attachParams)
	}

	if attachMode == "" {
		return fmt.Errorf("bitrix is not configured (нужен BITRIX_WEBHOOK_URL или вход в Bitrix24)")
	}
	if attachErr != nil {
		return fmt.Errorf("bitrix %s: %w", attachMode, attachErr)
	}

	_, err = s.createEvent(ctx, &job.ID, "info", "bitrix.deal.file_attached",
		fmt.Sprintf("Файл %s прикреплён к сделке (%s)", bitrixFileName, attachMode), strconvItoa(dealID))
	return err
}

func bitrixAttachFileName(job domain.Job, document domain.GeneratedDocument) string {
	estimate := estimateintent.UnmarshalPayload(job.Payload)
	if name := estimateDocxFileName(estimate); strings.TrimSpace(name) != "" {
		return name
	}
	if name := strings.TrimSpace(document.FileName); name != "" {
		return name
	}
	return "smeta.docx"
}

func readFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func strconvItoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func (s *Service) bitrixEstimateAttachParams(job domain.Job) bitrixclient.AttachEstimateParams {
	estimate := estimateintent.UnmarshalPayload(job.Payload)
	objectLabel := strings.TrimSpace(estimate.ProjectName)
	if objectLabel == "" {
		objectLabel = strings.TrimSpace(estimate.ObjectDescription)
	}
	if objectLabel == "" {
		objectLabel = strings.TrimSpace(job.SourceName)
	}
	notice := "Смета добавлена, на проверку."
	if objectLabel != "" {
		notice = fmt.Sprintf("Смета добавлена для «%s», на проверку.", objectLabel)
	}
	return bitrixclient.AttachEstimateParams{
		FieldCode: s.integrations.BitrixDealEstimateField,
		Notice:    notice,
	}
}
