package bitrixclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type MoveDirection string

const (
	MoveDirectionAuto MoveDirection = "auto"
	MoveDirectionNext MoveDirection = "next"
	MoveDirectionPrev MoveDirection = "prev"
)

type Client struct {
	webhookURL string
	httpClient *http.Client
}

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

func New(webhookURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		webhookURL: strings.TrimRight(strings.TrimSpace(webhookURL), "/"),
		httpClient: httpClient,
	}
}

func (c *Client) WebhookConfigured() bool {
	return c != nil && strings.TrimSpace(c.webhookURL) != ""
}

var reWebhookRestPath = regexp.MustCompile(`(?i)/rest/(\d+)/[^/]+/?$`)

// WebhookOwnerUserID — числовой ID пользователя Bitrix24 из входящего вебхука …/rest/{userId}/{token}/.
func (c *Client) WebhookOwnerUserID() (int, error) {
	if !c.WebhookConfigured() {
		return 0, fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	u := c.webhookURL
	if m := reWebhookRestPath.FindStringSubmatch(u); len(m) > 1 {
		id, err := strconv.Atoi(m[1])
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("invalid user id in webhook URL")
		}
		return id, nil
	}
	return 0, fmt.Errorf("cannot parse user id from BITRIX_WEBHOOK_URL (expected .../rest/{userId}/{token}/)")
}

type bitrixDeal struct {
	ID         int
	StageID    string
	CategoryID int
}

// MoveDealToStage перемещает сделку по направлению или на стадию по имени/коду (как в bot-service).
func (c *Client) MoveDealToStage(ctx context.Context, dealID int, stageID string, direction MoveDirection) (int, string, error) {
	if !c.WebhookConfigured() {
		return 0, "", fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	if dealID <= 0 {
		return 0, "", fmt.Errorf("deal id is required")
	}

	deal, err := c.getDealByID(ctx, dealID)
	if err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "not found") {
			return 0, "", err
		}
		resolvedDealID, err2 := c.FindDealIDByTitle(ctx, strconv.Itoa(dealID))
		if err2 != nil {
			return 0, "", fmt.Errorf("deal not found by id; resolve by title failed: %v", err2)
		}
		deal, err = c.getDealByID(ctx, resolvedDealID)
		if err != nil {
			return 0, "", err
		}
		dealID = resolvedDealID
	}

	var targetStageID string

	switch direction {
	case MoveDirectionPrev:
		prevStageID, err := c.findPrevStageID(ctx, deal.StageID, deal.CategoryID)
		if err != nil {
			return dealID, deal.StageID, err
		}
		targetStageID = prevStageID

	case MoveDirectionNext:
		nextStageID, err := c.findNextStageID(ctx, deal.StageID, deal.CategoryID)
		if err != nil {
			return dealID, deal.StageID, err
		}
		targetStageID = nextStageID

	default:
		if stageID != "" {
			targetStageID = stageID
			_, _ = c.findStageIDByName(ctx, stageID, deal.CategoryID)
		} else {
			nextStageID, err := c.findNextStageID(ctx, deal.StageID, deal.CategoryID)
			if err != nil {
				return dealID, deal.StageID, err
			}
			targetStageID = nextStageID
		}
	}

	if targetStageID == "" {
		return dealID, deal.StageID, fmt.Errorf("could not resolve target stage")
	}

	if targetStageID == deal.StageID {
		return dealID, deal.StageID, nil
	}

	if err := c.updateDealStageByID(ctx, dealID, targetStageID); err != nil {
		return dealID, targetStageID, err
	}

	return dealID, targetStageID, nil
}

// AddTask создаёт задачу в Bitrix24 (tasks.task.add).
func (c *Client) AddTask(ctx context.Context, title, description string) (string, error) {
	if !c.WebhookConfigured() {
		return "", fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("task title is required")
	}

	form := url.Values{}
	form.Set("fields[TITLE]", title)
	form.Set("fields[DESCRIPTION]", description)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL+"/tasks.task.add.json", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("bitrix tasks.task.add HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var envelope struct {
		Result           json.RawMessage `json:"result"`
		Error            any             `json:"error"`
		ErrorDescription string          `json:"error_description"`
	}
	if len(bytes.TrimSpace(body)) > 0 {
		_ = json.Unmarshal(body, &envelope)
	}
	if envelope.Error != nil || envelope.ErrorDescription != "" {
		return "", fmt.Errorf("bitrix error: %v, %s", envelope.Error, strings.TrimSpace(envelope.ErrorDescription))
	}

	taskID, err := parseTaskAddResultID(envelope.Result)
	if err != nil {
		return "", fmt.Errorf("%w: raw=%s", err, strings.TrimSpace(string(body)))
	}
	return taskID, nil
}

