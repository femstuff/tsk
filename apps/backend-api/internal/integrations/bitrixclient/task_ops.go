package bitrixclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// BitrixTaskPerson — пользователь Bitrix24 в контексте задачи.
type BitrixTaskPerson struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	WorkPosition string `json:"workPosition,omitempty"`
}

// BitrixTaskDetail — полная карточка задачи для мобильного приложения.
type BitrixTaskDetail struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Priority    string `json:"priority,omitempty"`
	Mark        string `json:"mark,omitempty"`

	Creator     BitrixTaskPerson   `json:"creator"`
	Responsible BitrixTaskPerson   `json:"responsible"`
	Accomplices []BitrixTaskPerson `json:"accomplices,omitempty"`
	Auditors    []BitrixTaskPerson `json:"auditors,omitempty"`

	CreatedDate   string `json:"createdDate,omitempty"`
	ChangedDate   string `json:"changedDate,omitempty"`
	Deadline      string `json:"deadline,omitempty"`
	ClosedDate    string `json:"closedDate,omitempty"`
	DateStart     string `json:"dateStart,omitempty"`
	StartDatePlan string `json:"startDatePlan,omitempty"`
	EndDatePlan   string `json:"endDatePlan,omitempty"`

	GroupID       string            `json:"groupId,omitempty"`
	StageID       string            `json:"stageId,omitempty"`
	ParentID      string            `json:"parentId,omitempty"`
	CommentsCount string            `json:"commentsCount,omitempty"`
	TimeEstimate  string            `json:"timeEstimate,omitempty"`
	DurationFact  string            `json:"durationFact,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
	CrmLinks      []string          `json:"crmLinks,omitempty"`
	Favorite      bool              `json:"favorite,omitempty"`
	AvailableActions map[string]bool `json:"availableActions,omitempty"`
}

var taskGetSelectFields = []string{
	"ID", "TITLE", "DESCRIPTION", "STATUS", "PRIORITY", "MARK",
	"CREATED_BY", "CREATED_DATE", "CHANGED_DATE",
	"RESPONSIBLE_ID", "ACCOMPLICES", "AUDITORS",
	"DEADLINE", "CLOSED_DATE", "DATE_START", "START_DATE_PLAN", "END_DATE_PLAN",
	"GROUP_ID", "STAGE_ID", "PARENT_ID",
	"COMMENTS_COUNT", "TIME_ESTIMATE", "DURATION_FACT",
	"TAGS", "UF_CRM_TASK", "FAVORITE",
}

func personFromMap(m map[string]any) BitrixTaskPerson {
	return BitrixTaskPerson{
		ID:           fieldFromRow(m, "id", "ID"),
		Name:         strings.TrimSpace(fieldFromRow(m, "name", "NAME", "formattedName", "FORMATTED_NAME")),
		WorkPosition: strings.TrimSpace(fieldFromRow(m, "workPosition", "WORK_POSITION")),
	}
}

func personField(row map[string]any, nestedKeys []string, idKeys ...string) BitrixTaskPerson {
	for _, nk := range nestedKeys {
		for k, v := range row {
			if !strings.EqualFold(k, nk) {
				continue
			}
			if m, ok := v.(map[string]any); ok {
				p := personFromMap(m)
				if p.ID != "" || p.Name != "" {
					return p
				}
			}
		}
	}
	id := fieldFromRow(row, idKeys...)
	if id != "" {
		return BitrixTaskPerson{ID: id}
	}
	return BitrixTaskPerson{}
}

func personsFromRow(row map[string]any, keys ...string) []BitrixTaskPerson {
	for _, want := range keys {
		for k, v := range row {
			if strings.EqualFold(k, want) {
				return parsePersonList(v)
			}
		}
	}
	return nil
}

func parsePersonList(v any) []BitrixTaskPerson {
	switch t := v.(type) {
	case []any:
		out := make([]BitrixTaskPerson, 0, len(t))
		for _, item := range t {
			switch entry := item.(type) {
			case map[string]any:
				p := personFromMap(entry)
				if p.ID != "" || p.Name != "" {
					out = append(out, p)
				}
			default:
				if item != nil {
					s := strings.TrimSpace(fmt.Sprint(item))
					if s != "" {
						out = append(out, BitrixTaskPerson{ID: s})
					}
				}
			}
		}
		return out
	case map[string]any:
		p := personFromMap(t)
		if p.ID != "" || p.Name != "" {
			return []BitrixTaskPerson{p}
		}
	}
	return nil
}

func stringsFromAny(v any) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if item == nil {
				continue
			}
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil
		}
		return []string{s}
	default:
		if v == nil {
			return nil
		}
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" {
			return nil
		}
		return []string{s}
	}
}

func boolFromRow(row map[string]any, keys ...string) bool {
	v := fieldFromRow(row, keys...)
	switch strings.ToUpper(v) {
	case "Y", "YES", "TRUE", "1":
		return true
	default:
		return false
	}
}

func actionsFromRow(row map[string]any) map[string]bool {
	for k, v := range row {
		if !strings.EqualFold(k, "action") && !strings.EqualFold(k, "actions") {
			continue
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		out := make(map[string]bool, len(m))
		for ak, av := range m {
			switch flag := av.(type) {
			case bool:
				out[ak] = flag
			case string:
				out[ak] = strings.EqualFold(flag, "true") || flag == "1" || strings.EqualFold(flag, "Y")
			default:
				out[ak] = fmt.Sprint(av) == "1" || strings.EqualFold(fmt.Sprint(av), "true")
			}
		}
		return out
	}
	return nil
}

func mapRowToTaskDetail(row map[string]any) BitrixTaskDetail {
	if nested, ok := row["task"].(map[string]any); ok {
		row = nested
	}

	creator := personField(row, []string{"creator", "createdByUser"}, "createdBy", "CREATED_BY")
	responsible := personField(row, []string{"responsible", "responsibleUser"}, "responsibleId", "RESPONSIBLE_ID")

	return BitrixTaskDetail{
		ID:          fieldFromRow(row, "id", "ID"),
		Title:       strings.TrimSpace(fieldFromRow(row, "title", "TITLE")),
		Description: strings.TrimSpace(fieldFromRow(row, "description", "DESCRIPTION")),
		Status:      fieldFromRow(row, "status", "STATUS"),
		Priority:    fieldFromRow(row, "priority", "PRIORITY"),
		Mark:        fieldFromRow(row, "mark", "MARK"),

		Creator:     creator,
		Responsible: responsible,
		Accomplices: personsFromRow(row, "accomplices", "ACCOMPLICES"),
		Auditors:    personsFromRow(row, "auditors", "AUDITORS"),

		CreatedDate:   fieldFromRow(row, "createdDate", "CREATED_DATE"),
		ChangedDate:   fieldFromRow(row, "changedDate", "CHANGED_DATE"),
		Deadline:      fieldFromRow(row, "deadline", "DEADLINE"),
		ClosedDate:    fieldFromRow(row, "closedDate", "CLOSED_DATE", "closed_date"),
		DateStart:     fieldFromRow(row, "dateStart", "DATE_START"),
		StartDatePlan: fieldFromRow(row, "startDatePlan", "START_DATE_PLAN"),
		EndDatePlan:   fieldFromRow(row, "endDatePlan", "END_DATE_PLAN"),

		GroupID:       fieldFromRow(row, "groupId", "GROUP_ID"),
		StageID:       fieldFromRow(row, "stageId", "STAGE_ID"),
		ParentID:      fieldFromRow(row, "parentId", "PARENT_ID"),
		CommentsCount: fieldFromRow(row, "commentsCount", "COMMENTS_COUNT"),
		TimeEstimate:  fieldFromRow(row, "timeEstimate", "TIME_ESTIMATE"),
		DurationFact:  fieldFromRow(row, "durationFact", "DURATION_FACT"),
		Tags:          stringsFromAny(rowValueCI(row, "tags", "TAGS")),
		CrmLinks:      stringsFromAny(rowValueCI(row, "ufCrmTask", "UF_CRM_TASK")),
		Favorite:      boolFromRow(row, "favorite", "FAVORITE"),
		AvailableActions: actionsFromRow(row),
	}
}

func rowValueCI(row map[string]any, keys ...string) any {
	for _, want := range keys {
		for k, v := range row {
			if strings.EqualFold(k, want) {
				return v
			}
		}
	}
	return nil
}

func parseTaskGetResult(raw json.RawMessage) (BitrixTaskDetail, error) {
	raw = bytesTrim(raw)
	if len(raw) == 0 {
		return BitrixTaskDetail{}, fmt.Errorf("empty tasks.task.get result")
	}
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		return BitrixTaskDetail{}, fmt.Errorf("tasks.task.get JSON: %w", err)
	}
	detail := mapRowToTaskDetail(row)
	if detail.ID == "" {
		return BitrixTaskDetail{}, fmt.Errorf("task not found")
	}
	return detail, nil
}

type bitrixRESTPoster interface {
	postForm(ctx context.Context, method string, form url.Values) (json.RawMessage, error)
}

type webhookREST struct {
	webhookURL string
	httpClient *http.Client
}

func (w webhookREST) postForm(ctx context.Context, method string, form url.Values) (json.RawMessage, error) {
	if strings.TrimSpace(w.webhookURL) == "" {
		return nil, fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	endpoint := strings.TrimSuffix(w.webhookURL, "/") + "/" + strings.TrimSuffix(method, ".json") + ".json"
	return postBitrixForm(ctx, w.httpClient, endpoint, form)
}

type tokenRESTPoster struct {
	token *TokenREST
}

func (t tokenRESTPoster) postForm(ctx context.Context, method string, form url.Values) (json.RawMessage, error) {
	if !t.token.Configured() {
		return nil, fmt.Errorf("bitrix oauth client is not configured")
	}
	return postBitrixForm(ctx, t.token.httpClient, t.token.methodURL(method), form)
}

func postBitrixForm(ctx context.Context, httpClient *http.Client, endpoint string, form url.Values) (json.RawMessage, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
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
	if len(bytesTrim(body)) > 0 {
		_ = json.Unmarshal(body, &envelope)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bitrix HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if envelope.Error != nil || envelope.ErrorDescription != "" {
		return nil, fmt.Errorf("bitrix error: %v, %s", envelope.Error, strings.TrimSpace(envelope.ErrorDescription))
	}
	return envelope.Result, nil
}

func getTaskVia(ctx context.Context, poster bitrixRESTPoster, taskID string) (BitrixTaskDetail, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return BitrixTaskDetail{}, fmt.Errorf("task id is required")
	}
	form := url.Values{}
	form.Set("taskId", taskID)
	for _, field := range taskGetSelectFields {
		form.Add("select[]", field)
	}

	raw, err := poster.postForm(ctx, "tasks.task.get", form)
	if err != nil {
		return BitrixTaskDetail{}, fmt.Errorf("tasks.task.get: %w", err)
	}
	return parseTaskGetResult(raw)
}

func updateTaskStatusVia(ctx context.Context, poster bitrixRESTPoster, taskID string, status int) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}
	if status <= 0 {
		return fmt.Errorf("status is required")
	}
	form := url.Values{}
	form.Set("taskId", taskID)
	form.Set("fields[STATUS]", strconv.Itoa(status))

	_, err := poster.postForm(ctx, "tasks.task.update", form)
	if err != nil {
		return fmt.Errorf("tasks.task.update: %w", err)
	}
	return nil
}

// GetTask — карточка задачи через входящий вебхук.
func (c *Client) GetTask(ctx context.Context, taskID string) (BitrixTaskDetail, error) {
	return getTaskVia(ctx, webhookREST{webhookURL: c.webhookURL, httpClient: c.httpClient}, taskID)
}

// UpdateTaskStatus — смена статуса задачи через вебхук.
func (c *Client) UpdateTaskStatus(ctx context.Context, taskID string, status int) error {
	return updateTaskStatusVia(ctx, webhookREST{webhookURL: c.webhookURL, httpClient: c.httpClient}, taskID, status)
}

// GetTask — карточка задачи от имени OAuth-пользователя.
func (t *TokenREST) GetTask(ctx context.Context, taskID string) (BitrixTaskDetail, error) {
	return getTaskVia(ctx, tokenRESTPoster{token: t}, taskID)
}

// UpdateTaskStatus — смена статуса задачи от имени OAuth-пользователя.
func (t *TokenREST) UpdateTaskStatus(ctx context.Context, taskID string, status int) error {
	return updateTaskStatusVia(ctx, tokenRESTPoster{token: t}, taskID, status)
}
