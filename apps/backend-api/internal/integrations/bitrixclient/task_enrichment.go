package bitrixclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BitrixTaskFile — вложение задачи или комментария.
type BitrixTaskFile struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Size        string `json:"size,omitempty"`
	DownloadURL string `json:"downloadUrl,omitempty"`
	ViewURL     string `json:"viewUrl,omitempty"`
}

// BitrixTaskComment — сообщение в чате задачи Bitrix24.
type BitrixTaskComment struct {
	ID         string           `json:"id"`
	AuthorID   string           `json:"authorId,omitempty"`
	AuthorName string           `json:"authorName,omitempty"`
	PostDate   string           `json:"postDate,omitempty"`
	Message    string           `json:"message"`
	Files      []BitrixTaskFile `json:"files,omitempty"`
}

// BitrixTaskChecklistItem — пункт чек-листа задачи.
type BitrixTaskChecklistItem struct {
	ID         string `json:"id,omitempty"`
	Title      string `json:"title"`
	IsComplete bool   `json:"isComplete"`
	SortIndex  int    `json:"sortIndex,omitempty"`
}

func enrichTaskDetail(ctx context.Context, poster bitrixRESTPoster, taskID string, detail BitrixTaskDetail, portalHost string) BitrixTaskDetail {
	legacyComments, legacyErr := listTaskCommentsVia(ctx, poster, taskID)
	var chatComments []BitrixTaskComment
	if strings.TrimSpace(detail.ChatID) != "" {
		if items, err := listTaskChatMessagesVia(ctx, poster, detail.ChatID); err == nil {
			chatComments = items
		}
	}
	detail.Comments = mergeTaskComments(legacyComments, chatComments)
	if len(detail.Comments) == 0 && legacyErr != nil && len(chatComments) == 0 {
		detail.Comments = nil
	}
	if len(detail.Comments) > 0 {
		detail.Files = mergeTaskFiles(detail.Files, filesFromComments(detail.Comments))
	}
	detail = enrichTaskPeople(ctx, poster, detail)
	detail = enrichTaskReferences(ctx, poster, detail)
	detail.Files = finalizeTaskFileURLs(detail.Files, portalHost)
	detail.Comments = finalizeCommentFileURLs(detail.Comments, portalHost)
	return detail
}

func listTaskCommentsVia(ctx context.Context, poster bitrixRESTPoster, taskID string) ([]BitrixTaskComment, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}

	form := url.Values{}
	form.Set("TASKID", taskID)
	form.Set("ORDER[POST_DATE]", "asc")

	raw, err := poster.postForm(ctx, "task.commentitem.getlist", form)
	if err != nil {
		return nil, err
	}
	return parseTaskCommentsResult(raw)
}

func parseTaskCommentsResult(raw json.RawMessage) ([]BitrixTaskComment, error) {
	raw = bytesTrim(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("task.commentitem.getlist JSON: %w", err)
	}

	out := make([]BitrixTaskComment, 0, len(rows))
	for _, row := range rows {
		id := fieldFromRow(row, "ID", "id")
		if id == "" {
			continue
		}
		comment := BitrixTaskComment{
			ID:         id,
			AuthorID:   fieldFromRow(row, "AUTHOR_ID", "authorId"),
			AuthorName: strings.TrimSpace(fieldFromRow(row, "AUTHOR_NAME", "authorName")),
			PostDate:   fieldFromRow(row, "POST_DATE", "postDate"),
			Message:    strings.TrimSpace(fieldFromRow(row, "POST_MESSAGE", "postMessage")),
			Files:      parseAttachedObjects(rowValueCI(row, "ATTACHED_OBJECTS", "attachedObjects")),
		}
		out = append(out, comment)
	}
	return out, nil
}

