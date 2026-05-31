package documentjob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	domain "tsk/backend-api/internal/domain/documentjob"
	"tsk/backend-api/internal/application/estimateintent"
)

func (s *Service) resolveEstimateTemplateID(ctx context.Context, templateID string) (domain.Template, error) {
	if tid := strings.TrimSpace(templateID); tid != "" {
		template, err := s.templates.GetTemplateByID(ctx, tid)
		if err != nil {
			return domain.Template{}, err
		}
		if !estimateintent.IsEstimateCategory(template.Category) {
			return domain.Template{}, errors.New("выбранный шаблон не является сметой")
		}
		return template, nil
	}

	list, err := s.templates.ListTemplates(ctx)
	if err != nil {
		return domain.Template{}, err
	}
	for _, template := range list {
		if estimateintent.IsEstimateCategory(template.Category) {
			return template, nil
		}
	}
	return domain.Template{}, errors.New("шаблон сметы не найден")
}

func (s *Service) transcribeVoice(ctx context.Context, fileName, mimeType string, audio []byte) (string, error) {
	if s.whisper == nil {
		return "", errors.New("транскрипция недоступна: задайте WHISPER_BASE_URL")
	}
	transcript, err := s.whisper.Transcribe(ctx, fileName, mimeType, audio)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordError("whisper_transcribe")
		}
		return "", fmt.Errorf("транскрипция: %w", err)
	}
	return strings.TrimSpace(transcript), nil
}

func (s *Service) buildEstimatePayload(transcript string) (string, estimateintent.Estimate, error) {
	estimate := estimateintent.Parse(transcript)
	payload, err := estimateintent.MarshalPayload(estimate)
	if err != nil {
		return "", estimateintent.Estimate{}, err
	}
	return payload, estimate, nil
}

func (s *Service) renderEstimateDocument(job domain.Job, template domain.Template) string {
	estimate := estimateintent.UnmarshalPayload(job.Payload)
	if strings.TrimSpace(estimate.RawTranscript) == "" && strings.TrimSpace(job.Payload) != "" {
		estimate.RawTranscript = job.Payload
	}

	var b strings.Builder
	b.WriteString(estimateintent.RenderForm4(estimate))
	b.WriteString("\n\n")
	b.WriteString(strings.Repeat("=", 40))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Шаблон: %s (%s)\n", template.Name, template.Version))
	b.WriteString(fmt.Sprintf("ID заявки: %s\n", job.ID))
	b.WriteString(fmt.Sprintf("Источник: %s\n", job.SourceName))
	b.WriteString(fmt.Sprintf("Инициатор: %s\n", job.RequestedBy))
	return strings.TrimSpace(b.String())
}

func (s *Service) saveVoiceTranscription(ctx context.Context, source string, transcript string, jobID, sourceDocID *string) {
	s.saveTranscription(ctx, source, transcript, jobID, sourceDocID)
}

func estimateSourceName(sourceName string, estimate estimateintent.Estimate) string {
	if name := strings.TrimSpace(sourceName); name != "" {
		return name
	}
	if project := strings.TrimSpace(estimate.ProjectName); project != "" {
		return project
	}
	if n := strings.TrimSpace(estimate.EstimateNumber); n != "" {
		return "Смета № " + n
	}
	return "Голосовая смета"
}

func previewEstimateFields(estimate estimateintent.Estimate) string {
	preview := map[string]any{
		"estimateNumber":    estimate.EstimateNumber,
		"projectName":       estimate.ProjectName,
		"objectDescription": estimate.ObjectDescription,
		"basis":             estimate.Basis,
		"estimatedCost":     estimate.EstimatedCost,
		"laborCosts":        estimate.LaborCosts,
		"priceDate":         estimate.PriceDate,
		"lineItemsCount":    len(estimate.LineItems),
		"grandTotal":        estimate.GrandTotal,
	}
	raw, _ := json.Marshal(preview)
	return string(raw)
}
