package estimateintent

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

var (
	reEstimateNumberStart = regexp.MustCompile(`(?i)(?:локальн(?:ый|ого)\s+смет(?:ный|ного)?\s+расч[её]т\s*)?(?:номер\s+смет[ыи]|смет[аы]\s+номер|смет[аы]\s+№|№\s*смет[ыи]?)`)
	reEstimateNumberDigits = regexp.MustCompile(`(?i)(?:номер\s+смет[ыи]|смет[аы]\s+номер|расч[её]т\s+номер|номер)\s*[:—\-]?\s*(\d{1,8})\b`)

	reProjectNameStart = regexp.MustCompile(`(?i)(?:наименование\s+(?:стройк[ии]|строк[ии])|на\s+именование\s+(?:стройк[ии]|строк[ии])|стройк[аеи]\b)`)
	reObjectDescStart  = regexp.MustCompile(`(?i)(?:наименование\s+(?:работ|объекта)|на\s+именование\s+работ|работ[ыа]\s+и\s+затрат[ыа]|на\s+объект[е]?|работ[ыа]\b)`)
	reBasisStart       = regexp.MustCompile(`(?i)(?:основани[ея]|чертеж[иа])`)

	reEstimatedCostStart = regexp.MustCompile(`(?i)(?:сметн(?:ая|ой)\s+стоимость|стоимость\s+смет[ыи])`)
	reLaborCostsStart    = regexp.MustCompile(`(?i)(?:средств[ао]?\s+на\s+(?:оплат[уы]\s+труд[аа]|опаду\s+труд[аа])|средс[а-яё]*\s+у\s+труд[аа]|оплат[аы]\s+труд[аа]|(?:^|\.\s*)у\s+труда|фот|заработн(?:ая|ой)\s+плат[аы])`)
	rePriceDateStart     = regexp.MustCompile(`(?i)(?:составлен[ао]?\s+(?:в\s+)?(?:текущих|прогнозных)?\s*цен(?:ах)?\s*(?:по\s+состоянию\s+на)?|состав\s+текущих\s+цен(?:ах)?\s+по\s+состоянию\s+на|состояни[ею]\s+на|цен[ыа]\s+на)`)
	reApproverStart      = regexp.MustCompile(`(?i)(?:составил[а]?|составый|состав\s+у\s+|инженер[\s\-]*сметчик|утвержда(?:ет|ю)|подпис[аь])`)

	reTotalDirectStart = regexp.MustCompile(`(?i)итого\s+прям(?:ые|ых)\s+затрат[ыа]?(?:\s+по\s+смет[еи])?`)
	reGrandTotalStart  = regexp.MustCompile(`(?i)(?:всего\s+по\s+смет[еи]|итого\s+всего)`)

	reLineSplit = regexp.MustCompile(`(?i)(?:^|\.|\s)(?:позици[яию]|строк[аеи]|пункт)\s+(?:номер\s+)?(\d+)`)

	reCodeInChunk        = regexp.MustCompile(`(?i)(?:шифр|код(?:\s+ресурса)?)\s*[:—\-]?\s*([0-9]{2}[\-–—]?[0-9]{2}[\-–—]?[0-9]{0,6})`)
	reDescriptionInChunk = regexp.MustCompile(`(?i)(?:наименование(?:\s+работ)?|работ[ыа])\s*[:—\-]?\s*`)
	reUnitInChunk        = regexp.MustCompile(`(?i)(?:единиц[аы]\s+измерения|ед(?:\.|\s)?\s*изм|измерения)\s*[:—\-]?\s*`)
	reQuantityInChunk    = regexp.MustCompile(`(?i)(?:кол(?:ичество|-во)|кол-во)\s*[:—\-]?\s*(\d[\d\s]*)`)
	reBasePerUnitChunk   = regexp.MustCompile(`(?i)базисн(?:ая|ых|ые)?\s*(?:цен[аы])?\s*(?:на\s+единицу|за\s+единицу)\s*[:—\-]?\s*([\d\s]+)`)
	reBaseTotalChunk     = regexp.MustCompile(`(?i)базисн(?:ая|ых|ые)?\s*(?:цен[аы])?\s*общ(?:ая|ую|ей)\s*[:—\-]?\s*([\d\s]+)`)
	reCurPerUnitChunk    = regexp.MustCompile(`(?i)текущ(?:ая|их|ие)?\s*(?:прогнозн(?:ая|ых|ые)?)?\s*(?:цен[аы])?\s*(?:на\s+единицу|за\s+единицу)\s*[:—\-]?\s*([\d\s]+)`)
	reCurTotalChunk      = regexp.MustCompile(`(?i)текущ(?:ая|их|ие)?\s*(?:прогнозн(?:ая|ых|ые)?)?\s*(?:цен[аы])?\s*общ(?:ая|ую|ей)\s*[:—\-]?\s*([\d\s]+)`)
)