func parseAttachedObjects(v any) []BitrixTaskFile {
	switch t := v.(type) {
	case map[string]any:
		return filesFromObjectMap(t)
	case []any:
		out := make([]BitrixTaskFile, 0, len(t))
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				if file := mapRowToTaskFile(m); file.Name != "" || file.ID != "" {
					out = append(out, file)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func filesFromObjectMap(m map[string]any) []BitrixTaskFile {
	if len(m) == 0 {
		return nil
	}
	out := make([]BitrixTaskFile, 0, len(m))
	for key, value := range m {
		row, ok := value.(map[string]any)
		if !ok {
			continue
		}
		file := mapRowToTaskFile(row)
		if file.ID == "" {
			file.ID = key
		}
		if file.Name != "" || file.ID != "" {
			out = append(out, file)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func mapRowToTaskFile(row map[string]any) BitrixTaskFile {
	return BitrixTaskFile{
		ID:          fieldFromRow(row, "ATTACHMENT_ID", "attachmentId", "FILE_ID", "fileId", "ID", "id"),
		Name:        strings.TrimSpace(fieldFromRow(row, "NAME", "name")),
		Size:        fieldFromRow(row, "SIZE", "size"),
		DownloadURL: fieldFromRow(row, "DOWNLOAD_URL", "downloadUrl"),
		ViewURL:     fieldFromRow(row, "VIEW_URL", "viewUrl"),
	}
}

func filesFromTaskRow(row map[string]any) []BitrixTaskFile {
	raw := rowValueCI(row, "ufTaskWebdavFiles", "UF_TASK_WEBDAV_FILES", "files", "FILES")
	switch t := raw.(type) {
	case []any:
		out := make([]BitrixTaskFile, 0, len(t))
		for _, item := range t {
			switch entry := item.(type) {
			case map[string]any:
				if file := mapRowToTaskFile(entry); file.Name != "" || file.ID != "" {
					out = append(out, file)
				}
			default:
				ref := strings.TrimSpace(fmt.Sprint(item))
				if ref != "" {
					out = append(out, BitrixTaskFile{ID: ref, Name: ref})
				}
			}
		}
		return out
	case map[string]any:
		return filesFromObjectMap(t)
	default:
		return nil
	}
}

func filesFromComments(comments []BitrixTaskComment) []BitrixTaskFile {
	var out []BitrixTaskFile
	for _, comment := range comments {
		out = append(out, comment.Files...)
	}
	return out
}

func mergeTaskFiles(base, extra []BitrixTaskFile) []BitrixTaskFile {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]BitrixTaskFile, 0, len(base)+len(extra))
	appendUnique := func(file BitrixTaskFile) {
		key := strings.TrimSpace(file.ID)
		if key == "" {
			key = strings.TrimSpace(file.Name) + "|" + strings.TrimSpace(file.DownloadURL)
		}
		if key == "" || key == "|" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, file)
	}
	for _, file := range base {
		appendUnique(file)
	}
	for _, file := range extra {
		appendUnique(file)
	}
	return out
}

func checklistFromRow(row map[string]any) []BitrixTaskChecklistItem {
	raw := rowValueCI(row, "checklist", "CHECKLIST")
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]BitrixTaskChecklistItem, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title := strings.TrimSpace(fieldFromRow(m, "TITLE", "title"))
		if title == "" {
			continue
		}
		sortIndex, _ := strconv.Atoi(strings.TrimSpace(fieldFromRow(m, "SORT_INDEX", "sortIndex")))
		out = append(out, BitrixTaskChecklistItem{
			ID:         fieldFromRow(m, "ID", "id"),
			Title:      title,
			IsComplete: boolFromRow(m, "IS_COMPLETE", "isComplete"),
			SortIndex:  sortIndex,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortIndex != out[j].SortIndex {
			return out[i].SortIndex < out[j].SortIndex
		}
		return out[i].Title < out[j].Title
	})
	return out
}

func portalHostFromWebhookURL(webhookURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(webhookURL))
	if err != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Host
}

func absolutizeBitrixURL(rawURL, portalHost string) string {
	rawURL = strings.TrimSpace(rawURL)
	portalHost = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(portalHost, "https://"), "http://"))
	if rawURL == "" || portalHost == "" {
		return rawURL
	}
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	if strings.HasPrefix(rawURL, "/") {
		return "https://" + portalHost + rawURL
	}
	return rawURL
}

func absolutizeTaskFileURLs(files []BitrixTaskFile, portalHost string) []BitrixTaskFile {
	return finalizeTaskFileURLs(files, portalHost)
}

func finalizeTaskFileURLs(files []BitrixTaskFile, portalHost string) []BitrixTaskFile {
	if len(files) == 0 {
		return files
	}
	out := make([]BitrixTaskFile, len(files))
	for i, file := range files {
		out[i] = finalizeTaskFileURL(file, portalHost)
	}
	return out
}

func finalizeTaskFileURL(file BitrixTaskFile, portalHost string) BitrixTaskFile {
	file.DownloadURL = absolutizeBitrixURL(file.DownloadURL, portalHost)
	file.ViewURL = absolutizeBitrixURL(file.ViewURL, portalHost)
	if file.DownloadURL == "" && file.ViewURL == "" {
		if built := buildDiskFileURL(portalHost, file.ID); built != "" {
			file.DownloadURL = built
			file.ViewURL = built
		}
	}
	if file.ViewURL == "" {
		file.ViewURL = file.DownloadURL
	}
	if file.DownloadURL == "" {
		file.DownloadURL = file.ViewURL
	}
	return file
}

func buildDiskFileURL(portalHost, fileID string) string {
	portalHost = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(portalHost, "https://"), "http://"))
	fileID = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(fileID), "n"))
	if portalHost == "" || fileID == "" {
		return ""
	}
	return "https://" + portalHost + "/bitrix/tools/disk/uf.php?attachedId=" + url.QueryEscape(fileID)
}

