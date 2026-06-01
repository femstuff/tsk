package estimateintent

import (
	"regexp"
	"strings"
)

// FinalizeEstimate — выравнивание сумм и очистка после regex/LLM.
func FinalizeEstimate(e Estimate) Estimate {
	e.EstimateNumber = strings.TrimSpace(e.EstimateNumber)
	e.ProjectName = strings.TrimSpace(e.ProjectName)
	e.ObjectDescription = trimLeadingJunk(e.ObjectDescription)
	e.ObjectDescription = fixConstructionTypos(e.ObjectDescription)
	e.Basis = strings.TrimSpace(e.Basis)
	e.PriceDate = clipPriceDate(e.PriceDate)
	e.Approver = strings.TrimSpace(e.Approver)

	e.EstimatedCost = normalizeAmount(e.EstimatedCost)
	e.LaborCosts = normalizeAmount(e.LaborCosts)
	e.TotalDirectCosts = normalizeAmount(e.TotalDirectCosts)
	e.GrandTotal = normalizeAmount(e.GrandTotal)

	e = reconcileMoneyFields(e)

	if e.ProjectName == "" {
		e.ProjectName = guessProjectName(e.RawTranscript)
	}
	if e.EstimateNumber == "" {
		e.EstimateNumber = guessEstimateNumber(e.RawTranscript)
	}

	e.LineItems = sanitizeLineItems(e.LineItems)
	if e.LineItems == nil {
		e.LineItems = []LineItem{}
	}
	e = ensureRussianObjectNames(e)
	return e
}

func reconcileMoneyFields(e Estimate) Estimate {
	if moneyLooksWrong(e.EstimatedCost, e.GrandTotal) && e.GrandTotal != "" {
		e.EstimatedCost = e.GrandTotal
	}
	if e.EstimatedCost != "" && e.GrandTotal == "" {
		e.GrandTotal = e.EstimatedCost
	}
	if e.LaborCosts != "" && len(compactDigits(e.LaborCosts)) <= 2 {
		if v := extractLaborFallback(normalizeTranscript(e.RawTranscript)); v != "" {
			e.LaborCosts = v
		}
	}
	return e
}

func trimLeadingJunk(s string) string {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)
	for _, prefix := range []string{"у, ", "у,", "о, ", "а, "} {
		if strings.HasPrefix(lower, prefix) {
			s = strings.TrimSpace(s[len(prefix):])
			lower = strings.ToLower(s)
		}
	}
	return strings.TrimSpace(s)
}

func guessProjectName(transcript string) string {
	text := normalizeTranscript(transcript)
	if v := extractAfter(text, reProjectNameStart); v != "" {
		return trimField(v)
	}
	re := regexp.MustCompile(`(?i)(?:трц[\s\-]*[а-яa-z0-9\-]+)`)
	if m := re.FindString(transcript); m != "" {
		return trimField(m)
	}
	return ""
}

func guessEstimateNumber(transcript string) string {
	text := normalizeTranscript(transcript)
	return parseEstimateNumber(text)
}

func sanitizeLineItems(items []LineItem) []LineItem {
	if len(items) == 0 {
		return items
	}
	out := make([]LineItem, 0, len(items))
	for _, item := range items {
		item.Description = fixConstructionTypos(trimLineDescription(item.Description))
		item.Unit = trimLineUnit(item.Unit)
		if !lineItemSane(item) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func lineItemSane(item LineItem) bool {
	desc := strings.TrimSpace(item.Description)
	if desc == "" && strings.TrimSpace(item.Code) == "" {
		return false
	}
	if len(desc) > 200 {
		return false
	}
	if descriptionLooksDuplicated(desc) {
		return false
	}
	return true
}

// descriptionLooksDuplicated — типичный сбой LLM: один и тот же абзац дважды подряд.
func descriptionLooksDuplicated(s string) bool {
	s = collapseSpaces(strings.ToLower(s))
	if len(s) < 80 {
		return false
	}
	mid := len(s) / 2
	half := strings.TrimSpace(s[:mid])
	rest := strings.TrimSpace(s[mid:])
	if len(half) < 40 {
		return false
	}
	return half == rest || strings.HasPrefix(rest, half)
}

func collapseSpaces(s string) string {
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(s, " "))
}

func trimLineDescription(desc string) string {
	desc = trimField(desc)
	if loc := regexp.MustCompile(`(?i)\s+единиц[аы]\s+измерения`).FindStringIndex(desc); loc != nil && loc[0] > 0 {
		desc = trimField(desc[:loc[0]])
	}
	if loc := regexp.MustCompile(`(?i)\s+кол(?:ичество|-во)\s`).FindStringIndex(desc); loc != nil && loc[0] > 0 {
		desc = trimField(desc[:loc[0]])
	}
	return desc
}

func trimLineUnit(unit string) string {
	unit = trimField(unit)
	if loc := regexp.MustCompile(`(?i)\s+кол(?:ичество|-во)`).FindStringIndex(unit); loc != nil && loc[0] > 0 {
		unit = trimField(unit[:loc[0]])
	}
	return unit
}

func saneLineItems(items []LineItem) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if !lineItemSane(item) {
			return false
		}
	}
	return true
}