// Parse извлекает из транскрипта поля локальной сметы (форма № 4).
func Parse(transcript string) Estimate {
	raw := strings.TrimSpace(transcript)
	text := normalizeTranscript(raw)
	out := Estimate{
		RawTranscript: raw,
		LineItems:     []LineItem{},
	}

	if text == "" {
		return out
	}

	out.EstimateNumber = parseEstimateNumber(text)
	out.ProjectName = extractAfter(text, reProjectNameStart)
	out.ObjectDescription = fixConstructionTypos(extractAfter(text, reObjectDescStart))
	out.Basis = extractBasis(text)
	out.EstimatedCost = extractMoneyAfter(text, reEstimatedCostStart)
	out.LaborCosts = extractMoneyAfter(text, reLaborCostsStart)
	if out.LaborCosts == "" {
		out.LaborCosts = extractLaborFallback(text)
	}
	out.PriceDate = clipPriceDate(extractAfter(text, rePriceDateStart))
	out.Approver = extractApprover(text)
	out.TotalDirectCosts = extractMoneyAfter(text, reTotalDirectStart)
	out.GrandTotal = extractMoneyAfter(text, reGrandTotalStart)

	out.LineItems = parseLineItems(text)

	if out.GrandTotal == "" && out.EstimatedCost != "" {
		out.GrandTotal = out.EstimatedCost
	}
	if out.EstimatedCost == "" && out.GrandTotal != "" {
		out.EstimatedCost = out.GrandTotal
	}

	return FinalizeEstimate(out)
}

func parseEstimateNumber(text string) string {
	if m := reEstimateNumberDigits.FindStringSubmatch(text); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	chunk := extractAfter(text, reEstimateNumberStart)
	chunk = clipAtNextField(chunk)
	if m := regexp.MustCompile(`\d{1,8}`).FindString(chunk); m != "" {
		return m
	}
	return trimField(chunk)
}

func clipPriceDate(value string) string {
	value = trimField(value)
	if value == "" {
		return ""
	}
	if loc := regexp.MustCompile(`(?i)[\s.,]+состав[а-яё]*`).FindStringIndex(value); loc != nil && loc[0] > 0 {
		value = value[:loc[0]]
	}
	if loc := regexp.MustCompile(`(?i)\s+инженер`).FindStringIndex(value); loc != nil && loc[0] > 0 {
		value = value[:loc[0]]
	}
	value = regexp.MustCompile(`(?i)[\s.,]+состав[а-яё]*\s*$`).ReplaceAllString(value, "")
	return trimField(value)
}

func extractLaborFallback(text string) string {
	// «… у труда 320 тысяч …» после сбоя Whisper.
	if m := regexp.MustCompile(`(?i)у\s+труда\s+(.{0,80})`).FindStringSubmatch(text); len(m) >= 2 {
		if v := parseMoneyPhrase(m[1]); v != "" {
			return v
		}
	}
	return ""
}

func extractApprover(text string) string {
	if v := extractAfter(text, reApproverStart); v != "" {
		v = regexp.MustCompile(`(?i)^у\s+`).ReplaceAllString(v, "")
		return trimField(v)
	}
	return ""
}

func extractBasis(text string) string {
	chunk := extractAfter(text, reBasisStart)
	chunk = regexp.MustCompile(`(?i)^(?:чертежи|чертеж)\s*(?:номер)?\s*`).ReplaceAllString(chunk, "")
	return trimField(chunk)
}

