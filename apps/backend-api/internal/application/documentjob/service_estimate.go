package documentjob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"tsk/backend-api/internal/application/estimateintent"
	domain "tsk/backend-api/internal/domain/documentjob"
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
	if id := pickEstimateTemplateID(list); id != "" {
		return s.templates.GetTemplateByID(ctx, id)
	}
	return domain.Template{}, errors.New("шаблон сметы не найден: перезапустите backend или загрузите docx в админке")
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

func (s *Service) buildEstimatePayload(ctx context.Context, transcript string) (string, estimateintent.Estimate, error) {
	estimate := estimateintent.ParseWithEnricher(ctx, transcript, s.estimateLLM)
	payload, err := estimateintent.MarshalPayload(estimate)
	if err != nil {
		return "", estimateintent.Estimate{}, err
	}
	return payload, estimate, nil
}

type renderedDocument struct {
	Content  []byte
	FileName string
	MimeType string
}

func (s *Service) renderEstimateDocument(job domain.Job, template domain.Template) (renderedDocument, error) {
	estimate := estimateintent.UnmarshalPayload(job.Payload)
	if strings.TrimSpace(estimate.RawTranscript) == "" && strings.TrimSpace(job.Payload) != "" && !strings.HasPrefix(strings.TrimSpace(job.Payload), "{") {
		estimate.RawTranscript = job.Payload
	}

	templateBytes, err := s.readTemplateBytes(template)
	if err != nil {
		return renderedDocument{}, err
	}

	if estimateintent.IsDocxTemplate(template.FileName, template.MimeType) {
		filled, err := estimateintent.BuildEstimateDocx(estimate)
		if err != nil {
			return renderedDocument{}, err
		}
		return renderedDocument{
			Content:  filled,
			FileName: estimateDocxFileName(estimate),
			MimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		}, nil
	}

	if tpl := string(templateBytes); estimateintent.UsesPlaceholderTemplate(tpl) {
		return renderedDocument{
			Content:  []byte(estimateintent.FillTemplate(tpl, estimate)),
			FileName: "estimate-filled.txt",
			MimeType: "text/plain; charset=utf-8",
		}, nil
	}

	var b strings.Builder
	b.WriteString(estimateintent.RenderForm4(estimate))
	return renderedDocument{
		Content:  []byte(strings.TrimSpace(b.String())),
		FileName: "estimate-filled.txt",
		MimeType: "text/plain; charset=utf-8",
	}, nil
}

func (s *Service) readTemplateBytes(template domain.Template) ([]byte, error) {
	path, err := s.storage.Resolve(template.StorageKey)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}


func (s *Service) saveVoiceTranscription(ctx context.Context, source string, transcript string, jobID, sourceDocID *string) {
	s.saveTranscription(ctx, source, transcript, jobID, sourceDocID)
}

func estimateDocxFileName(estimate estimateintent.Estimate) string {
	base := "smeta"
	if n := strings.TrimSpace(estimate.EstimateNumber); n != "" {
		base += "-nr" + n
	}
	return base + ".docx"
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
