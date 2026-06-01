package documentjob

import (
	"context"
	"strings"

	"tsk/backend-api/internal/application/estimateintent"
	domain "tsk/backend-api/internal/domain/documentjob"
)

const estimateTemplateSeedName = "Смета строительно-отделочные работы"

func (s *Service) ensureEstimateTemplate(ctx context.Context) error {
	list, err := s.templates.ListTemplates(ctx)
	if err != nil {
		return err
	}

	hasDocxEstimate := false
	for _, template := range list {
		if !estimateintent.IsEstimateCategory(template.Category) {
			continue
		}
		if estimateintent.IsDocxTemplate(template.FileName, template.MimeType) {
			hasDocxEstimate = true
			break
		}
	}
	if hasDocxEstimate {
		return nil
	}

	content, err := estimateintent.DefaultEstimateDocxTemplate()
	if err != nil {
		return err
	}

	_, err = s.CreateTemplate(ctx, CreateTemplateInput{
		Name:        estimateTemplateSeedName,
		Category:    "estimate",
		Version:     "docx-1",
		Description: "Локальная смета (форма № 4) — Word docx, заполнение из голоса.",
		FileName:    "blank-smety-stroitelno-otdelochnye-raboty.docx",
		MimeType:    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Content:     content,
	})
	if err != nil {
		return err
	}

	if _, err := s.createEvent(ctx, nil, "info", "system.seeded", "Загружен шаблон сметы Word (docx)", estimateTemplateSeedName); err != nil {
		return err
	}
	return nil
}

func pickEstimateTemplateID(list []domain.Template) string {
	for _, template := range list {
		if !estimateintent.IsEstimateCategory(template.Category) {
			continue
		}
		if estimateintent.IsDocxTemplate(template.FileName, template.MimeType) {
			return template.ID
		}
	}
	for _, template := range list {
		if estimateintent.IsEstimateCategory(template.Category) {
			return template.ID
		}
	}
	for _, template := range list {
		name := strings.ToLower(template.Name)
		if strings.Contains(name, "смет") {
			return template.ID
		}
	}
	return ""
}
