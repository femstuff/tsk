package estimateintent

import (
	"regexp"
	"strconv"
	"strings"
)

// Маркеры следующего поля — для ограничения захвата (голос без точек и запятых).
var reNextFieldMarker = regexp.MustCompile(`(?i)\s+(?:наименование\s+стройк|наименование\s+строк|наименование\s+работ|на\s+именование\s+(?:стройк|строк|работ)|работ[ыа]\s+и\s+затрат|основани|чертеж|сметн(?:ая|ой)\s+стоимость|стоимость\s+смет|средств[ао]?\s+на\s+оплат|средств[ао]?\s+на\s+опаду|средс[а-яё]*\s+у\s+труд|оплат[аы]\s+труд|составлен|состав\s+текущих|состав\s+у\s+|составил|составый|состояни[ею]\s+на|утвержд|единиц[аы]\s+измерения|ед\.?\s*изм|кол(?:ичество|-во)|базисн|текущ|позици[яию]\s+(?:номер\s+)?\d|итого\s+прям|всего\s+по\s+смет)`)

// Типичные огрехи Whisper перед разметкой границ полей.
var whisperAliases = []struct{ pattern, replacement string }{
	{`(?i)на\s+именование\s+строк[ии]`, "наименование стройки"},
	{`(?i)на\s+именование\s+стройк[ии]`, "наименование стройки"},
	{`(?i)на\s+именование\s+работ`, "наименование работ"},
	{`(?i)наименование\s+строк[ии]`, "наименование стройки"},
	{`(?i)средств[ао]?\s+на\s+опаду\s+труда`, "средства на оплату труда"},
	{`(?i)средств[ао]?\s+на\s+оплату\s+труда`, "средства на оплату труда"},
	{`(?i)средс[а-яё]*\s+у\s+труда`, "средства на оплату труда"},
	{`(?i)средств[ао]?\s+напат[а-яё]*\s+у\s+труда`, "средства на оплату труда"},
	{`(?i)средсанапат\s+у\s+труда`, "средства на оплату труда"},
	{`(?i)состав\s+текущих\s+ценах`, "составлен в текущих ценах"},
	{`(?i)составлен\s+текущих\s+ценах`, "составлен в текущих ценах"},
	{`(?i)состав\s+у\s+инженер`, "составил инженер"},
	{`(?i)составый\s+инженер`, "составил инженер"},
}

var boundaryKeywords = []string{
	"наименование стройки",
	"наименование работ",
	"наименование работ и затрат",
	"основание",
	"чертежи",
	"чертежи номер",
	"сметная стоимость",
	"средства на оплату труда",
	"оплата труда",
	"составлен",
	"составлена",
	"составил",
	"позиция",
	"позицию",
	"итого прямые",
	"всего по смете",
}

func normalizeTranscript(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}
	text = strings.ReplaceAll(text, "ё", "е")
	text = strings.ReplaceAll(text, "Ё", "Е")
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	for _, a := range whisperAliases {
		text = regexp.MustCompile(a.pattern).ReplaceAllString(text, a.replacement)
	}

	for _, kw := range boundaryKeywords {
		re := regexp.MustCompile(`(?i)([^\s.])\s+(` + regexp.QuoteMeta(kw) + `)`)
		text = re.ReplaceAllString(text, `$1. $2`)
	}
	text = regexp.MustCompile(`(?i)\s+позици[яию]\s+`).ReplaceAllString(text, ". позиция ")
	text = regexp.MustCompile(`(?i)\s+позици[яию]\s+номер\s+`).ReplaceAllString(text, ". позиция ")
	return text
}

func trimField(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, ".,;:-—")
	value = regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

func clipAtNextField(value string) string {
	value = trimField(value)
	if value == "" {
		return ""
	}
	if loc := reNextFieldMarker.FindStringIndex(value); loc != nil && loc[0] > 0 {
		value = value[:loc[0]]
	}
	return trimField(value)
}

// fixConstructionTypos — частые огрехи Whisper в блоке работ.
func fixConstructionTypos(value string) string {
	if value == "" {
		return value
	}
	repl := []struct{ pattern, replacement string }{
		{`(?i)\bустройста\b`, "устройства"},
		{`(?i)\bкровал[иейюя]\b`, "кровли"},
		{`(?i)\bгидроизаляци[яию]\b`, "гидроизоляция"},
	}
	for _, r := range repl {
		value = regexp.MustCompile(r.pattern).ReplaceAllString(value, r.replacement)
	}
	return value
}

func extractAfter(text string, start *regexp.Regexp) string {
	loc := start.FindStringSubmatchIndex(text)
	if loc == nil || len(loc) < 2 {
		return ""
	}
	rest := text[loc[1]:]
	if strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, "—") || strings.HasPrefix(rest, "-") {
		rest = strings.TrimLeft(rest, ":—- \t")
	}
	return clipAtNextField(rest)
}

var (
	reDigitsInPhrase      = regexp.MustCompile(`\d[\d\s]*\d|\d+`)
	reGroupedMoneyDigits  = regexp.MustCompile(`\d{1,3}(?:[,\s]\d{3})+`)
)

// parseMoneyPhrase — цифры или «миллион 250 тысяч рублей», «980 тысяч», «1 250 000», «1,250,000».
func parseMoneyPhrase(phrase string) string {
	phrase = strings.ToLower(trimField(phrase))
	if phrase == "" {
		return ""
	}

	if m := reGroupedMoneyDigits.FindString(phrase); m != "" {
		if v := compactDigits(m); len(v) >= 4 {
			return v
		}
	}

	total := int64(0)
	million := regexp.MustCompile(`(\d+)\s*миллион`).FindStringSubmatch(phrase)
	if len(million) >= 2 {
		if n, err := strconv.ParseInt(million[1], 10, 64); err == nil {
			total += n * 1_000_000
		}
	} else if strings.Contains(phrase, "миллион") {
		total += 1_000_000
	}

	thousand := regexp.MustCompile(`(\d+)\s*тысяч`).FindStringSubmatch(phrase)
	if len(thousand) >= 2 {
		if n, err := strconv.ParseInt(thousand[1], 10, 64); err == nil {
			total += n * 1_000
		}
	} else if strings.Contains(phrase, "тысяч") {
		// «320 тысяч» без явной цифры перед тысяч — ищем число перед словом
		if m := regexp.MustCompile(`(\d+)\s*тысяч`).FindStringSubmatch(phrase); len(m) >= 2 {
			if n, err := strconv.ParseInt(m[1], 10, 64); err == nil {
				total += n * 1_000
			}
		}
	}

	if total > 0 {
		return strconv.FormatInt(total, 10)
	}
	if m := reDigitsInPhrase.FindString(phrase); m != "" {
		return compactDigits(m)
	}
	return ""
}

func compactDigits(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func extractMoneyAfter(text string, start *regexp.Regexp) string {
	loc := start.FindStringIndex(text)
	if loc == nil {
		return ""
	}
	chunk := text[loc[1]:]
	if end := reNextFieldMarker.FindStringIndex(chunk); end != nil {
		chunk = chunk[:end[0]]
	}
	chunk = trimField(chunk)
	// убрать префикс метки до первой цифры или «миллион»
	if idx := regexp.MustCompile(`(?i)(?:\d|миллион|тысяч)`).FindStringIndex(chunk); idx != nil {
		chunk = chunk[idx[0]:]
	}
	return parseMoneyPhrase(chunk)
}