func parseLineItems(text string) []LineItem {
	indices := reLineSplit.FindAllStringSubmatchIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	items := make([]LineItem, 0, len(indices))
	for i, loc := range indices {
		seq := 1
		if len(loc) >= 4 {
			if n, err := strconv.Atoi(strings.TrimSpace(text[loc[2]:loc[3]])); err == nil && n > 0 {
				seq = n
			}
		}
		start := loc[0]
		if start > 0 && text[start-1] == '.' {
			start--
		}
		end := len(text)
		if i+1 < len(indices) {
			end = indices[i+1][0]
		}
		chunk := strings.TrimSpace(text[start:end])
		item := parseLineItemChunk(seq, chunk)
		if itemHasData(item) {
			items = append(items, item)
		}
	}
	return items
}

func parseLineItemChunk(seq int, chunk string) LineItem {
	item := LineItem{Seq: seq}

	if m := reCodeInChunk.FindStringSubmatch(chunk); len(m) >= 2 {
		item.Code = trimField(m[1])
	}

	item.Description = extractDescriptionFromChunk(chunk)
	item.Unit = extractAfter(chunk, reUnitInChunk)
	if item.Unit == "" {
		if m := regexp.MustCompile(`(?i)(?:квадратный\s+метр|м2|м\s*2)`).FindString(chunk); m != "" {
			item.Unit = "м2"
		}
	}

	if m := reQuantityInChunk.FindStringSubmatch(chunk); len(m) >= 2 {
		item.Quantity = normalizeAmount(m[1])
	}

	item.BasePricePerUnit = normalizeAmount(firstMatchSub(reBasePerUnitChunk, chunk))
	item.BasePriceTotal = normalizeSpokenOrDigits(firstMatchSub(reBaseTotalChunk, chunk))
	item.CurrentPricePerUnit = normalizeAmount(firstMatchSub(reCurPerUnitChunk, chunk))
	item.CurrentPriceTotal = normalizeSpokenOrDigits(firstMatchSub(reCurTotalChunk, chunk))

	return item
}

func extractDescriptionFromChunk(chunk string) string {
	loc := reDescriptionInChunk.FindStringIndex(chunk)
	if loc == nil {
		// «шифр … наименование монтаж кровли» без слова наименование — текст после шифра до единица/количество
		if m := reCodeInChunk.FindStringSubmatchIndex(chunk); len(m) >= 4 {
			rest := strings.TrimSpace(chunk[m[3]:])
			if strings.HasPrefix(strings.ToLower(rest), "наименование") {
				rest = strings.TrimSpace(rest[len("наименование"):])
			}
			return clipAtNextField(rest)
		}
		return ""
	}
	rest := chunk[loc[1]:]
	return clipAtNextField(rest)
}

func firstMatchSub(re *regexp.Regexp, text string) string {
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func normalizeSpokenOrDigits(value string) string {
	if v := parseMoneyPhrase(value); v != "" {
		return v
	}
	return normalizeAmount(value)
}

func itemHasData(item LineItem) bool {
	return strings.TrimSpace(item.Code) != "" ||
		strings.TrimSpace(item.Description) != "" ||
		strings.TrimSpace(item.Unit) != "" ||
		strings.TrimSpace(item.Quantity) != "" ||
		strings.TrimSpace(item.BasePricePerUnit) != "" ||
		strings.TrimSpace(item.BasePriceTotal) != "" ||
		strings.TrimSpace(item.CurrentPricePerUnit) != "" ||
		strings.TrimSpace(item.CurrentPriceTotal) != ""
}

func normalizeAmount(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "миллион") || strings.Contains(lower, "тысяч") {
		if v := parseMoneyPhrase(value); v != "" {
			return v
		}
	}
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, ",", ".")
	return value
}

// MarshalPayload сериализует смету в JSON для хранения в заявке.
func MarshalPayload(estimate Estimate) (string, error) {
	raw, err := json.Marshal(Payload{Estimate: estimate})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// UnmarshalPayload читает JSON из заявки; при ошибке возвращает смету только с транскript.
func UnmarshalPayload(payload string) Estimate {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return Estimate{LineItems: []LineItem{}}
	}
	if strings.HasPrefix(payload, "{") {
		var wrapped Payload
		if err := json.Unmarshal([]byte(payload), &wrapped); err == nil {
			if wrapped.Estimate.LineItems == nil {
				wrapped.Estimate.LineItems = []LineItem{}
			}
			return wrapped.Estimate
		}
	}
	return Parse(payload)
}

// IsEstimateCategory проверяет, что шаблон относится к сметам.
func IsEstimateCategory(category string) bool {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "estimate", "estimates", "smeta", "смета", "сметы":
		return true
	default:
		return false
	}
}
