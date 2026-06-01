package estimateintent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// LLMEnricher дополняет regex-парсер через chat-completions (OpenAI-совместимый API).
type LLMEnricher struct {
	apiURL string
	apiKey string
	model  string
	client *http.Client
}

func NewLLMEnricher(apiURL, apiKey, model string, client *http.Client) *LLMEnricher {
	apiURL = strings.TrimSpace(apiURL)
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)
	if apiKey == "" {
		return nil
	}
	if apiURL == "" {
		apiURL = "https://openrouter.ai/api/v1/chat/completions"
	}
	if model == "" {
		model = "google/gemini-2.0-flash-001"
	}
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	return &LLMEnricher{apiURL: apiURL, apiKey: apiKey, model: model, client: client}
}

type llmChatRequest struct {
	Model       string       `json:"model"`
	Temperature float64      `json:"temperature"`
	Messages    []llmMessage `json:"messages"`
}

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmChatResponse struct {
	Choices []struct {
		Message llmMessage `json:"message"`
	} `json:"choices"`
}

// llmEstimateDTO — только поля шапки/таблицы для JSON от модели.
type llmEstimateDTO struct {
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
}

// Enrich вызывает LLM и сливает результат с base (regex). При ошибке возвращает base.
func (e *LLMEnricher) Enrich(ctx context.Context, transcript string, base Estimate) (Estimate, error) {
	if e == nil {
		return base, nil
	}
	dto, err := e.extract(ctx, transcript, base)
	if err != nil {
		return FinalizeEstimate(base), err
	}
	merged := mergeEstimates(base, dtoToEstimate(dto))
	merged.RawTranscript = base.RawTranscript
	return FinalizeEstimate(merged), nil
}