func finalizeCommentFileURLs(comments []BitrixTaskComment, portalHost string) []BitrixTaskComment {
	if portalHost == "" {
		return comments
	}
	for i := range comments {
		comments[i].Files = finalizeTaskFileURLs(comments[i].Files, portalHost)
	}
	return comments
}

func absolutizeCommentFileURLs(comments []BitrixTaskComment, portalHost string) []BitrixTaskComment {
	return finalizeCommentFileURLs(comments, portalHost)
}

func mergeTaskComments(parts ...[]BitrixTaskComment) []BitrixTaskComment {
	seen := make(map[string]struct{})
	out := make([]BitrixTaskComment, 0)
	for _, batch := range parts {
		for _, comment := range batch {
			id := strings.TrimSpace(comment.ID)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, comment)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return commentTimeMs(out[i].PostDate) < commentTimeMs(out[j].PostDate)
	})
	return out
}

func commentTimeMs(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"02.01.2006 15:04:05",
		"02.01.2006 15:04",
	}
	for _, layout := range layouts {
		if tt, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return tt.UnixMilli()
		}
	}
	if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return id
	}
	return 0
}

func normalizeChatDialogID(chatID string) string {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return ""
	}
	lower := strings.ToLower(chatID)
	if strings.HasPrefix(lower, "chat") {
		return chatID
	}
	return "chat" + chatID
}

func listTaskChatMessagesVia(ctx context.Context, poster bitrixRESTPoster, chatID string) ([]BitrixTaskComment, error) {
	dialogID := normalizeChatDialogID(chatID)
	if dialogID == "" {
		return nil, fmt.Errorf("chat id is required")
	}
	form := url.Values{}
	form.Set("DIALOG_ID", dialogID)
	form.Set("LIMIT", "100")

	raw, err := poster.postForm(ctx, "im.dialog.messages.get", form)
	if err != nil {
		return nil, err
	}
	return parseIMDialogMessages(raw)
}

func parseIMDialogMessages(raw json.RawMessage) ([]BitrixTaskComment, error) {
	raw = bytesTrim(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var rows []map[string]any
	var envelope struct {
		Messages []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && len(envelope.Messages) > 0 {
		rows = envelope.Messages
	} else if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("im.dialog.messages.get JSON: %w", err)
	}

	out := make([]BitrixTaskComment, 0, len(rows))
	for _, row := range rows {
		id := fieldFromRow(row, "id", "ID")
		if id == "" {
			continue
		}
		message := strings.TrimSpace(fieldFromRow(row, "text", "TEXT", "message", "MESSAGE"))
		if message == "" {
			message = strings.TrimSpace(fieldFromRow(row, "message", "MESSAGE"))
		}
		comment := BitrixTaskComment{
			ID:         id,
			AuthorID:   fieldFromRow(row, "author_id", "AUTHOR_ID", "authorId"),
			AuthorName: strings.TrimSpace(fieldFromRow(row, "author_name", "AUTHOR_NAME", "authorName")),
			PostDate:   fieldFromRow(row, "date", "DATE"),
			Message:    message,
			Files:      parseIMMessageFiles(row),
		}
		if comment.Message == "" && len(comment.Files) == 0 {
			continue
		}
		out = append(out, comment)
	}
	return out, nil
}

func parseIMMessageFiles(row map[string]any) []BitrixTaskFile {
	params, ok := rowValueCI(row, "params", "PARAMS").(map[string]any)
	if !ok {
		return nil
	}
	rawFiles := rowValueCI(params, "FILE_ID", "fileId", "FILES", "files", "ATTACH")
	switch t := rawFiles.(type) {
	case []any:
		out := make([]BitrixTaskFile, 0, len(t))
		for _, item := range t {
			switch entry := item.(type) {
			case map[string]any:
				out = append(out, mapRowToTaskFile(entry))
			default:
				id := strings.TrimSpace(fmt.Sprint(entry))
				if id != "" && id != "<nil>" {
					out = append(out, BitrixTaskFile{ID: id, Name: "Файл " + id})
				}
			}
		}
		return out
	case map[string]any:
		return filesFromObjectMap(t)
	default:
		return nil
	}
}

func parseBitrixTaskIDInt(taskID string) (int, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return 0, fmt.Errorf("task id is required")
	}
	if id, err := strconv.Atoi(taskID); err == nil && id > 0 {
		return id, nil
	}
	if f, err := strconv.ParseFloat(taskID, 64); err == nil && f > 0 {
		return int(f), nil
	}
	return 0, fmt.Errorf("invalid task id: %q", taskID)
}

