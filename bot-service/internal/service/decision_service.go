package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Decision struct {
	ShouldMove    bool   `json:"should_move"`
	DealID        int    `json:"deal_id"`
	DealTitle     string `json:"deal_title"`
	TargetStageID string `json:"target_stage_id"`
	Reason        string `json:"reason"`
}

type DecisionService struct {
	apiURL       string
	apiKey       string
	model        string
	defaultStage string
	client       *http.Client
}

type llmRequest struct {
	Model       string       `json:"model"`
	Temperature float64      `json:"temperature"`
	Messages    []llmMessage `json:"messages"`
}

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmResponse struct {
	Choices []struct {
		Message llmMessage `json:"message"`
	} `json:"choices"`
}

func NewDecisionService(apiURL, apiKey, model, defaultStage string, client *http.Client) *DecisionService {
	if client == nil {
		client = http.DefaultClient
	}

	return &DecisionService{
		apiURL:       apiURL,
		apiKey:       apiKey,
		model:        model,
		defaultStage: defaultStage,
		client:       client,
	}
}

func (s *DecisionService) Analyze(transcribedText string) (Decision, error) {
	if strings.TrimSpace(s.apiKey) == "" {
		return Decision{}, fmt.Errorf("LLM_API_KEY is empty")
	}

	systemPrompt := "Ты помощник по продажам. Проанализируй транскрипт звонка/голосового и верни только JSON без markdown. Структура JSON: {\"should_move\": boolean, \"deal_id\": number, \"deal_title\": string, \"target_stage_id\": string, \"reason\": string}. should_move=true только если есть уверенный признак, что сделку нужно переместить. deal_id=0, если ID сделки явно не назван. deal_title заполняй названием сделки из текста (например '67'), если оно произнесено. target_stage_id используй из контекста или оставь пустым."
	userPrompt := fmt.Sprintf("Транскрипт:\n%s\n\nВерни только JSON.", transcribedText)

	reqBody, err := json.Marshal(llmRequest{
		Model:       s.model,
		Temperature: 0,
		Messages: []llmMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return Decision{}, err
	}

	req, err := http.NewRequest(http.MethodPost, s.apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return Decision{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	if isOpenRouterURL(s.apiURL) {
		req.Header.Set("HTTP-Referer", "https://local.transcribation")
		req.Header.Set("X-Title", "telegram-transcription-bot")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return Decision{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Decision{}, err
	}

	if resp.StatusCode >= 300 {
		return Decision{}, fmt.Errorf("llm status: %s, body: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var llmResp llmResponse
	if err := json.Unmarshal(respBody, &llmResp); err != nil {
		return Decision{}, err
	}
	if len(llmResp.Choices) == 0 {
		return Decision{}, fmt.Errorf("llm response has no choices")
	}

	content := strings.TrimSpace(llmResp.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var decision Decision
	if err := json.Unmarshal([]byte(content), &decision); err != nil {
		return Decision{}, fmt.Errorf("invalid decision json: %w", err)
	}

	if decision.TargetStageID == "" {
		decision.TargetStageID = s.defaultStage
	}
	decision.DealTitle = strings.TrimSpace(decision.DealTitle)
	decision.Reason = strings.TrimSpace(decision.Reason)

	return decision, nil
}

func (d Decision) Describe() string {
	dealID := "0"
	if d.DealID > 0 {
		dealID = strconv.Itoa(d.DealID)
	}
	return fmt.Sprintf("move=%t, deal_id=%s, deal_title=%q, stage=%s, reason=%s", d.ShouldMove, dealID, d.DealTitle, d.TargetStageID, d.Reason)
}

func isOpenRouterURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, "openrouter.ai")
}
