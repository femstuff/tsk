package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Decision struct {
	ShouldMove  bool        `json:"should_move"`
	Direction   string      `json:"direction"`
	DealID      int         `json:"deal_id"`
	DealTitle   string      `json:"deal_title"`
	TargetStage interface{} `json:"target_stage"` // может быть string или int
	Reason      string      `json:"reason"`
}

func (d *Decision) GetTargetStage() string {
	if d.TargetStage == nil {
		return ""
	}
	switch v := d.TargetStage.(type) {
	case string:
		return v
	case float64:
		return strconv.Itoa(int(v))
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprintf("%v", v)
	}
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

type DecisionService struct {
	apiURL       string
	apiKey       string
	model        string
	defaultStage string
	client       *http.Client
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

	systemPrompt := `Ты помощник по продажам. Проанализируй транскрипт и верни только JSON.

Доступные стадии: NEW, PREPARATION, EXECUTING, PREPAYMENT_INVOICE, FINAL_INVOICE, SUCCESS, FAIL

Правила:
1. should_move = true если сказано: да, согласен, перемести, на стадию, в этап
2. direction = "next" - следующий этап
   direction = "prev" - предыдущий этап
   direction = "to_stage" - на конкретную стадию
3. target_stage = строкой (например "EXECUTING"), НЕ数字
4. deal_id = число (ID сделки), по умолчанию 0
5. deal_title = строка (название сделки)

Пример правильного ответа: {"should_move": true, "direction": "to_stage", "target_stage": "EXECUTING", "deal_id": 0, "deal_title": "тест"}

Верни ТОЛЬКО JSON.`

	userPrompt := fmt.Sprintf("Транскрипт: %s", transcribedText)

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

	req, err := http.NewRequest("POST", s.apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return Decision{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return Decision{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Decision{}, err
	}

	if resp.StatusCode != 200 {
		return Decision{}, fmt.Errorf("LLM API error %d: %s", resp.StatusCode, string(respBody))
	}

	var llmResp llmResponse
	if err := json.Unmarshal(respBody, &llmResp); err != nil {
		return Decision{}, err
	}
	if len(llmResp.Choices) == 0 {
		return Decision{}, fmt.Errorf("no choices in response")
	}

	content := llmResp.Choices[0].Message.Content
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var decision Decision
	if err := json.Unmarshal([]byte(content), &decision); err != nil {
		return Decision{}, fmt.Errorf("json parse error: %v, content: %s", err, content)
	}

	targetStage := decision.GetTargetStage()

	if targetStage != "" && targetStage != "0" {
		stageMap := map[string]string{
			"0": "NEW",
			"1": "PREPARATION",
			"2": "EXECUTING",
			"3": "PREPAYMENT_INVOICE",
			"4": "FINAL_INVOICE",
			"5": "SUCCESS",
			"6": "FAIL",
		}
		if mapped, ok := stageMap[targetStage]; ok {
			targetStage = mapped
		}
	}

	decision.TargetStage = targetStage

	if decision.Direction == "" {
		if strings.Contains(transcribedText, "предыдущ") || strings.Contains(transcribedText, "назад") {
			decision.Direction = "prev"
		} else if strings.Contains(transcribedText, "стади") || strings.Contains(transcribedText, "этап") {
			decision.Direction = "to_stage"
		} else {
			decision.Direction = "next"
		}
	}

	log.Printf("LLM решение: should_move=%v, direction=%s, deal_id=%d, deal_title=%s, target_stage=%s",
		decision.ShouldMove, decision.Direction, decision.DealID, decision.DealTitle, decision.GetTargetStage())

	return decision, nil
}
