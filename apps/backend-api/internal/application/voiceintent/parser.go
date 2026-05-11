package voiceintent

import (
	"regexp"
	"strconv"
	"strings"
)

// Action — упрощённое намерение после транскрипции (рус/англ ключевые слова).
const (
	ActionNone       = "none"
	ActionMoveNext   = "move_next"
	ActionMovePrev   = "move_prev"
	ActionMoveStage  = "move_stage"
	ActionCreateTask = "create_task"
)

type Intent struct {
	Action     string
	DealID     int
	DealTitle  string // название для поиска в Bitrix (crm.deal.list)
	StageHint  string
	TaskTitle  string
	RawSummary string
}

var reDealID = regexp.MustCompile(`(?i)(?:сделк[аеиу]?\s*(?:№|номер|id)?\s*|deal\s*#?\s*|id\s*сделк[аи]?\s*)(\d{1,12})`)
var reBareNumber = regexp.MustCompile(`\b(\d{3,12})\b`)
var reGuillemetTitle = regexp.MustCompile(`«([^»]{2,200})»`)
var reQuotedTitle = regexp.MustCompile(`"([^"]{2,200})"`)
// «сделку тест на следующий этап» — явно «на следующ…», иначе reDealThenCommand съедает «на» в название.
var reDealOnNextStage = regexp.MustCompile(
	`(?i)(?:(?:перемести|переведи|перенеси|перевести|move)\s+)?сделк[ауeи]\s+(.+?)\s+на\s+следующ`,
)
var reDealThenCommand = regexp.MustCompile(
	`(?i)сделк[ауeи]\s+(.+?)\s+(?:следующ(?:ий|ую|его)?|назад|на\s+стади|на\s+этап|перевед|дальше|вперёд|вперед)`,
)
var reDealNextNoNa = regexp.MustCompile(
	`(?i)сделк[ауeи]\s+(.+?)\s+следующ(?:ий|ую)?(?:\s+этап|\s+стад|\s+шаг)?`,
)
var reByName = regexp.MustCompile(`(?i)по\s+назван(?:ию|ию)\s+([^,\.;]{2,120})`)
var reNamedDeal = regexp.MustCompile(`(?i)назван(?:ие|ием)\s+сделки\s+([^,\.;]{2,120})`)

// Parse извлекает из текста: номер сделки, действие (следующая стадия / назад / на стадию / задача).
func Parse(text string, dealIDOverride int) Intent {
	trimmed := strings.TrimSpace(text)
	lower := strings.ToLower(trimmed)

	out := Intent{
		Action:     ActionNone,
		RawSummary: trimmed,
	}

	dealID := dealIDOverride
	if dealID <= 0 {
		if m := reDealID.FindStringSubmatch(trimmed); len(m) > 1 {
			if id, err := strconv.Atoi(m[1]); err == nil {
				dealID = id
			}
		}
	}
	if dealID <= 0 {
		if m := reBareNumber.FindStringSubmatch(trimmed); len(m) > 1 {
			if id, err := strconv.Atoi(m[1]); err == nil && id >= 100 {
				dealID = id
			}
		}
	}
	out.DealID = dealID
	out.DealTitle = extractDealTitleFromSpeech(trimmed)

	if looksLikeCreateTask(lower) {
		out.Action = ActionCreateTask
		out.TaskTitle = extractTaskTitle(trimmed)
		if out.TaskTitle == "" {
			out.TaskTitle = trimmed
		}
		if len(out.TaskTitle) > 240 {
			out.TaskTitle = out.TaskTitle[:240]
		}
		return out
	}

	if strings.Contains(lower, "следующ") || strings.Contains(lower, "дальше") ||
		strings.Contains(lower, "вперёд") || strings.Contains(lower, "вперед") ||
		strings.Contains(lower, "next stage") || strings.Contains(lower, "move next") {
		out.Action = ActionMoveNext
		return finalizeIntent(out)
	}

	if strings.Contains(lower, "назад") || strings.Contains(lower, "предыдущ") ||
		strings.Contains(lower, "previous") || strings.Contains(lower, "move back") {
		out.Action = ActionMovePrev
		return finalizeIntent(out)
	}

	stageHint := extractStageHint(lower, trimmed)
	if stageHint != "" && (strings.Contains(lower, "стади") || strings.Contains(lower, "этап") ||
		strings.Contains(lower, "stage") || strings.Contains(lower, "на ") && stageHint != "") {
		out.Action = ActionMoveStage
		out.StageHint = stageHint
		return finalizeIntent(out)
	}

	// Явные коды стадий без слова «стадия»
	if looksLikeSymbolicStage(lower) {
		out.Action = ActionMoveStage
		out.StageHint = symbolicStageToken(lower)
		return finalizeIntent(out)
	}

	return finalizeIntent(out)
}