// parseTaskAddResultID — Bitrix может вернуть result как число или { "task": { "id": ... } }.
func parseTaskAddResultID(raw json.RawMessage) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", fmt.Errorf("empty result")
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil && n != "" {
		return strings.TrimSpace(string(n)), nil
	}
	var wrapped struct {
		Task struct {
			ID interface{} `json:"id"`
		} `json:"task"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Task.ID != nil {
		return strings.TrimSpace(fmt.Sprint(wrapped.Task.ID)), nil
	}
	var flat struct {
		ID interface{} `json:"id"`
	}
	if err := json.Unmarshal(raw, &flat); err == nil && flat.ID != nil {
		return strings.TrimSpace(fmt.Sprint(flat.ID)), nil
	}
	return "", fmt.Errorf("unexpected tasks.task.add result shape")
}

// BitrixTaskBrief — краткая запись из tasks.task.list (вебхук = контекст пользователя, создавшего вебхук).
type BitrixTaskBrief struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Status        string `json:"status"`
	Deadline      string `json:"deadline,omitempty"`
	ClosedDate    string `json:"closedDate,omitempty"`
	CreatedDate   string `json:"createdDate,omitempty"`
	ChangedDate   string `json:"changedDate,omitempty"`
	ResponsibleID string `json:"responsibleId,omitempty"`
}

// ListTasks возвращает последние задачи (без фильтра по ответственному — устаревший режим).
func (c *Client) ListTasks(ctx context.Context, limit int) ([]BitrixTaskBrief, error) {
	return c.listTasksPOST(ctx, 0, limit)
}

// ListTasksForResponsible — задачи, где пользователь responsibleId — ответственный (POST tasks.task.list).
func (c *Client) ListTasksForResponsible(ctx context.Context, responsibleID int, limit int) ([]BitrixTaskBrief, error) {
	if responsibleID <= 0 {
		return nil, fmt.Errorf("responsible user id is required")
	}
	return c.listTasksPOST(ctx, responsibleID, limit)
}

func (c *Client) listTasksPOST(ctx context.Context, responsibleID int, limit int) ([]BitrixTaskBrief, error) {
	if !c.WebhookConfigured() {
		return nil, fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	form := url.Values{}
	if responsibleID > 0 {
		form.Set("filter[RESPONSIBLE_ID]", strconv.Itoa(responsibleID))
	}
	form.Add("select[]", "ID")
	form.Add("select[]", "TITLE")
	form.Add("select[]", "STATUS")
	form.Add("select[]", "DEADLINE")
	form.Add("select[]", "CLOSED_DATE")
	form.Add("select[]", "CREATED_DATE")
	form.Add("select[]", "CHANGED_DATE")
	form.Add("select[]", "RESPONSIBLE_ID")
	form.Set("order[CHANGED_DATE]", "desc")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL+"/tasks.task.list.json", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Result           json.RawMessage `json:"result"`
		Error            any             `json:"error"`
		ErrorDescription string          `json:"error_description"`
	}
	if len(bytes.TrimSpace(body)) > 0 {
		_ = json.Unmarshal(body, &envelope)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bitrix tasks.task.list HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if envelope.Error != nil || envelope.ErrorDescription != "" {
		return nil, fmt.Errorf("bitrix error: %v, %s", envelope.Error, strings.TrimSpace(envelope.ErrorDescription))
	}

	tasks, err := parseTasksListResult(envelope.Result)
	if err != nil {
		return nil, err
	}
	sortBitrixTasksByRecency(tasks)
	if len(tasks) > limit {
		tasks = tasks[:limit]
	}
	return tasks, nil
}

func parseTasksListResult(raw json.RawMessage) ([]BitrixTaskBrief, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty tasks.task.list result")
	}
	var wrapped struct {
		Tasks []map[string]any `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Tasks != nil {
		return mapSliceToTaskBriefs(wrapped.Tasks), nil
	}
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err == nil {
		return mapSliceToTaskBriefs(arr), nil
	}
	return nil, fmt.Errorf("unexpected tasks.task.list result shape: %s", strings.TrimSpace(string(raw)))
}