func (e *LLMEnricher) extract(ctx context.Context, transcript string, draft Estimate) (llmEstimateDTO, error) {
	system := `Ты извлекаешь поля локальной строительной сметы (форма №4) из русского транскрипта голосовой диктовки.
Транскрипт может содержать ошибки распознавания (например «наименование строки» = «наименование стройки», «средсанапат у труда» = оплата труда).

Верни ТОЛЬКО один JSON-объект без markdown:
{
  "estimateNumber": "",
  "projectName": "",
  "objectDescription": "",
  "basis": "",
  "estimatedCost": "",
  "laborCosts": "",
  "priceDate": "",
  "approver": "",
  "totalDirectCosts": "",
  "grandTotal": "",
  "lineItems": []
}

Правила:
- Суммы — только цифры, без пробелов и валюты (1250000, 320000).
- Пустые неизвестные поля — пустая строка "".
- lineItems — массив позиций с полями seq, code, description, unit, quantity, basePricePerUnit, basePriceTotal, currentPricePerUnit, currentPriceTotal; если позиций нет — [].
- Не выдумывай данные, которых нет в транскрипте.
- В lineItems.description — только краткое наименование работы (до 80 символов), без повторов и без итогов по смете.
- projectName, objectDescription и все lineItems.description — строго на русском языке, только кириллица (А–Я, а–я, ё). Латиницу не используй: TRC → ТРЦ, Crystal → Кристалл, AP → АР. Дефисы и цифры допустимы.`

	draftJSON, _ := json.Marshal(estimateToDTO(draft))
	user := fmt.Sprintf("Транскрипт:\n%s\n\nЧерновик автопарсера (исправь ошибки, не копируй длинные куски):\n%s",
		strings.TrimSpace(transcript), string(draftJSON))

	body, err := json.Marshal(llmChatRequest{
		Model:       e.model,
		Temperature: 0.1,
		Messages: []llmMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})
	if err != nil {
		return llmEstimateDTO{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.apiURL, bytes.NewReader(body))
	if err != nil {
		return llmEstimateDTO{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return llmEstimateDTO{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return llmEstimateDTO{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return llmEstimateDTO{}, fmt.Errorf("LLM API %d: %s", resp.StatusCode, truncate(string(respBody), 400))
	}

	var chat llmChatResponse
	if err := json.Unmarshal(respBody, &chat); err != nil {
		return llmEstimateDTO{}, err
	}
	if len(chat.Choices) == 0 {
		return llmEstimateDTO{}, fmt.Errorf("LLM: empty choices")
	}

	content := strings.TrimSpace(chat.Choices[0].Message.Content)
	content = stripJSONFence(content)

	var dto llmEstimateDTO
	if err := json.Unmarshal([]byte(content), &dto); err != nil {
		return llmEstimateDTO{}, fmt.Errorf("LLM JSON: %w (raw=%s)", err, truncate(content, 200))
	}
	return dto, nil
}

func dtoToEstimate(d llmEstimateDTO) Estimate {
	items := d.LineItems
	if items == nil {
		items = []LineItem{}
	}
	return Estimate{
		EstimateNumber:    strings.TrimSpace(d.EstimateNumber),
		ProjectName:       strings.TrimSpace(d.ProjectName),
		ObjectDescription: strings.TrimSpace(d.ObjectDescription),
		Basis:             strings.TrimSpace(d.Basis),
		EstimatedCost:     normalizeAmount(d.EstimatedCost),
		LaborCosts:        normalizeAmount(d.LaborCosts),
		PriceDate:         strings.TrimSpace(d.PriceDate),
		Approver:          strings.TrimSpace(d.Approver),
		TotalDirectCosts:  normalizeAmount(d.TotalDirectCosts),
		GrandTotal:        normalizeAmount(d.GrandTotal),
		LineItems:         items,
	}
}

// mergeEstimates: LLM заполняет пустое; для сумм LLM заменяет явно неверные короткие regex-значения.
func mergeEstimates(base, llm Estimate) Estimate {
	out := base
	out.EstimateNumber = pickField(base.EstimateNumber, llm.EstimateNumber)
	out.ProjectName = pickField(base.ProjectName, llm.ProjectName)
	out.ObjectDescription = pickField(base.ObjectDescription, llm.ObjectDescription)
	out.Basis = pickField(base.Basis, llm.Basis)
	out.EstimatedCost = pickMoney(base.EstimatedCost, llm.EstimatedCost)
	out.LaborCosts = pickMoney(base.LaborCosts, llm.LaborCosts)
	out.PriceDate = pickField(base.PriceDate, llm.PriceDate)
	out.Approver = pickField(base.Approver, llm.Approver)
	out.TotalDirectCosts = pickMoney(base.TotalDirectCosts, llm.TotalDirectCosts)
	out.GrandTotal = pickMoney(base.GrandTotal, llm.GrandTotal)
	switch {
	case saneLineItems(llm.LineItems):
		out.LineItems = llm.LineItems
	case saneLineItems(base.LineItems):
		out.LineItems = base.LineItems
	default:
		out.LineItems = []LineItem{}
	}
	return out
}

func estimateToDTO(e Estimate) llmEstimateDTO {
	return llmEstimateDTO{
		EstimateNumber:    e.EstimateNumber,
		ProjectName:       e.ProjectName,
		ObjectDescription: e.ObjectDescription,
		Basis:             e.Basis,
		EstimatedCost:     e.EstimatedCost,
		LaborCosts:        e.LaborCosts,
		PriceDate:         e.PriceDate,
		Approver:          e.Approver,
		TotalDirectCosts:  e.TotalDirectCosts,
		GrandTotal:        e.GrandTotal,
		LineItems:         e.LineItems,
	}
}

func pickField(base, llm string) string {
	base = strings.TrimSpace(base)
	llm = strings.TrimSpace(llm)
	if llm == "" {
		return base
	}
	if base == "" || fieldLooksBroken(base) {
		return llm
	}
	if len(llm) > len(base)+8 && !fieldLooksBroken(llm) {
		return llm
	}
	return base
}

func fieldLooksBroken(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	if len(s) <= 2 {
		return true
	}
	if strings.HasPrefix(strings.ToLower(s), "у,") {
		return true
	}
	return false
}

func pickMoney(base, llm string) string {
	base = strings.TrimSpace(base)
	llm = normalizeAmount(llm)
	if llm == "" {
		return base
	}
	if base == "" || moneyLooksWrong(base, llm) || base == "1" {
		return llm
	}
	return base
}

func moneyLooksWrong(base, llm string) bool {
	b := compactDigits(base)
	l := compactDigits(llm)
	if b == "" || l == "" {
		return false
	}
	if len(b) <= 2 && len(l) >= 4 {
		return true
	}
	return false
}

func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	s = regexp.MustCompile("(?is)^```(?:json)?\\s*").ReplaceAllString(s, "")
	s = regexp.MustCompile("(?is)\\s*```\\s*$").ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// ParseWithEnricher — regex, затем опционально LLM.
func ParseWithEnricher(ctx context.Context, transcript string, enricher *LLMEnricher) Estimate {
	base := Parse(transcript)
	if enricher == nil {
		return base
	}
	merged, err := enricher.Enrich(ctx, transcript, base)
	if err != nil {
		return FinalizeEstimate(base)
	}
	return merged
}
