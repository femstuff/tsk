package estimateintent

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

var (
	reEstimateNumber = regexp.MustCompile(`(?i)(?:номер\s+смет[ыи]|смет[аы]\s+номер|смет[аы]\s+№|№\s*смет[ыи]?|локальн(?:ый|ого)\s+смет(?:ный|ного)?\s+расч[её]т\s*(?:№|номер)?)\s*[:—\-]?\s*([^\.,;\n]+)`)
	reProjectName    = regexp.MustCompile(`(?i)(?:наименование\s+стройк[ии]|стройк[аеи]\s+[:—\-]?|объект\s+строительства)\s*[:—\-]?\s*([^\.,;\n]+)`)
	reObjectDesc     = regexp.MustCompile(`(?i)(?:наименование\s+(?:работ|объекта)|работ[ыа]\s+и\s+затрат[ыа]|на\s+объект[е]?)\s*[:—\-]?\s*([^\.,;\n]+)`)
	reBasis          = regexp.MustCompile(`(?i)(?:основани[ея]|чертеж[иа])\s*(?:№|номер)?\s*[:—\-]?\s*([^\.,;\n]+)`)
	reEstimatedCost  = regexp.MustCompile(`(?i)(?:сметн(?:ая|ой)\s+стоимость|стоимость\s+смет[ыи]|итого\s+по\s+смет[еи])\s*[:—\-]?\s*([\d\s]+(?:[.,]\d+)?)\s*(?:руб|₽)?`)
	reLaborCosts     = regexp.MustCompile(`(?i)(?:средств[ао]\s+на\s+оплат[уы]\s+труд[аа]|оплат[аы]\s+труд[аа]|фот|заработн(?:ая|ой)\s+плат[аы])\s*[:—\-]?\s*([\d\s]+(?:[.,]\d+)?)\s*(?:руб|₽)?`)
	rePriceDate      = regexp.MustCompile(`(?i)(?:состояни[ею]\s+на|цен[ыа]\s+на|составлен[ао]?\s+(?:в\s+)?(?:текущих|прогнозных)?\s*цен(?:ах)?\s*(?:по\s+состоянию\s+на)?)\s*[:—\-]?\s*([^\.,;\n]+)`)
	reApprover       = regexp.MustCompile(`(?i)(?:утвержда(?:ет|ю)|составил[а]?|подпис[аь])\s*[:—\-]?\s*([^\.,;\n]+)`)
	reTotalDirect    = regexp.MustCompile(`(?i)итого\s+прям(?:ые|ых)\s+затрат[ыа]?\s*(?:по\s+смет[еи])?\s*[:—\-]?\s*([\d\s]+(?:[.,]\d+)?)\s*(?:руб|₽)?`)
	reGrandTotal     = regexp.MustCompile(`(?i)(?:всего\s+по\s+смет[еи]|итого\s+всего)\s*[:—\-]?\s*([\d\s]+(?:[.,]\d+)?)\s*(?:руб|₽)?`)

	reLineSplit = regexp.MustCompile(`(?i)(?:^|\s)(?:позици[яию]|строк[аеи]|пункт)\s+(?:номер\s+)?(\d+)`)

	reCode        = regexp.MustCompile(`(?i)(?:шифр|код(?:\s+ресурса)?|норматив)\s*[:—\-]?\s*([^\.,;\n]+)`)
	reDescription = regexp.MustCompile(`(?i)(?:наименование(?:\s+работ)?|работ[ыа])\s*[:—\-]?\s*([^\.,;\n]+)`)
	reUnit        = regexp.MustCompile(`(?i)(?:единиц[аы]\s+измерения|ед(?:\.|\s)?\s*изм)\s*[:—\-]?\s*([^\.,;\n]+)`)
	reQuantity    = regexp.MustCompile(`(?i)(?:кол(?:ичество|-во)|кол-во)\s*[:—\-]?\s*([\d\s]+(?:[.,]\d+)?)`)
	reBasePerUnit = regexp.MustCompile(`(?i)(?:базисн(?:ая|ых|ые)?\s*(?:цен[аы])?\s*(?:на\s+единицу|за\s+единицу)|в\s+базисных\s+ценах\s+на\s+единицу)\s*[:—\-]?\s*([\d\s]+(?:[.,]\d+)?)`)
	reBaseTotal   = regexp.MustCompile(`(?i)(?:базисн(?:ая|ых|ые)?\s*(?:цен[аы])?\s*общ(?:ая|ую|ей)|в\s+базисных\s+ценах\s+общ(?:ая|ую|ей))\s*[:—\-]?\s*([\d\s]+(?:[.,]\d+)?)`)
	reCurPerUnit  = regexp.MustCompile(`(?i)(?:текущ(?:ая|их|ие)?\s*(?:прогнозн(?:ая|ых|ые)?)?\s*(?:цен[аы])?\s*(?:на\s+единицу|за\s+единицу)|в\s+текущих\s+(?:прогнозных\s+)?ценах\s+на\s+единицу)\s*[:—\-]?\s*([\d\s]+(?:[.,]\d+)?)`)
	reCurTotal    = regexp.MustCompile(`(?i)(?:текущ(?:ая|их|ие)?\s*(?:прогнозн(?:ая|ых|ые)?)?\s*(?:цен[аы])?\s*общ(?:ая|ую|ей)|в\s+текущих\s+(?:прогнозных\s+)?ценах\s+общ(?:ая|ую|ей))\s*[:—\-]?\s*([\d\s]+(?:[.,]\d+)?)`)
)

