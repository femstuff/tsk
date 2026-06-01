package estimateintent

import "strings"

// ValidationWarnings возвращает подсказки по незаполненным полям для экрана проверки.
func ValidationWarnings(estimate Estimate) []string {
	var warnings []string
	if strings.TrimSpace(estimate.ProjectName) == "" {
		warnings = append(warnings, "Не указано наименование стройки")
	}
	if strings.TrimSpace(estimate.ObjectDescription) == "" {
		warnings = append(warnings, "Не указаны работы / объект")
	}
	if name := strings.TrimSpace(estimate.ProjectName); name != "" && containsLatinLetters(name) {
		warnings = append(warnings, "Наименование стройки должно быть на русском (кириллица)")
	}
	if name := strings.TrimSpace(estimate.ObjectDescription); name != "" && containsLatinLetters(name) {
		warnings = append(warnings, "Наименование объекта/работ должно быть на русском (кириллица)")
	}
	if strings.TrimSpace(estimate.EstimatedCost) == "" && strings.TrimSpace(estimate.GrandTotal) == "" {
		warnings = append(warnings, "Не указана сметная стоимость или итого")
	}
	if len(estimate.LineItems) == 0 {
		warnings = append(warnings, "Не распознаны строки табличной части")
	}
	if strings.TrimSpace(estimate.RawTranscript) == "" {
		warnings = append(warnings, "Пустой транскрипт")
	}
	if warnings == nil {
		return []string{}
	}
	return warnings
}

// EstimatePreview — ответ API для экрана проверки полей.
type EstimatePreview struct {
	EstimateNumber    string     `json:"estimateNumber"`
	ProjectName       string     `json:"projectName"`
	ObjectDescription string     `json:"objectDescription"`
	Basis             string     `json:"basis"`
	EstimatedCost     string     `json:"estimatedCost"`
	LaborCosts        string     `json:"laborCosts"`
	PriceDate         string     `json:"priceDate"`
	Approver          string     `json:"approver"`
	TotalDirectCosts  string     `json:"totalDirectCosts"`
	GrandTotal        string     `json:"grandTotal"`
	LineItems         []LineItem `json:"lineItems"`
	RawTranscript     string     `json:"rawTranscript"`
	ValidationWarnings []string  `json:"validationWarnings"`
}

func ToPreview(estimate Estimate) EstimatePreview {
	lineItems := estimate.LineItems
	if lineItems == nil {
		lineItems = []LineItem{}
	}
	return EstimatePreview{
		EstimateNumber:     estimate.EstimateNumber,
		ProjectName:        estimate.ProjectName,
		ObjectDescription:  estimate.ObjectDescription,
		Basis:              estimate.Basis,
		EstimatedCost:      estimate.EstimatedCost,
		LaborCosts:         estimate.LaborCosts,
		PriceDate:          estimate.PriceDate,
		Approver:           estimate.Approver,
		TotalDirectCosts:   estimate.TotalDirectCosts,
		GrandTotal:         estimate.GrandTotal,
		LineItems:          lineItems,
		RawTranscript:      estimate.RawTranscript,
		ValidationWarnings: ValidationWarnings(estimate),
	}
}
