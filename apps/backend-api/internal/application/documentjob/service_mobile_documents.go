package documentjob

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tsk/backend-api/internal/application/estimateintent"
	domain "tsk/backend-api/internal/domain/documentjob"
)

// MobileDocumentJobView — заявка с полями сметы для мобильного списка/карточки.
type MobileDocumentJobView struct {
	Job               domain.Job                     `json:"job"`
	Estimate          estimateintent.EstimatePreview `json:"estimate"`
	GeneratedDocument *domain.GeneratedDocument      `json:"generatedDocument,omitempty"`
	DownloadPath      string                         `json:"downloadPath,omitempty"`
	CanRetryBitrix    bool                           `json:"canRetryBitrixAttach"`
}

func (s *Service) ListMobileDocumentJobs(ctx context.Context) ([]MobileDocumentJobView, error) {
	jobs, err := s.jobs.ListJobsByRequestedBy(ctx, mobileAppRequestedBy)
	if err != nil {
		return nil, err
	}
	items := make([]MobileDocumentJobView, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, s.buildMobileDocumentJobView(ctx, job))
	}
	return items, nil
}

func (s *Service) GetMobileDocumentJob(ctx context.Context, jobID string) (MobileDocumentJobView, error) {
	job, err := s.jobs.GetJobByID(ctx, strings.TrimSpace(jobID))
	if err != nil {
		return MobileDocumentJobView{}, err
	}
	if job.RequestedBy != mobileAppRequestedBy {
		return MobileDocumentJobView{}, domain.ErrJobNotFound
	}
	return s.buildMobileDocumentJobView(ctx, job), nil
}

func (s *Service) ConfirmMobileDocumentJob(ctx context.Context, jobID, oauthSessionID string) (MobileDocumentJobView, error) {
	jobID = strings.TrimSpace(jobID)
	job, err := s.jobs.GetJobByID(ctx, jobID)
	if err != nil {
		return MobileDocumentJobView{}, err
	}
	if job.RequestedBy != mobileAppRequestedBy {
		return MobileDocumentJobView{}, domain.ErrJobNotFound
	}
	if job.Status != domain.StatusAwaitingReview {
		return MobileDocumentJobView{}, fmt.Errorf("заявка не ожидает подтверждения (статус: %s)", job.Status)
	}

	updatedJob, document, err := s.processJobByID(ctx, job.ID)
	if err != nil {
		return MobileDocumentJobView{}, err
	}

	if updatedJob.Status == domain.StatusAwaitingBitrixAttach {
		if err := s.attachGeneratedDocumentToBitrixDeal(ctx, oauthSessionID, updatedJob, document); err != nil {
			if _, e := s.createEvent(ctx, &updatedJob.ID, "warning", "bitrix.deal.file_attach_failed", "Документ сформирован, но не прикреплён к сделке", err.Error()); e != nil {
				return MobileDocumentJobView{}, e
			}
			updatedJob, _ = s.markJobAwaitingBitrixAttach(ctx, updatedJob, err)
		} else {
			updatedJob, err = s.markJobCompletedAfterDocument(ctx, updatedJob, document)
			if err != nil {
				return MobileDocumentJobView{}, err
			}
		}
	}

	view := s.buildMobileDocumentJobView(ctx, updatedJob)
	view.GeneratedDocument = &document
	return view, nil
}

// RetryAttachMobileDocumentJob повторяет прикрепление к полю «Смета» в сделке.
func (s *Service) RetryAttachMobileDocumentJob(ctx context.Context, jobID, oauthSessionID string) (MobileDocumentJobView, error) {
	jobID = strings.TrimSpace(jobID)
	job, err := s.jobs.GetJobByID(ctx, jobID)
	if err != nil {
		return MobileDocumentJobView{}, err
	}
	if job.RequestedBy != mobileAppRequestedBy {
		return MobileDocumentJobView{}, domain.ErrJobNotFound
	}
	if job.Status != domain.StatusAwaitingBitrixAttach {
		return MobileDocumentJobView{}, fmt.Errorf("повтор возможен только для статуса «ожидает прикрепления к Bitrix» (сейчас: %s)", job.Status)
	}
	if job.ResultDocumentID == nil || strings.TrimSpace(*job.ResultDocumentID) == "" {
		return MobileDocumentJobView{}, errors.New("нет сформированного документа")
	}

	document, err := s.documents.GetGeneratedDocumentByID(ctx, *job.ResultDocumentID)
	if err != nil {
		return MobileDocumentJobView{}, err
	}

	if err := s.attachGeneratedDocumentToBitrixDeal(ctx, oauthSessionID, job, document); err != nil {
		job, _ = s.markJobAwaitingBitrixAttach(ctx, job, err)
		view := s.buildMobileDocumentJobView(ctx, job)
		view.GeneratedDocument = &document
		return view, nil
	}

	updatedJob, err := s.markJobCompletedAfterDocument(ctx, job, document)
	if err != nil {
		return MobileDocumentJobView{}, err
	}
	view := s.buildMobileDocumentJobView(ctx, updatedJob)
	view.GeneratedDocument = &document
	return view, nil
}

func (s *Service) buildMobileDocumentJobView(ctx context.Context, job domain.Job) MobileDocumentJobView {
	estimate := estimateintent.UnmarshalPayload(job.Payload)
	view := MobileDocumentJobView{
		Job:            job,
		Estimate:       estimateintent.ToPreview(estimate),
		CanRetryBitrix: job.Status == domain.StatusAwaitingBitrixAttach && job.BitrixDealID != nil && *job.BitrixDealID > 0,
	}
	if job.ResultDocumentID != nil && strings.TrimSpace(*job.ResultDocumentID) != "" {
		if doc, err := s.documents.GetGeneratedDocumentByID(ctx, *job.ResultDocumentID); err == nil {
			view.GeneratedDocument = &doc
			view.DownloadPath = "/api/v1/generated-documents/" + doc.ID + "/download"
		}
	}
	return view
}

func (s *Service) resolveBitrixDealMeta(ctx context.Context, dealID int, dealTitle, oauthSessionID string) (int, string, error) {
	if dealID <= 0 {
		return 0, "", errors.New("выберите сделку Bitrix24")
	}
	title := strings.TrimSpace(dealTitle)
	if title != "" {
		return dealID, title, nil
	}
	if strings.TrimSpace(oauthSessionID) != "" {
		if detail, err := s.GetBitrixDealForMobile(ctx, fmt.Sprintf("%d", dealID), oauthSessionID); err == nil {
			if t := strings.TrimSpace(detail.Title); t != "" {
				return dealID, t, nil
			}
		}
	}
	return dealID, fmt.Sprintf("Сделка #%d", dealID), nil
}
