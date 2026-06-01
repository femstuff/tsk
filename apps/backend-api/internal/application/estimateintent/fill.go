package estimateintent

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed templates/estimate-blank.txt
var DefaultEstimateTemplate string

// UsesPlaceholderTemplate reports whether a template file supports {{key}} substitution.
func UsesPlaceholderTemplate(content string) bool {
	return strings.Contains(content, "{{") && strings.Contains(content, "}}")
}

// FillTemplate подставляет поля сметы в шаблон с плейсхолдерами {{key}}.
func FillTemplate(template string, estimate Estimate) string {
	if !UsesPlaceholderTemplate(template) {
		return ""
	}

	replacements := map[string]string{
		"estimateNumber":    displayField(estimate.EstimateNumber),
		"projectName":       displayField(estimate.ProjectName),
		"objectDescription": displayField(estimate.ObjectDescription),
		"basis":             displayField(estimate.Basis),
		"estimatedCost":     displayAmount(estimate.EstimatedCost),
		"laborCosts":        displayAmount(estimate.LaborCosts),
		"priceDate":         displayField(estimate.PriceDate),
		"approver":          displayField(estimate.Approver),
		"totalDirectCosts":  displayAmount(estimate.TotalDirectCosts),
		"grandTotal":        displayAmount(estimate.GrandTotal),
		"lineItems":         formatLineItemsBlock(estimate.LineItems),
		"rawTranscript":     strings.TrimSpace(estimate.RawTranscript),
	}

	out := template
	for key, value := range replacements {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return strings.TrimSpace(out)
}

func displayField(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "___________"
	}
	return value
}

func displayAmount(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "___________"
	}
	return value
}

func formatLineItemsBlock(items []LineItem) string {
	if len(items) == 0 {
		return "(позиции не распознаны — проверьте транскрипт ниже)"
	}

	var b strings.Builder
	b.WriteString("№ | Шифр | Наименование | Ед. | Кол-во | Базис/ед | Базис/общ | Текущ/ед | Текущ/общ\n")
	for _, item := range items {
		seq := item.Seq
		if seq <= 0 {
			seq = 1
		}
		fmt.Fprintf(&b, "%d | %s | %s | %s | %s | %s | %s | %s | %s\n",
			seq,
			displayField(item.Code),
			displayField(item.Description),
			displayField(item.Unit),
			displayAmount(item.Quantity),
			displayAmount(item.BasePricePerUnit),
			displayAmount(item.BasePriceTotal),
			displayAmount(item.CurrentPricePerUnit),
			displayAmount(item.CurrentPriceTotal),
		)
	}
	return strings.TrimRight(b.String(), "\n")
}