func finalizeIntent(out Intent) Intent {
	if strings.TrimSpace(out.DealTitle) == "" {
		out.DealTitle = extractDealTitleFromSpeech(out.RawSummary)
	}
	return out
}

// extractDealTitleFromSpeech — название сделки в кавычках/ёлочках или шаблоны «сделку X … следующий».
func extractDealTitleFromSpeech(s string) string {
	s = strings.TrimSpace(s)
	if m := reGuillemetTitle.FindStringSubmatch(s); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	if m := reQuotedTitle.FindStringSubmatch(s); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	if m := reDealOnNextStage.FindStringSubmatch(s); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	if m := reDealNextNoNa.FindStringSubmatch(s); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	if m := reDealThenCommand.FindStringSubmatch(s); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	if m := reByName.FindStringSubmatch(s); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	if m := reNamedDeal.FindStringSubmatch(s); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func looksLikeCreateTask(lower string) bool {
	return strings.Contains(lower, "создай задач") || strings.Contains(lower, "новая задач") ||
		strings.Contains(lower, "создать задач") || strings.Contains(lower, "добавь задач") ||
		strings.Contains(lower, "create task") || strings.Contains(lower, "add task")
}

func extractTaskTitle(original string) string {
	for _, sep := range []string{":", "—", "-", "–"} {
		if i := strings.Index(original, sep); i >= 0 && i+1 < len(original) {
			return strings.TrimSpace(original[i+1:])
		}
	}
	return strings.TrimSpace(original)
}

func extractStageHint(lower, original string) string {
	tokens := []string{
		"executing", "preparation", "new", "success", "fail",
		"в работе", "подготовк", "новый", "успех", "отказ", "предоплат", "финальн",
	}
	for _, tok := range tokens {
		if strings.Contains(lower, tok) {
			if sym := symbolicStageToken(lower); sym != "" {
				return sym
			}
			return tok
		}
	}
	_ = original
	return ""
}

func looksLikeSymbolicStage(lower string) bool {
	return strings.Contains(lower, "executing") || strings.Contains(lower, "preparation") ||
		strings.Contains(lower, "prepay") || strings.Contains(lower, "final_invoice") ||
		strings.Contains(lower, "success") || strings.Contains(lower, "fail")
}

func symbolicStageToken(lower string) string {
	switch {
	case strings.Contains(lower, "executing") || strings.Contains(lower, "в работе"):
		return "EXECUTING"
	case strings.Contains(lower, "preparation") || strings.Contains(lower, "подготовк"):
		return "PREPARATION"
	case strings.Contains(lower, "prepay") || strings.Contains(lower, "предоплат"):
		return "PREPAYMENT_INVOICE"
	case strings.Contains(lower, "final") || strings.Contains(lower, "финальн"):
		return "FINAL_INVOICE"
	case strings.Contains(lower, "success") || strings.Contains(lower, "успех"):
		return "SUCCESS"
	case strings.Contains(lower, "fail") || strings.Contains(lower, "отказ") || strings.Contains(lower, "проигрыш"):
		return "FAIL"
	case strings.Contains(lower, "new") || strings.Contains(lower, "новый") || strings.Contains(lower, "новая"):
		return "NEW"
	default:
		return ""
	}
}
