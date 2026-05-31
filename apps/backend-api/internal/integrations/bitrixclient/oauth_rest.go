package bitrixclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// TokenREST — вызовы Bitrix REST от имени OAuth access_token пользователя.
type TokenREST struct {
	restBase    string
	accessToken string
	httpClient  *http.Client
}

func NewTokenREST(portalDomain, restEndpoint, accessToken string, httpClient *http.Client) *TokenREST {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &TokenREST{
		restBase:    RestBaseURL(portalDomain, restEndpoint),
		accessToken: strings.TrimSpace(accessToken),
		httpClient:  httpClient,
	}
}

func (t *TokenREST) Configured() bool {
	return t != nil && t.restBase != "" && t.accessToken != ""
}

func (t *TokenREST) methodURL(method string) string {
	base := t.restBase
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	method = strings.TrimPrefix(strings.TrimSpace(method), "/")
	method = strings.TrimSuffix(method, ".json")
	return base + method + "?auth=" + url.QueryEscape(t.accessToken)
}

func (t *TokenREST) ListTasksForResponsible(ctx context.Context, responsibleID int, limit int) ([]BitrixTaskBrief, error) {
	return t.ListTasksForUser(ctx, responsibleID, limit)
}

// ListTasksForUser — задачи, где пользователь ответственный или соисполнитель.
func (t *TokenREST) ListTasksForUser(ctx context.Context, userID int, limit int) ([]BitrixTaskBrief, error) {
	if !t.Configured() {
		return nil, fmt.Errorf("bitrix oauth client is not configured")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("bitrix user id is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	byID := make(map[string]BitrixTaskBrief)
	appendUnique := func(items []BitrixTaskBrief) {
		for _, item := range items {
			if item.ID == "" {
				continue
			}
			byID[item.ID] = item
		}
	}

	responsible, err := t.listTasksWithFilter(ctx, "RESPONSIBLE_ID", userID, limit)
	if err != nil {
		return nil, err
	}
	appendUnique(responsible)

	accomplice, err := t.listTasksWithFilter(ctx, "ACCOMPLICE", userID, limit)
	if err == nil {
		appendUnique(accomplice)
	}

	created, err := t.listTasksWithFilter(ctx, "CREATED_BY", userID, limit)
	if err == nil {
		appendUnique(created)
	}

	out := make([]BitrixTaskBrief, 0, len(byID))
	for _, item := range byID {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID > out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (t *TokenREST) listTasksWithFilter(ctx context.Context, filterField string, userID int, limit int) ([]BitrixTaskBrief, error) {
	form := url.Values{}
	form.Set("filter["+filterField+"]", strconv.Itoa(userID))
	form.Add("select[]", "ID")
	form.Add("select[]", "TITLE")
	form.Add("select[]", "STATUS")
	form.Add("select[]", "DEADLINE")
	form.Add("select[]", "CLOSED_DATE")
	form.Set("order[ID]", "desc")
	form.Set("start", "0")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.methodURL("tasks.task.list"), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
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
		return nil, fmt.Errorf("bitrix tasks.task.list HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if envelope.Error != nil || envelope.ErrorDescription != "" {
		return nil, fmt.Errorf("bitrix error: %v, %s", envelope.Error, strings.TrimSpace(envelope.ErrorDescription))
	}

	tasks, err := parseTasksListResult(envelope.Result)
	if err != nil {
		return nil, err
	}
	if len(tasks) > limit {
		tasks = tasks[:limit]
	}
	return tasks, nil
}

func bytesTrim(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}
