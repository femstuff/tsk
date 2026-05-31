package bitrixclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type BitrixNotification struct {
	ID     string `json:"id"`
	Title  string `json:"title,omitempty"`
	Text   string `json:"text"`
	Date   string `json:"date"`
	Read   bool   `json:"read"`
	Module string `json:"module,omitempty"`
	Tag    string `json:"tag,omitempty"`
}

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

func (c *Client) ListNotifications(ctx context.Context, limit int) ([]BitrixNotification, error) {
	if !c.WebhookConfigured() {
		return nil, fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	return listNotificationsVia(ctx, webhookREST{webhookURL: c.webhookURL, httpClient: c.httpClient}, limit)
}

func (t *TokenREST) ListNotifications(ctx context.Context, limit int) ([]BitrixNotification, error) {
	return listNotificationsVia(ctx, tokenRESTPoster{token: t}, limit)
}

func listNotificationsVia(ctx context.Context, poster bitrixRESTPoster, limit int) ([]BitrixNotification, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	out := make([]BitrixNotification, 0, limit)
	lastID := ""
	lastType := ""

	for len(out) < limit {
		pageLimit := limit - len(out)
		if pageLimit > 50 {
			pageLimit = 50
		}

		form := urlValuesForNotify(lastID, lastType, pageLimit)
		raw, err := poster.postForm(ctx, "im.notify.get", form)
		if err != nil {
			if len(out) > 0 {
				return out, nil
			}
			return nil, fmt.Errorf("im.notify.get: %w", err)
		}

		page, nextID, nextType, err := parseNotifyPage(raw)
		if err != nil {
			if len(out) > 0 {
				return out, nil
			}
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		out = append(out, page...)
		if nextID == "" || (nextID == lastID && nextType == lastType) {
			break
		}
		lastID = nextID
		lastType = nextType
	}

	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func urlValuesForNotify(lastID, lastType string, limit int) url.Values {
	form := url.Values{}
	form.Set("LIMIT", strconv.Itoa(limit))
	if strings.TrimSpace(lastID) != "" {
		form.Set("LAST_ID", lastID)
	}
	if strings.TrimSpace(lastType) != "" {
		form.Set("LAST_TYPE", lastType)
	}
	return form
}

func parseNotifyPage(raw json.RawMessage) ([]BitrixNotification, string, string, error) {
	raw = bytesTrim(raw)
	if len(raw) == 0 {
		return nil, "", "", nil
	}

	var envelope struct {
		Notifications []map[string]any `json:"notifications"`
		LastID        any              `json:"LAST_ID"`
		LastType      any              `json:"LAST_TYPE"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		var items []map[string]any
		if err2 := json.Unmarshal(raw, &items); err2 == nil {
			return mapNotifyRows(items), "", "", nil
		}
		return nil, "", "", fmt.Errorf("im.notify.get JSON: %w", err)
	}

	nextID := strings.TrimSpace(fmt.Sprint(envelope.LastID))
	nextType := strings.TrimSpace(fmt.Sprint(envelope.LastType))
	if nextID == "<nil>" {
		nextID = ""
	}
	if nextType == "<nil>" {
		nextType = ""
	}

	page := mapNotifyRows(envelope.Notifications)
	if len(page) > 0 && nextID == "" {
		last := page[len(page)-1]
		nextID = last.ID
	}
	return page, nextID, nextType, nil
}

func mapNotifyRows(rows []map[string]any) []BitrixNotification {
	out := make([]BitrixNotification, 0, len(rows))
	for _, row := range rows {
		id := fieldFromRow(row, "ID", "id")
		if id == "" {
			continue
		}
		text := stripHTML(fieldFromRow(row, "TEXT", "text", "MESSAGE", "message"))
		title := stripHTML(fieldFromRow(row, "TITLE", "title", "SUBJECT", "subject"))
		if title == "" {
			title = stripHTML(fieldFromRow(row, "TAG", "tag"))
		}
		if text == "" && title != "" {
			text = title
			title = ""
		}
		readRaw := strings.ToUpper(fieldFromRow(row, "READ", "read"))
		out = append(out, BitrixNotification{
			ID:     id,
			Title:  title,
			Text:   text,
			Date:   fieldFromRow(row, "DATE", "date"),
			Read:   readRaw == "Y" || readRaw == "TRUE" || readRaw == "1",
			Module: fieldFromRow(row, "MODULE", "module"),
			Tag:    fieldFromRow(row, "TAG", "tag"),
		})
	}
	return out
}

func stripHTML(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	return strings.Join(strings.Fields(s), " ")
}
