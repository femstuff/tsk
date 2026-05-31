package bitrixclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
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
	if comments, err := listTaskCommentsVia(ctx, poster, taskID); err == nil {
		detail.Comments = comments
		detail.Files = mergeTaskFiles(detail.Files, filesFromComments(comments))
	}
	detail = enrichTaskPeople(ctx, poster, detail)
	detail = enrichTaskReferences(ctx, poster, detail)
	detail.Files = absolutizeTaskFileURLs(detail.Files, portalHost)
	detail.Comments = absolutizeCommentFileURLs(detail.Comments, portalHost)
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
	if portalHost == "" || len(files) == 0 {
		return files
	}
	for i := range files {
		files[i].DownloadURL = absolutizeBitrixURL(files[i].DownloadURL, portalHost)
		files[i].ViewURL = absolutizeBitrixURL(files[i].ViewURL, portalHost)
	}
	return files
}

func absolutizeCommentFileURLs(comments []BitrixTaskComment, portalHost string) []BitrixTaskComment {
	if portalHost == "" {
		return comments
	}
	for i := range comments {
		comments[i].Files = absolutizeTaskFileURLs(comments[i].Files, portalHost)
	}
	return comments
}