func mapSliceToTaskBriefs(rows []map[string]any) []BitrixTaskBrief {
	out := make([]BitrixTaskBrief, 0, len(rows))
	for _, row := range rows {
		if nested, ok := row["task"].(map[string]any); ok {
			row = nested
		}
		id := fieldFromRow(row, "id", "ID")
		title := fieldFromRow(row, "title", "TITLE")
		status := fieldFromRow(row, "status", "STATUS")
		deadline := fieldFromRow(row, "deadline", "DEADLINE")
		closed := fieldFromRow(row, "closedDate", "CLOSED_DATE", "closed_date")
		if id == "" && title == "" {
			continue
		}
		out = append(out, BitrixTaskBrief{
			ID: id, Title: strings.TrimSpace(title), Status: status,
			Deadline: deadline, ClosedDate: closed,
			CreatedDate:   fieldFromRow(row, "createdDate", "CREATED_DATE"),
			ChangedDate:   fieldFromRow(row, "changedDate", "CHANGED_DATE"),
			ResponsibleID: fieldFromRow(row, "responsibleId", "RESPONSIBLE_ID"),
		})
	}
	return out
}

func bitrixTaskRecencyKey(task BitrixTaskBrief) string {
	if changed := strings.TrimSpace(task.ChangedDate); changed != "" {
		return changed
	}
	if created := strings.TrimSpace(task.CreatedDate); created != "" {
		return created
	}
	return task.ID
}

func sortBitrixTasksByRecency(tasks []BitrixTaskBrief) {
	sort.Slice(tasks, func(i, j int) bool {
		return bitrixTaskRecencyKey(tasks[i]) > bitrixTaskRecencyKey(tasks[j])
	})
}

func fieldFromRow(row map[string]any, keys ...string) string {
	for _, want := range keys {
		for k, v := range row {
			if strings.EqualFold(k, want) && v != nil {
				return strings.TrimSpace(fmt.Sprint(v))
			}
		}
	}
	return ""
}

