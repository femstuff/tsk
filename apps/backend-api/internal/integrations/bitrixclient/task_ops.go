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

	GroupID          string            `json:"groupId,omitempty"`
	GroupTitle       string            `json:"groupTitle,omitempty"`
	StageID          string            `json:"stageId,omitempty"`
	StageLabel       string            `json:"stageLabel,omitempty"`
	ParentID         string            `json:"parentId,omitempty"`
	ParentTitle      string            `json:"parentTitle,omitempty"`
	CommentsCount    string            `json:"commentsCount,omitempty"`
	TimeEstimate     string            `json:"timeEstimate,omitempty"`
	DurationFact     string            `json:"durationFact,omitempty"`
	DurationPlan     string            `json:"durationPlan,omitempty"`
	DurationType     string            `json:"durationType,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	CrmLinks         []string          `json:"crmLinks,omitempty"`
	Favorite         bool              `json:"favorite,omitempty"`
	AllowTimeTracking bool             `json:"allowTimeTracking,omitempty"`
	TaskControl      bool              `json:"taskControl,omitempty"`
	Multitask        bool              `json:"multitask,omitempty"`
	ForumTopicID     string            `json:"forumTopicId,omitempty"`
	ChatID           string            `json:"chatId,omitempty"`
	AvailableActions map[string]bool   `json:"availableActions,omitempty"`
	Checklist        []BitrixTaskChecklistItem `json:"checklist,omitempty"`
	Files            []BitrixTaskFile          `json:"files,omitempty"`
	Comments         []BitrixTaskComment       `json:"comments,omitempty"`
}

var taskGetSelectFields = []string{
	"ID", "TITLE", "DESCRIPTION", "STATUS", "PRIORITY", "MARK",
	"CREATED_BY", "CREATED_DATE", "CHANGED_DATE",
	"RESPONSIBLE_ID", "ACCOMPLICES", "AUDITORS",
	"DEADLINE", "CLOSED_DATE", "DATE_START", "START_DATE_PLAN", "END_DATE_PLAN",
	"GROUP_ID", "STAGE_ID", "PARENT_ID",
	"COMMENTS_COUNT", "TIME_ESTIMATE", "DURATION_FACT", "DURATION_PLAN", "DURATION_TYPE",
	"TAGS", "UF_CRM_TASK", "UF_TASK_WEBDAV_FILES", "FAVORITE",
	"ALLOW_TIME_TRACKING", "TASK_CONTROL", "MULTITASK", "FORUM_TOPIC_ID", "CHAT", "CHAT_ID", "CHECKLIST",
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

		GroupID:           fieldFromRow(row, "groupId", "GROUP_ID"),
		StageID:           fieldFromRow(row, "stageId", "STAGE_ID"),
		ParentID:          fieldFromRow(row, "parentId", "PARENT_ID"),
		CommentsCount:     fieldFromRow(row, "commentsCount", "COMMENTS_COUNT"),
		TimeEstimate:      fieldFromRow(row, "timeEstimate", "TIME_ESTIMATE"),
		DurationFact:      fieldFromRow(row, "durationFact", "DURATION_FACT"),
		DurationPlan:      fieldFromRow(row, "durationPlan", "DURATION_PLAN"),
		DurationType:      fieldFromRow(row, "durationType", "DURATION_TYPE"),
		Tags:              stringsFromAny(rowValueCI(row, "tags", "TAGS")),
		CrmLinks:          stringsFromAny(rowValueCI(row, "ufCrmTask", "UF_CRM_TASK")),
		Favorite:          boolFromRow(row, "favorite", "FAVORITE"),
		AllowTimeTracking: boolFromRow(row, "allowTimeTracking", "ALLOW_TIME_TRACKING"),
		TaskControl:       boolFromRow(row, "taskControl", "TASK_CONTROL"),
		Multitask:         boolFromRow(row, "multitask", "MULTITASK"),
		ForumTopicID:      fieldFromRow(row, "forumTopicId", "FORUM_TOPIC_ID"),
		ChatID:            chatIDFromTaskRow(row),
		AvailableActions:  actionsFromRow(row),
		Checklist:         checklistFromRow(row),
		Files:             filesFromTaskRow(row),
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

func chatIDFromTaskRow(row map[string]any) string {
	for _, key := range []string{"chatId", "CHAT_ID", "chat_id"} {
		if v := strings.TrimSpace(fieldFromRow(row, key)); v != "" && v != "<nil>" {
			return v
		}
	}
	raw := rowValueCI(row, "chat", "CHAT")
	switch t := raw.(type) {
	case map[string]any:
		return fieldFromRow(t, "id", "ID")
	case float64:
		return strconv.Itoa(int(t))
	case int:
		return strconv.Itoa(t)
	default:
		s := strings.TrimSpace(fmt.Sprint(raw))
		if s == "" || s == "<nil>" {
			return ""
		}
		return s
	}
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

type bitrixRESTJSONPoster interface {
	postJSON(ctx context.Context, method string, payload any) (json.RawMessage, error)
}

type bitrixRESTV3Poster interface {
	postFormV3(ctx context.Context, method string, form url.Values) (json.RawMessage, error)
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

func (w webhookREST) postFormV3(ctx context.Context, method string, form url.Values) (json.RawMessage, error) {
	if strings.TrimSpace(w.webhookURL) == "" {
		return nil, fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	base := RestAPIV3Base(w.webhookURL)
	endpoint := strings.TrimSuffix(base, "/") + "/" + strings.TrimSuffix(method, ".json") + ".json"
	return postBitrixForm(ctx, w.httpClient, endpoint, form)
}

func (w webhookREST) postJSON(ctx context.Context, method string, payload any) (json.RawMessage, error) {
	if strings.TrimSpace(w.webhookURL) == "" {
		return nil, fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	endpoint := strings.TrimSuffix(w.webhookURL, "/") + "/" + strings.TrimSuffix(method, ".json") + ".json"
	return postBitrixJSON(ctx, w.httpClient, endpoint, payload)
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

func (t tokenRESTPoster) postFormV3(ctx context.Context, method string, form url.Values) (json.RawMessage, error) {
	if !t.token.Configured() {
		return nil, fmt.Errorf("bitrix oauth client is not configured")
	}
	base := RestAPIV3Base(t.token.restBase)
	method = strings.TrimPrefix(strings.TrimSpace(method), "/")
	method = strings.TrimSuffix(method, ".json")
	endpoint := strings.TrimSuffix(base, "/") + "/" + method + ".json?auth=" + url.QueryEscape(t.token.accessToken)
	return postBitrixForm(ctx, t.token.httpClient, endpoint, form)
}

func (t tokenRESTPoster) postJSON(ctx context.Context, method string, payload any) (json.RawMessage, error) {
	if !t.token.Configured() {
		return nil, fmt.Errorf("bitrix oauth client is not configured")
	}
	return postBitrixJSON(ctx, t.token.httpClient, t.token.methodURL(method), payload)
}

func postBitrixJSON(ctx context.Context, httpClient *http.Client, endpoint string, payload any) (json.RawMessage, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Result           json.RawMessage `json:"result"`
		Error            any             `json:"error"`
		ErrorDescription string          `json:"error_description"`
	}
	if len(bytesTrim(respBody)) > 0 {
		_ = json.Unmarshal(respBody, &envelope)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bitrix HTTP %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	if envelope.Error != nil || envelope.ErrorDescription != "" {
		return nil, fmt.Errorf("bitrix error: %v, %s", envelope.Error, strings.TrimSpace(envelope.ErrorDescription))
	}
	return envelope.Result, nil
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
	poster := webhookREST{webhookURL: c.webhookURL, httpClient: c.httpClient}
	detail, err := getTaskVia(ctx, poster, taskID)
	if err != nil {
		return detail, err
	}
	return enrichTaskDetail(ctx, poster, taskID, detail, portalHostFromWebhookURL(c.webhookURL)), nil
}

// UpdateTaskStatus — смена статуса задачи через вебхук.
func (c *Client) UpdateTaskStatus(ctx context.Context, taskID string, status int) error {
	return updateTaskStatusVia(ctx, webhookREST{webhookURL: c.webhookURL, httpClient: c.httpClient}, taskID, status)
}

// GetTask — карточка задачи от имени OAuth-пользователя.
func (t *TokenREST) GetTask(ctx context.Context, taskID string) (BitrixTaskDetail, error) {
	poster := tokenRESTPoster{token: t}
	detail, err := getTaskVia(ctx, poster, taskID)
	if err != nil {
		return detail, err
	}
	host := ""
	if parsed, err := url.Parse(t.restBase); err == nil {
		host = parsed.Host
	}
	return enrichTaskDetail(ctx, poster, taskID, detail, host), nil
}

// UpdateTaskStatus — смена статуса задачи от имени OAuth-пользователя.
func (t *TokenREST) UpdateTaskStatus(ctx context.Context, taskID string, status int) error {
	return updateTaskStatusVia(ctx, tokenRESTPoster{token: t}, taskID, status)
}