// Parse извлекает из транскрипта поля локальной сметы (форма № 4).
func Parse(transcript string) Estimate {
	text := strings.TrimSpace(transcript)
	out := Estimate{
		RawTranscript: text,
		LineItems:     []LineItem{},
	}

	if text == "" {
		return out
	}

	out.EstimateNumber = firstMatch(reEstimateNumber, text)
	out.ProjectName = firstMatch(reProjectName, text)
	out.ObjectDescription = firstMatch(reObjectDesc, text)
	out.Basis = firstMatch(reBasis, text)
	out.EstimatedCost = normalizeAmount(firstMatch(reEstimatedCost, text))
	out.LaborCosts = normalizeAmount(firstMatch(reLaborCosts, text))
	out.PriceDate = firstMatch(rePriceDate, text)
	out.Approver = firstMatch(reApprover, text)
	out.TotalDirectCosts = normalizeAmount(firstMatch(reTotalDirect, text))
	out.GrandTotal = normalizeAmount(firstMatch(reGrandTotal, text))

	out.LineItems = parseLineItems(text)

	if out.GrandTotal == "" && out.EstimatedCost != "" {
		out.GrandTotal = out.EstimatedCost
	}

	return out
}

func parseLineItems(text string) []LineItem {
	indices := reLineSplit.FindAllStringSubmatchIndex(text, -1)
	if len(indices) == 0 {
		if chunk := strings.TrimSpace(text); chunk != "" {
			if item := parseLineItemChunk(1, chunk); itemHasData(item) {
				return []LineItem{item}
			}
		}
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
	return LineItem{
		Seq:                 seq,
		Code:                firstMatch(reCode, chunk),
		Description:         firstMatch(reDescription, chunk),
		Unit:                firstMatch(reUnit, chunk),
		Quantity:            normalizeAmount(firstMatch(reQuantity, chunk)),
		BasePricePerUnit:    normalizeAmount(firstMatch(reBasePerUnit, chunk)),
		BasePriceTotal:      normalizeAmount(firstMatch(reBaseTotal, chunk)),
		CurrentPricePerUnit: normalizeAmount(firstMatch(reCurPerUnit, chunk)),
		CurrentPriceTotal:   normalizeAmount(firstMatch(reCurTotal, chunk)),
	}
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

func firstMatch(re *regexp.Regexp, text string) string {
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func normalizeAmount(value string) string {
	value = strings.TrimSpace(value)
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