func (c *Client) updateDealStageByID(ctx context.Context, dealID int, stageID string) error {
	form := url.Values{}
	form.Set("id", strconv.Itoa(dealID))
	form.Set("fields[STAGE_ID]", stageID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL+"/crm.deal.update.json", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
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

func (c *Client) getDealByID(ctx context.Context, dealID int) (bitrixDeal, error) {
	u := c.webhookURL + "/crm.deal.get.json?id=" + strconv.Itoa(dealID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return bitrixDeal{}, err
	}
	resp, err := c.httpClient.Do(req)
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

func (c *Client) findNextStageID(ctx context.Context, currentStageID string, categoryID int) (string, error) {
	stages, err := c.listStages(ctx, categoryID)
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

func (c *Client) findPrevStageID(ctx context.Context, currentStageID string, categoryID int) (string, error) {
	stages, err := c.listStages(ctx, categoryID)
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

func (c *Client) listStages(ctx context.Context, categoryID int) ([]bitrixStage, error) {
	u := c.webhookURL + "/crm.status.list.json?filter[ENTITY_ID]=DEAL_STAGE"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
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

// FindDealIDByTitle ищет сделку по точному названию.
func (c *Client) FindDealIDByTitle(ctx context.Context, title string) (int, error) {
	u := c.webhookURL + "/crm.deal.list.json?filter[TITLE]=" + url.QueryEscape(title) + "&select[]=ID"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var parsed bitrixDealListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}

	if len(parsed.Result) == 0 {
		return 0, fmt.Errorf("deal with title %q not found", title)
	}

	id, err := strconv.Atoi(parsed.Result[0].ID)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// FindDealIDByTitleFuzzy ищет сделку, у которой TITLE содержит needle (без учёта регистра), по последним сделкам.
func (c *Client) FindDealIDByTitleFuzzy(ctx context.Context, needle string) (id int, matchedTitle string, err error) {
	if !c.WebhookConfigured() {
		return 0, "", fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	needle = strings.TrimSpace(strings.ToLower(needle))
	if needle == "" {
		return 0, "", fmt.Errorf("empty deal title search string")
	}

	for start := 0; start < 400; start += 50 {
		u := fmt.Sprintf("%s/crm.deal.list.json?select[]=ID&select[]=TITLE&order[ID]=DESC&start=%d", c.webhookURL, start)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return 0, "", err
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, "", err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return 0, "", readErr
		}

		var parsed bitrixDealListResponse
		if jsonErr := json.Unmarshal(body, &parsed); jsonErr != nil {
			return 0, "", fmt.Errorf("deal.list parse: %w", jsonErr)
		}
		if resp.StatusCode >= 300 {
			return 0, "", fmt.Errorf("deal.list HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		if parsed.Error != nil || parsed.ErrorDescription != "" {
			return 0, "", fmt.Errorf("bitrix deal.list error: %v, %s", parsed.Error, strings.TrimSpace(parsed.ErrorDescription))
		}
		if len(parsed.Result) == 0 {
			break
		}
		for _, row := range parsed.Result {
			t := strings.TrimSpace(strings.ToLower(row.Title))
			if t == "" {
				continue
			}
			if strings.Contains(t, needle) {
				id, convErr := strconv.Atoi(row.ID)
				if convErr != nil {
					continue
				}
				return id, row.Title, nil
			}
		}
	}

	return 0, "", fmt.Errorf("no deal with title containing %q (searched recent deals)", needle)
}

func (c *Client) findStageIDByName(ctx context.Context, stageName string, categoryID int) (string, error) {
	stages, err := c.listStages(ctx, categoryID)
	if err != nil {
		return "", err
	}

	stageNameLower := strings.ToLower(stageName)

	mapping := map[string]string{
		"новый": "NEW", "новая": "NEW", "start": "NEW",
		"в работе": "EXECUTING", "работаем": "EXECUTING", "обработка": "EXECUTING",
		"выполнение": "EXECUTING", "executing": "EXECUTING", "in work": "EXECUTING", "in_work": "EXECUTING",
		"предоплата": "PREPAYMENT_INVOICE", "счет": "PREPAYMENT_INVOICE",
		"выставлен счет": "PREPAYMENT_INVOICE", "предоплатный счет": "PREPAYMENT_INVOICE",
		"финальный счет": "FINAL_INVOICE", "окончательный счет": "FINAL_INVOICE",
		"финал": "FINAL_INVOICE", "final invoice": "FINAL_INVOICE", "final": "FINAL_INVOICE",
		"подготовка": "PREPARATION", "prepare": "PREPARATION",
		"успех": "SUCCESS", "успешно": "SUCCESS", "выиграли": "SUCCESS", "win": "SUCCESS",
		"проигрыш": "FAIL", "проиграли": "FAIL", "потеряли": "FAIL", "отказ": "FAIL", "fail": "FAIL",
	}

	if mapped, ok := mapping[stageNameLower]; ok {
		for _, st := range stages {
			if strings.HasSuffix(st.ID, mapped) {
				return st.ID, nil
			}
		}
	}

	for _, st := range stages {
		if strings.Contains(strings.ToLower(st.ID), stageNameLower) {
			return st.ID, nil
		}
	}

	if strings.Contains(stageNameLower, "финальн") || strings.Contains(stageNameLower, "final") {
		for _, st := range stages {
			if strings.Contains(strings.ToUpper(st.ID), "FINAL") {
				return st.ID, nil
			}
		}
	}

	if strings.Contains(stageNameLower, "предоплат") || strings.Contains(stageNameLower, "prepayment") {
		for _, st := range stages {
			if strings.Contains(strings.ToUpper(st.ID), "PREPAYMENT") {
				return st.ID, nil
			}
		}
	}

	return "", fmt.Errorf("стадия '%s' не найдена в категории %d", stageName, categoryID)
}

// ResolveStageID превращает подсказку (EXECUTING / «в работе») в реальный STATUS_ID для воронки сделки.
func (c *Client) ResolveStageID(ctx context.Context, dealID int, stageHint string) (string, error) {
	deal, err := c.getDealByID(ctx, dealID)
	if err != nil {
		return "", err
	}
	return c.findStageIDByName(ctx, stageHint, deal.CategoryID)
}
