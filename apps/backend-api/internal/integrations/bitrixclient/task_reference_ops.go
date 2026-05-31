package bitrixclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func enrichTaskReferences(ctx context.Context, poster bitrixRESTPoster, detail BitrixTaskDetail) BitrixTaskDetail {
	if parentID := strings.TrimSpace(detail.ParentID); parentID != "" && parentID != "0" {
		detail.ParentTitle = lookupTaskTitle(ctx, poster, parentID)
	}
	if groupID := strings.TrimSpace(detail.GroupID); groupID != "" && groupID != "0" {
		detail.GroupTitle = lookupSonetGroupTitle(ctx, poster, groupID)
		if stageID := strings.TrimSpace(detail.StageID); stageID != "" && stageID != "0" {
			if label := lookupTaskStageLabel(ctx, poster, groupID, stageID); label != "" {
				detail.StageLabel = label
			}
		}
	}
	if len(detail.CrmLinks) > 0 {
		labels := make([]string, 0, len(detail.CrmLinks))
		for _, link := range detail.CrmLinks {
			labels = append(labels, resolveCrmTaskLinkLabel(ctx, poster, link))
		}
		detail.CrmLinks = labels
	}
	return detail
}

func lookupTaskTitle(ctx context.Context, poster bitrixRESTPoster, taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" || poster == nil {
		return ""
	}
	form := url.Values{}
	form.Set("taskId", taskID)
	form.Add("select[]", "ID")
	form.Add("select[]", "TITLE")
	raw, err := poster.postForm(ctx, "tasks.task.get", form)
	if err != nil {
		return ""
	}
	detail, err := parseTaskGetResult(raw)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(detail.Title)
}

func lookupSonetGroupTitle(ctx context.Context, poster bitrixRESTPoster, groupID string) string {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" || groupID == "0" || poster == nil {
		return ""
	}
	form := url.Values{}
	form.Set("ID", groupID)
	raw, err := poster.postForm(ctx, "sonet_group.get", form)
	if err != nil {
		return ""
	}
	var row map[string]any
	if err := json.Unmarshal(bytesTrim(raw), &row); err != nil {
		return ""
	}
	return strings.TrimSpace(fieldFromRow(row, "NAME", "name"))
}

func lookupTaskStageLabel(ctx context.Context, poster bitrixRESTPoster, groupID, stageID string) string {
	groupID = strings.TrimSpace(groupID)
	stageID = strings.TrimSpace(stageID)
	if groupID == "" || stageID == "" || poster == nil {
		return ""
	}
	form := url.Values{}
	form.Set("entityId", groupID)
	raw, err := poster.postForm(ctx, "task.stages.get", form)
	if err != nil {
		return ""
	}
	raw = bytesTrim(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		var wrapped map[string]map[string]any
		if err2 := json.Unmarshal(raw, &wrapped); err2 == nil {
			for _, row := range wrapped {
				if label := matchTaskStageRow(row, stageID); label != "" {
					return label
				}
			}
		}
		return ""
	}
	for _, row := range rows {
		if label := matchTaskStageRow(row, stageID); label != "" {
			return label
		}
	}
	return ""
}

func matchTaskStageRow(row map[string]any, stageID string) string {
	id := fieldFromRow(row, "ID", "id")
	if id != stageID {
		return ""
	}
	label := strings.TrimSpace(fieldFromRow(row, "TITLE", "title", "NAME", "name"))
	if label == "" {
		return id
	}
	return label
}

func resolveCrmTaskLinkLabel(ctx context.Context, poster bitrixRESTPoster, link string) string {
	link = strings.TrimSpace(link)
	if link == "" {
		return ""
	}
	upper := strings.ToUpper(link)
	switch {
	case strings.HasPrefix(upper, "CO_"):
		id := strings.TrimSpace(strings.TrimPrefix(link, "CO_"))
		title := lookupCrmCompanyTitle(ctx, poster, id)
		return formatEntityReference("Компания", title, id, link)
	case strings.HasPrefix(upper, "C_"):
		id := strings.TrimSpace(strings.TrimPrefix(link, "C_"))
		title := lookupCrmContactTitle(ctx, poster, id)
		return formatEntityReference("Контакт", title, id, link)
	case strings.HasPrefix(upper, "D_"):
		id := strings.TrimSpace(strings.TrimPrefix(link, "D_"))
		title := lookupCrmDealTitle(ctx, poster, id)
		return formatEntityReference("Сделка", title, id, link)
	case strings.HasPrefix(upper, "L_"):
		id := strings.TrimSpace(strings.TrimPrefix(link, "L_"))
		title := lookupCrmLeadTitle(ctx, poster, id)
		return formatEntityReference("Лид", title, id, link)
	default:
		return link
	}
}

func formatEntityReference(kind, title, id, fallback string) string {
	title = strings.TrimSpace(title)
	id = strings.TrimSpace(id)
	switch {
	case title != "" && id != "":
		if kind != "" {
			return fmt.Sprintf("%s: %s (#%s)", kind, title, id)
		}
		return fmt.Sprintf("%s (#%s)", title, id)
	case title != "":
		if kind != "" {
			return kind + ": " + title
		}
		return title
	case id != "":
		if kind != "" {
			return fmt.Sprintf("%s #%s", kind, id)
		}
		return "#" + id
	default:
		return fallback
	}
}

func lookupCrmDealTitle(ctx context.Context, poster bitrixRESTPoster, dealID string) string {
	return lookupCrmEntityTitle(ctx, poster, "crm.deal.get", dealID)
}

func lookupCrmLeadTitle(ctx context.Context, poster bitrixRESTPoster, leadID string) string {
	return lookupCrmEntityTitle(ctx, poster, "crm.lead.get", leadID)
}

func lookupCrmContactTitle(ctx context.Context, poster bitrixRESTPoster, contactID string) string {
	return lookupCrmEntityTitle(ctx, poster, "crm.contact.get", contactID)
}

func lookupCrmCompanyTitle(ctx context.Context, poster bitrixRESTPoster, companyID string) string {
	return lookupCrmEntityTitle(ctx, poster, "crm.company.get", companyID)
}

func lookupCrmEntityTitle(ctx context.Context, poster bitrixRESTPoster, method, entityID string) string {
	entityID = strings.TrimSpace(entityID)
	if entityID == "" || entityID == "0" || poster == nil {
	 return ""
	}
	form := url.Values{}
	form.Set("id", entityID)
	raw, err := poster.postForm(ctx, method, form)
	if err != nil {
		return ""
	}
	var row map[string]any
	if err := json.Unmarshal(bytesTrim(raw), &row); err != nil {
		return ""
	}
	title := strings.TrimSpace(fieldFromRow(row, "TITLE", "title"))
	if title == "" {
		title = strings.TrimSpace(fieldFromRow(row, "NAME", "name"))
	}
	return title
}