func addTaskCommentVia(ctx context.Context, poster bitrixRESTPoster, taskID, message, authorID string) error {
	taskIDInt, err := parseBitrixTaskIDInt(taskID)
	if err != nil {
		return err
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("comment message is required")
	}

	fields := map[string]any{
		"POST_MESSAGE": message,
	}
	if authorID = strings.TrimSpace(authorID); authorID != "" {
		if aid, err := strconv.Atoi(authorID); err == nil && aid > 0 {
			fields["AUTHOR_ID"] = aid
		}
	}

	if jsonPoster, ok := poster.(bitrixRESTJSONPoster); ok {
		payload := map[string]any{
			"TASKID": taskIDInt,
			"FIELDS": fields,
		}
		if _, err := jsonPoster.postJSON(ctx, "task.commentitem.add", payload); err == nil {
			return nil
		} else if !strings.Contains(err.Error(), "WRONG_ARGUMENTS") {
			return fmt.Errorf("task.commentitem.add: %w", err)
		}
	}

	form := url.Values{}
	form.Set("TASKID", strconv.Itoa(taskIDInt))
	form.Set("FIELDS[POST_MESSAGE]", message)
	if aid, ok := fields["AUTHOR_ID"]; ok {
		form.Set("FIELDS[AUTHOR_ID]", strconv.Itoa(aid.(int)))
	}
	_, err = poster.postForm(ctx, "task.commentitem.add", form)
	if err != nil {
		return fmt.Errorf("task.commentitem.add: %w", err)
	}
	return nil
}

func sendTaskChatMessageV3Via(ctx context.Context, poster bitrixRESTPoster, taskID, message string) error {
	taskID = strings.TrimSpace(taskID)
	message = strings.TrimSpace(message)
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}
	if message == "" {
		return fmt.Errorf("comment message is required")
	}
	v3Poster, ok := poster.(bitrixRESTV3Poster)
	if !ok {
		return fmt.Errorf("rest v3 not supported")
	}
	form := url.Values{}
	form.Set("fields[taskId]", taskID)
	form.Set("fields[text]", message)
	_, err := v3Poster.postFormV3(ctx, "tasks.task.chat.message.send", form)
	if err != nil {
		return fmt.Errorf("tasks.task.chat.message.send: %w", err)
	}
	return nil
}

func sendTaskChatMessageVia(ctx context.Context, poster bitrixRESTPoster, chatID, message string) error {
	dialogID := normalizeChatDialogID(chatID)
	message = strings.TrimSpace(message)
	if dialogID == "" {
		return fmt.Errorf("chat id is required")
	}
	if message == "" {
		return fmt.Errorf("comment message is required")
	}

	payload := map[string]any{
		"DIALOG_ID": dialogID,
		"MESSAGE":   message,
	}
	if jsonPoster, ok := poster.(bitrixRESTJSONPoster); ok {
		if _, err := jsonPoster.postJSON(ctx, "im.message.add", payload); err == nil {
			return nil
		}
	}

	form := url.Values{}
	form.Set("DIALOG_ID", dialogID)
	form.Set("MESSAGE", message)
	_, err := poster.postForm(ctx, "im.message.add", form)
	if err != nil {
		return fmt.Errorf("im.message.add: %w", err)
	}
	return nil
}

func isBitrixMethodNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") ||
		strings.Contains(msg, "methodnotfound") ||
		strings.Contains(msg, "не найден")
}

func addTaskCommentOrChatMessageVia(ctx context.Context, poster bitrixRESTPoster, taskID, chatID, message, authorID string) error {
	var failures []string

	if strings.TrimSpace(chatID) != "" {
		if err := sendTaskChatMessageVia(ctx, poster, chatID, message); err == nil {
			return nil
		} else {
			failures = append(failures, err.Error())
		}
	}

	if err := addTaskCommentVia(ctx, poster, taskID, message, authorID); err == nil {
		return nil
	} else {
		failures = append(failures, err.Error())
	}

	if err := sendTaskChatMessageV3Via(ctx, poster, taskID, message); err == nil {
		return nil
	} else if !isBitrixMethodNotFound(err) {
		failures = append(failures, err.Error())
	}

	if len(failures) == 0 {
		return fmt.Errorf("не удалось отправить сообщение: чат задачи недоступен")
	}
	return fmt.Errorf("не удалось отправить сообщение в чат задачи")
}

func (c *Client) AddTaskComment(ctx context.Context, taskID, chatID, message, authorID string) error {
	if !c.WebhookConfigured() {
		return fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	return addTaskCommentOrChatMessageVia(ctx, webhookREST{webhookURL: c.webhookURL, httpClient: c.httpClient}, taskID, chatID, message, authorID)
}

func (t *TokenREST) AddTaskComment(ctx context.Context, taskID, chatID, message, authorID string) error {
	return addTaskCommentOrChatMessageVia(ctx, tokenRESTPoster{token: t}, taskID, chatID, message, authorID)
}
