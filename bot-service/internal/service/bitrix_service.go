package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type BitrixService struct {
	webhookURL string
	client     *http.Client
}

type MoveDirection string

const (
	MoveDirectionAuto MoveDirection = "auto"
	MoveDirectionNext MoveDirection = "next"
	MoveDirectionPrev MoveDirection = "prev"
)

type bitrixResponse struct {
	Result           any    `json:"result"`
	Error            any    `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type bitrixDealListResponse struct {
	Result []struct {
		ID    string `json:"ID"`
		Title string `json:"TITLE"`
	} `json:"result"`
	Error            any    `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type bitrixDealGetResponse struct {
	Result struct {
		ID         string `json:"ID"`
		StageID    string `json:"STAGE_ID"`
		CategoryID string `json:"CATEGORY_ID"`
	} `json:"result"`
	Error            any    `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type bitrixStatusListResponse struct {
	Result []struct {
		StatusID   string `json:"STATUS_ID"`
		Sort       string `json:"SORT"`
		CategoryID string `json:"CATEGORY_ID"`
	} `json:"result"`
	Error            any    `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func NewBitrixService(webhookURL string, client *http.Client) *BitrixService {
	if client == nil {
		client = http.DefaultClient
	}

	return &BitrixService{
		webhookURL: strings.TrimRight(webhookURL, "/"),
		client:     client,
	}
}

func (s *BitrixService) MoveDealToStage(dealID int, stageID string, direction MoveDirection) (int, string, error) {
	if strings.TrimSpace(s.webhookURL) == "" {
		return 0, "", fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	if dealID <= 0 {
		return 0, "", fmt.Errorf("deal id is required")
	}
	if strings.TrimSpace(stageID) == "" {
		return 0, "", fmt.Errorf("stage id is required")
	}

	resolvedDealID := dealID
	deal, err := s.getDealByID(resolvedDealID)
	if err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "not found") {
			return 0, "", err
		}
		resolvedDealID, err = s.FindDealIDByTitle(strconv.Itoa(dealID))
		if err != nil {
			return 0, "", fmt.Errorf("deal not found by id; resolve by title failed: %v", err)
		}
		deal, err = s.getDealByID(resolvedDealID)
		if err != nil {
			return 0, "", err
		}
	}

	targetStageID := stageID
	switch direction {
	case MoveDirectionPrev:
		prevStageID, prevErr := s.findPrevStageID(deal.StageID, deal.CategoryID)
		if prevErr != nil {
			return resolvedDealID, deal.StageID, prevErr
		}
		targetStageID = prevStageID
	case MoveDirectionNext:
		nextStageID, nextErr := s.findNextStageID(deal.StageID, deal.CategoryID)
		if nextErr != nil {
			return resolvedDealID, deal.StageID, nextErr
		}
		targetStageID = nextStageID
	default:
		if deal.StageID == stageID {
			nextStageID, nextErr := s.findNextStageID(deal.StageID, deal.CategoryID)
			if nextErr != nil {
				return resolvedDealID, deal.StageID, nextErr
			}
			targetStageID = nextStageID
		}
	}

	if err := s.updateDealStageByID(resolvedDealID, targetStageID); err != nil {
		return resolvedDealID, targetStageID, err
	}

	return resolvedDealID, targetStageID, nil
}

func (s *BitrixService) updateDealStageByID(dealID int, stageID string) error {
	form := url.Values{}
	form.Set("id", strconv.Itoa(dealID))
	form.Set("fields[STAGE_ID]", stageID)

	req, err := http.NewRequest(http.MethodPost, s.webhookURL+"/crm.deal.update.json", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var parsed bitrixResponse
	if len(bytes.TrimSpace(body)) > 0 {
		_ = json.Unmarshal(body, &parsed)
	}

	if resp.StatusCode >= 300 {
		if parsed.Error != nil || parsed.ErrorDescription != "" {
			return fmt.Errorf("bitrix status: %s, error: %v, description: %s", resp.Status, parsed.Error, strings.TrimSpace(parsed.ErrorDescription))
		}
		return fmt.Errorf("bitrix status: %s, body: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	if parsed.Error != nil || parsed.ErrorDescription != "" {
		return fmt.Errorf("bitrix error: %v, description: %s", parsed.Error, strings.TrimSpace(parsed.ErrorDescription))
	}

	return nil
}

type bitrixDeal struct {
	ID         int
	StageID    string
	CategoryID int
}

func (s *BitrixService) getDealByID(dealID int) (bitrixDeal, error) {
	u := s.webhookURL + "/crm.deal.get.json?id=" + strconv.Itoa(dealID)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return bitrixDeal{}, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return bitrixDeal{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return bitrixDeal{}, err
	}

	var parsed bitrixDealGetResponse
	if len(bytes.TrimSpace(body)) > 0 {
		_ = json.Unmarshal(body, &parsed)
	}

	if resp.StatusCode >= 300 {
		return bitrixDeal{}, fmt.Errorf("bitrix deal.get status: %s, body: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if parsed.Error != nil || parsed.ErrorDescription != "" {
		return bitrixDeal{}, fmt.Errorf("bitrix deal.get error: %v, description: %s", parsed.Error, strings.TrimSpace(parsed.ErrorDescription))
	}

	id, _ := strconv.Atoi(parsed.Result.ID)
	categoryID, _ := strconv.Atoi(parsed.Result.CategoryID)
	return bitrixDeal{
		ID:         id,
		StageID:    parsed.Result.StageID,
		CategoryID: categoryID,
	}, nil
}

func (s *BitrixService) findNextStageID(currentStageID string, categoryID int) (string, error) {
	stages, err := s.listStages(categoryID)
	if err != nil {
		return "", err
	}

	currentIdx := -1
	for i, st := range stages {
		if st.ID == currentStageID {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 {
		return "", fmt.Errorf("current stage %s is not found in category %d", currentStageID, categoryID)
	}
	if currentIdx == len(stages)-1 {
		return "", fmt.Errorf("deal is already at the final stage %s", currentStageID)
	}

	return stages[currentIdx+1].ID, nil
}

func (s *BitrixService) findPrevStageID(currentStageID string, categoryID int) (string, error) {
	stages, err := s.listStages(categoryID)
	if err != nil {
		return "", err
	}

	currentIdx := -1
	for i, st := range stages {
		if st.ID == currentStageID {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 {
		return "", fmt.Errorf("current stage %s is not found in category %d", currentStageID, categoryID)
	}
	if currentIdx == 0 {
		return "", fmt.Errorf("deal is already at the first stage %s", currentStageID)
	}

	return stages[currentIdx-1].ID, nil
}

type bitrixStage struct {
	ID   string
	Sort int
}

func (s *BitrixService) listStages(categoryID int) ([]bitrixStage, error) {
	u := s.webhookURL + "/crm.status.list.json?filter[ENTITY_ID]=DEAL_STAGE"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed bitrixStatusListResponse
	if len(bytes.TrimSpace(body)) > 0 {
		_ = json.Unmarshal(body, &parsed)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bitrix status.list status: %s, body: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if parsed.Error != nil || parsed.ErrorDescription != "" {
		return nil, fmt.Errorf("bitrix status.list error: %v, description: %s", parsed.Error, strings.TrimSpace(parsed.ErrorDescription))
	}

	var stages []bitrixStage
	for _, item := range parsed.Result {
		cat, _ := strconv.Atoi(item.CategoryID)
		if cat != categoryID {
			continue
		}
		sv, _ := strconv.Atoi(item.Sort)
		stages = append(stages, bitrixStage{ID: item.StatusID, Sort: sv})
	}
	if len(stages) == 0 {
		return nil, fmt.Errorf("no stages found for category %d", categoryID)
	}

	sort.Slice(stages, func(i, j int) bool { return stages[i].Sort < stages[j].Sort })
	return stages, nil
}

func (s *BitrixService) FindDealIDByTitle(title string) (int, error) {
	u := s.webhookURL + "/crm.deal.list.json?filter[TITLE]=" + url.QueryEscape(title) + "&select[]=ID"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode >= 300 {
		return 0, fmt.Errorf("bitrix deal.list status: %s, body: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var parsed bitrixDealListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}
	if parsed.Error != nil || parsed.ErrorDescription != "" {
		return 0, fmt.Errorf("bitrix deal.list error: %v, description: %s", parsed.Error, strings.TrimSpace(parsed.ErrorDescription))
	}
	if len(parsed.Result) == 0 {
		return 0, fmt.Errorf("deal with title %q not found", title)
	}
	if len(parsed.Result) > 1 {
		return 0, fmt.Errorf("found %d deals with title %q", len(parsed.Result), title)
	}

	id, err := strconv.Atoi(parsed.Result[0].ID)
	if err != nil {
		return 0, fmt.Errorf("invalid deal id from bitrix: %s", parsed.Result[0].ID)
	}
	return id, nil
}
