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

// BitrixDealField — поле сделки для отображения в мобильном приложении.
type BitrixDealField struct {
	Key       string                  `json:"key"`
	Label     string                  `json:"label"`
	Value     string                  `json:"value"`
	RawValue  string                  `json:"rawValue,omitempty"`
	Editable  bool                    `json:"editable,omitempty"`
	Type      string                  `json:"type,omitempty"`
	Options   []BitrixDealFieldOption `json:"options,omitempty"`
}

type BitrixDealFieldOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type dealFieldMeta struct {
	fieldType string
	readOnly  bool
	immutable bool
	options   []BitrixDealFieldOption
}

var dealStandardFieldLabels = map[string]string{
	"ID":                     "ID",
	"TITLE":                  "Название",
	"TYPE_ID":                "Тип сделки",
	"STAGE_ID":               "Стадия",
	"STAGE_SEMANTIC_ID":      "Семантика стадии",
	"CATEGORY_ID":            "Воронка",
	"PROBABILITY":            "Вероятность, %",
	"CURRENCY_ID":            "Валюта",
	"OPPORTUNITY":            "Сумма",
	"IS_MANUAL_OPPORTUNITY":  "Сумма введена вручную",
	"TAX_VALUE":              "Налог",
	"LEAD_ID":                "Лид",
	"COMPANY_ID":             "Компания",
	"CONTACT_ID":             "Контакт",
	"QUOTE_ID":               "Предложение",
	"BEGINDATE":              "Дата начала",
	"CLOSEDATE":              "Дата завершения",
	"ASSIGNED_BY_ID":         "Ответственный",
	"CREATED_BY_ID":          "Кем создана",
	"MODIFY_BY_ID":           "Кем изменена",
	"MOVED_BY_ID":            "Кем перемещена",
	"DATE_CREATE":            "Создана",
	"DATE_MODIFY":            "Изменена",
	"OPENED":                 "Открыта",
	"CLOSED":                 "Закрыта",
	"COMMENTS":               "Комментарий",
	"ADDITIONAL_INFO":        "Дополнительно",
	"LOCATION_ID":            "Местоположение",
	"SOURCE_ID":              "Источник",
	"SOURCE_DESCRIPTION":     "Описание источника",
	"ORIGINATOR_ID":          "Источник (внешний)",
	"ORIGIN_ID":              "Внешний ID",
	"IS_NEW":                 "Новая",
	"IS_RECURRING":           "Регулярная",
	"IS_RETURN_CUSTOMER":     "Повторная продажа",
	"IS_REPEATED_APPROACH":   "Повторное обращение",
	"UTM_SOURCE":             "UTM Source",
	"UTM_MEDIUM":             "UTM Medium",
	"UTM_CAMPAIGN":           "UTM Campaign",
	"UTM_CONTENT":            "UTM Content",
	"UTM_TERM":               "UTM Term",
	"LAST_ACTIVITY_TIME":     "Последняя активность",
	"LAST_ACTIVITY_BY":       "Автор последней активности",
	"LAST_COMMUNICATION_TIME": "Последняя коммуникация",
}

var dealReadOnlyFieldKeys = map[string]struct{}{
	"ID": {}, "DATE_CREATE": {}, "DATE_MODIFY": {}, "CREATED_BY_ID": {}, "MODIFY_BY_ID": {},
	"MOVED_BY_ID": {}, "STAGE_SEMANTIC_ID": {}, "MOVED_BY_NAME": {}, "LAST_ACTIVITY_BY_NAME": {},
	"PRODUCTS": {}, "LINKED_CONTACTS": {}, "STAGE_ID": {},
}

func buildDealDetailFromRow(ctx context.Context, caller dealRESTCaller, row map[string]any) (BitrixDealDetail, error) {
	row = normalizeDealRow(row)
	fieldDefs := loadDealFieldDefinitions(ctx, caller)
	rowRaw := cloneDealRow(row)
	row = resolveDealEnumFields(row, fieldDefs.enums)

	users := newBitrixUserCache(dealRESTUserPoster{caller: caller})
	users.Prefetch(ctx, collectDealUserIDs(row, fieldDefs))

	brief := mapRowToDealBrief(row)
	enriched, _ := enrichDealsWithStageLabels(ctx, caller, []BitrixDealBrief{brief})
	if len(enriched) > 0 {
		brief = enriched[0]
	}

	detail := BitrixDealDetail{
		BitrixDealBrief: brief,
		Comments:        fieldFromRow(row, "COMMENTS", "comments"),
		AdditionalInfo:  fieldFromRow(row, "ADDITIONAL_INFO", "additionalInfo"),
		TypeID:          fieldFromRow(row, "TYPE_ID", "typeId"),
		Probability:     fieldFromRow(row, "PROBABILITY", "probability"),
		TaxValue:        fieldFromRow(row, "TAX_VALUE", "taxValue"),
		BeginDate:       fieldFromRow(row, "BEGINDATE", "beginDate"),
		CloseDate:       fieldFromRow(row, "CLOSEDATE", "closeDate"),
		SourceID:        fieldFromRow(row, "SOURCE_ID", "sourceId"),
		SourceDesc:      fieldFromRow(row, "SOURCE_DESCRIPTION", "sourceDescription"),
		LeadID:          fieldFromRow(row, "LEAD_ID", "leadId"),
		CompanyID:       fieldFromRow(row, "COMPANY_ID", "companyId"),
		ContactID:       fieldFromRow(row, "CONTACT_ID", "contactId"),
		QuoteID:         fieldFromRow(row, "QUOTE_ID", "quoteId"),
		UtmSource:       fieldFromRow(row, "UTM_SOURCE", "utmSource"),
		UtmMedium:       fieldFromRow(row, "UTM_MEDIUM", "utmMedium"),
		UtmCampaign:     fieldFromRow(row, "UTM_CAMPAIGN", "utmCampaign"),
		UtmContent:      fieldFromRow(row, "UTM_CONTENT", "utmContent"),
		UtmTerm:         fieldFromRow(row, "UTM_TERM", "utmTerm"),
		Opened:          boolFromRow(row, "OPENED", "opened"),
		IsNew:           boolFromRow(row, "IS_NEW", "isNew"),
		IsRecurring:     boolFromRow(row, "IS_RECURRING", "isRecurring"),
		IsReturnCustomer: boolFromRow(row, "IS_RETURN_CUSTOMER", "isReturnCustomer"),
	}

	if brief.AssignedByID != "" {
		detail.AssignedBy = users.Resolve(ctx, brief.AssignedByID)
		if detail.AssignedBy.ID == "" {
			detail.AssignedBy.ID = brief.AssignedByID
		}
	}
	if createdBy := fieldFromRow(row, "CREATED_BY_ID", "createdById"); createdBy != "" && createdBy != "0" {
		detail.CreatedBy = users.Resolve(ctx, createdBy)
	}
	if modifiedBy := fieldFromRow(row, "MODIFY_BY_ID", "modifyById"); modifiedBy != "" && modifiedBy != "0" {
		detail.ModifiedBy = users.Resolve(ctx, modifiedBy)
	}
	if detail.CompanyID != "" && detail.CompanyID != "0" {
		detail.CompanyTitle = lookupBitrixCompany(ctx, caller, detail.CompanyID)
	}
	if detail.ContactID != "" && detail.ContactID != "0" {
		detail.ContactTitle = lookupBitrixContact(ctx, caller, detail.ContactID)
	}
	if detail.LeadID != "" && detail.LeadID != "0" {
		detail.LeadTitle = lookupBitrixLead(ctx, caller, detail.LeadID)
	}

	resolvers := dealFieldResolvers{
		stageLabel: brief.StageLabel,
		assignedBy: detail.AssignedBy.Name,
		createdBy:  detail.CreatedBy.Name,
		modifiedBy: detail.ModifiedBy.Name,
		company:    detail.CompanyTitle,
		contact:    detail.ContactTitle,
		lead:       detail.LeadTitle,
		userName: func(userID string) string {
			return users.Name(ctx, userID)
		},
	}
	detail.Fields = collectDealFields(row, rowRaw, fieldDefs, resolvers)

	categoryID := resolveDealCategoryID(detail.CategoryID)
	if options, err := listDealStageOptions(ctx, caller, categoryID); err == nil && len(options) > 0 {
		detail.StageOptions = options
	} else if options, err := listDealStageOptions(ctx, caller, 0); err == nil && len(options) > 0 {
		detail.StageOptions = options
	} else {
		detail.StageOptions = []BitrixDealStageOption{}
	}
	return detail, nil
}

func cloneDealRow(row map[string]any) map[string]any {
	out := make(map[string]any, len(row))
	for key, value := range row {
		out[key] = value
	}
	return out
}

type dealFieldResolvers struct {
	stageLabel string
	assignedBy string
	createdBy  string
	modifiedBy string
	company    string
	contact    string
	lead       string
	userName   func(userID string) string
}

func loadDealFieldLabels(ctx context.Context, caller dealRESTCaller) map[string]string {
	return loadDealFieldDefinitions(ctx, caller).labels
}

func collectDealFields(row map[string]any, rowRaw map[string]any, defs dealFieldDefinitions, resolvers dealFieldResolvers) []BitrixDealField {
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]BitrixDealField, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		upper := strings.ToUpper(key)
		if _, ok := seen[upper]; ok {
			continue
		}
		meta := defs.meta[upper]
		value := formatDealFieldValue(key, row[key], resolvers)
		if value != "" && resolvers.userName != nil && isBitrixUserFieldKey(upper, meta.fieldType) {
			userID := dealFieldRaw(map[string]any{key: rowRaw[upper]})
			if userID == "" {
				userID = dealFieldRaw(map[string]any{key: row[upper]})
			}
			if userID != "" {
				if name := resolvers.userName(userID); name != "" {
					value = name
				}
			}
		}
		if value == "" {
			continue
		}
		seen[upper] = struct{}{}
		label := defs.labels[upper]
		if label == "" {
			label = humanizeDealFieldKey(upper)
		}
		rawValue := formatDealScalar(rowRaw[upper])
		if rawValue == "" {
			rawValue = formatDealScalar(row[upper])
		}
		out = append(out, BitrixDealField{
			Key:      upper,
			Label:    label,
			Value:    value,
			RawValue: rawValue,
			Editable: isDealFieldEditable(upper, meta),
			Type:     meta.fieldType,
			Options:  meta.options,
		})
	}
	return out
}

func isDealFieldEditable(code string, meta dealFieldMeta) bool {
	if _, skip := dealReadOnlyFieldKeys[code]; skip {
		return false
	}
	if meta.readOnly || meta.immutable {
		return false
	}
	switch strings.ToLower(meta.fieldType) {
	case "file", "resourcebooking", "employee", "crm", "crm_status", "char":
		return false
	default:
		return true
	}
}

func humanizeDealFieldKey(key string) string {
	if strings.HasPrefix(key, "UF_CRM_") {
		return "Поле " + strings.TrimPrefix(key, "UF_CRM_")
	}
	return strings.ReplaceAll(key, "_", " ")
}

func formatDealFieldValue(key string, raw any, resolvers dealFieldResolvers) string {
	upper := strings.ToUpper(strings.TrimSpace(key))
	switch upper {
	case "STAGE_ID":
		if resolvers.stageLabel != "" {
			return resolvers.stageLabel
		}
	case "ASSIGNED_BY_ID":
		if resolvers.assignedBy != "" {
			return resolvers.assignedBy
		}
	case "CREATED_BY_ID":
		if resolvers.createdBy != "" {
			return resolvers.createdBy
		}
	case "MODIFY_BY_ID":
		if resolvers.modifiedBy != "" {
			return resolvers.modifiedBy
		}
	case "COMPANY_ID":
		if resolvers.company != "" {
			return resolvers.company
		}
	case "CONTACT_ID":
		if resolvers.contact != "" {
			return resolvers.contact
		}
	case "MOVED_BY_ID":
		if resolvers.userName != nil {
			if name := resolvers.userName(dealFieldRaw(map[string]any{key: raw})); name != "" {
				return name
			}
		}
	case "LAST_ACTIVITY_BY":
		if resolvers.userName != nil {
			if name := resolvers.userName(dealFieldRaw(map[string]any{key: raw})); name != "" {
				return name
			}
		}
	case "LEAD_ID":
		if resolvers.lead != "" {
			return resolvers.lead
		}
	}
	if resolvers.userName != nil && isBitrixUserFieldKey(upper, "") {
		if id := dealFieldRaw(map[string]any{key: raw}); id != "" {
			if name := resolvers.userName(id); name != "" {
				return name
			}
		}
	}
	return formatDealScalar(raw)
}

func formatDealScalar(raw any) string {
	switch v := raw.(type) {
	case nil:
		return ""
	case string:
		s := strings.TrimSpace(v)
		if s == "" || s == "0" || strings.EqualFold(s, "null") {
			return ""
		}
		switch strings.ToUpper(s) {
		case "Y", "YES":
			return "Да"
		case "N", "NO":
			return "Нет"
		default:
			return s
		}
	case bool:
		if v {
			return "Да"
		}
		return "Нет"
	case float64:
		if v == 0 {
			return ""
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		if v == 0 {
			return ""
		}
		return strconv.Itoa(v)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			part := formatDealScalar(item)
			if part != "" {
				parts = append(parts, part)
			}
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		if title := strings.TrimSpace(fieldFromRow(v, "TITLE", "title", "NAME", "name")); title != "" {
			return title
		}
		return strings.TrimSpace(fmt.Sprint(v))
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "0" || s == "<nil>" {
			return ""
		}
		return s
	}
}

func lookupBitrixCompany(ctx context.Context, caller dealRESTCaller, companyID string) string {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" || companyID == "0" {
		return ""
	}
	form := url.Values{}
	form.Set("id", companyID)
	raw, err := caller.callDealMethodPOST(ctx, "crm.company.get", form)
	if err != nil {
		return ""
	}
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		return ""
	}
	title := strings.TrimSpace(fieldFromRow(row, "TITLE", "title"))
	if title != "" {
		return title
	}
	return ""
}

func lookupBitrixContact(ctx context.Context, caller dealRESTCaller, contactID string) string {
	contactID = strings.TrimSpace(contactID)
	if contactID == "" || contactID == "0" {
		return ""
	}
	form := url.Values{}
	form.Set("id", contactID)
	raw, err := caller.callDealMethodPOST(ctx, "crm.contact.get", form)
	if err != nil {
		return ""
	}
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		return ""
	}
	return userDisplayNameFromRow(row)
}

func lookupBitrixLead(ctx context.Context, caller dealRESTCaller, leadID string) string {
	leadID = strings.TrimSpace(leadID)
	if leadID == "" || leadID == "0" {
		return ""
	}
	form := url.Values{}
	form.Set("id", leadID)
	raw, err := caller.callDealMethodPOST(ctx, "crm.lead.get", form)
	if err != nil {
		return ""
	}
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		return ""
	}
	return strings.TrimSpace(fieldFromRow(row, "TITLE", "title"))
}

type dealFieldDefinitions struct {
	labels map[string]string
	meta   map[string]dealFieldMeta
	enums  map[string]map[string]string
	codes  []string
}

func fetchFullDealRow(ctx context.Context, caller dealRESTCaller, dealID string) (map[string]any, error) {
	dealID = strings.TrimSpace(dealID)
	if dealID == "" {
		return nil, fmt.Errorf("deal id is required")
	}

	defs := loadDealFieldDefinitions(ctx, caller)

	form := url.Values{}
	form.Set("id", dealID)
	rawGet, err := caller.callDealMethodPOST(ctx, "crm.deal.get", form)
	if err != nil {
		return nil, err
	}
	var rowGet map[string]any
	if err := json.Unmarshal(rawGet, &rowGet); err != nil {
		return nil, err
	}

	rowList, _ := fetchDealRowByList(ctx, caller, dealID, defs.codes)
	merged := mergeDealRows(rowGet, rowList)
	appendDealExtraFields(ctx, caller, dealID, merged)
	return normalizeDealRow(merged), nil
}

func fetchDealRowByList(ctx context.Context, caller dealRESTCaller, dealID string, fieldCodes []string) (map[string]any, error) {
	if len(fieldCodes) == 0 {
		return nil, nil
	}
	form := url.Values{}
	form.Set("filter[ID]", dealID)
	for _, code := range fieldCodes {
		code = strings.TrimSpace(code)
		if code != "" {
			form.Add("select[]", code)
		}
	}
	raw, err := caller.callDealListGET(ctx, form)
	if err != nil {
		return nil, err
	}
	page, err := parseDealListResult(raw)
	if err != nil || len(page) == 0 {
		return nil, err
	}
	return page[0], nil
}

func mergeDealRows(rows ...map[string]any) map[string]any {
	out := make(map[string]any)
	for _, row := range rows {
		if row == nil {
			continue
		}
		for key, value := range row {
			upper := strings.ToUpper(strings.TrimSpace(key))
			if upper == "CATEGORY_ID" {
				category := strings.TrimSpace(fmt.Sprint(value))
				if category == "" {
					category = "0"
				}
				out[upper] = category
				continue
			}
			if isEmptyDealValue(value) {
				continue
			}
			if existing, ok := out[upper]; ok && !isEmptyDealValue(existing) {
				continue
			}
			out[upper] = value
		}
	}
	return out
}

func normalizeDealRow(row map[string]any) map[string]any {
	return mergeDealRows(row)
}

func isEmptyDealValue(raw any) bool {
	return formatDealScalar(raw) == ""
}

func appendDealExtraFields(ctx context.Context, caller dealRESTCaller, dealID string, row map[string]any) {
	if products := loadDealProducts(ctx, caller, dealID); products != "" {
		row["PRODUCTS"] = products
	}
	if contacts := loadDealLinkedContacts(ctx, caller, dealID); contacts != "" {
		row["LINKED_CONTACTS"] = contacts
	}
	if movedBy := fieldFromRow(row, "MOVED_BY_ID"); movedBy != "" && movedBy != "0" {
		if name := lookupBitrixUser(ctx, caller, movedBy).Name; name != "" {
			row["MOVED_BY_NAME"] = name
		}
	}
	if lastActivityBy := fieldFromRow(row, "LAST_ACTIVITY_BY"); lastActivityBy != "" && lastActivityBy != "0" {
		if name := lookupBitrixUser(ctx, caller, lastActivityBy).Name; name != "" {
			row["LAST_ACTIVITY_BY_NAME"] = name
		}
	}
}

func loadDealProducts(ctx context.Context, caller dealRESTCaller, dealID string) string {
	form := url.Values{}
	form.Set("id", dealID)
	raw, err := caller.callDealMethodPOST(ctx, "crm.deal.productrows.get", form)
	if err != nil {
		return ""
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return ""
	}
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(fieldFromRow(row, "PRODUCT_NAME", "productName", "ORIGINAL_PRODUCT_NAME", "originalProductName"))
		if name == "" {
			name = strings.TrimSpace(fieldFromRow(row, "PRODUCT_ID", "productId"))
		}
		qty := strings.TrimSpace(fieldFromRow(row, "QUANTITY", "quantity"))
		price := strings.TrimSpace(fieldFromRow(row, "PRICE", "price"))
		if name == "" {
			continue
		}
		line := name
		if qty != "" {
			line += " × " + qty
		}
		if price != "" {
			line += " (" + price + ")"
		}
		parts = append(parts, line)
	}
	return strings.Join(parts, "; ")
}

func loadDealLinkedContacts(ctx context.Context, caller dealRESTCaller, dealID string) string {
	form := url.Values{}
	form.Set("id", dealID)
	raw, err := caller.callDealMethodPOST(ctx, "crm.deal.contact.items.get", form)
	if err != nil {
		return ""
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return ""
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		contactID := fieldFromRow(row, "CONTACT_ID", "contactId")
		if contactID == "" || contactID == "0" {
			continue
		}
		if name := lookupBitrixContact(ctx, caller, contactID); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

func loadDealFieldDefinitions(ctx context.Context, caller dealRESTCaller) dealFieldDefinitions {
	defs := dealFieldDefinitions{
		labels: make(map[string]string, len(dealStandardFieldLabels)),
		meta:   make(map[string]dealFieldMeta),
		enums:  make(map[string]map[string]string),
	}
	for key, label := range dealStandardFieldLabels {
		defs.labels[strings.ToUpper(key)] = label
	}
	defs.labels["PRODUCTS"] = "Товары"
	defs.labels["LINKED_CONTACTS"] = "Связанные контакты"
	defs.labels["MOVED_BY_NAME"] = "Кем перемещена"
	defs.labels["LAST_ACTIVITY_BY_NAME"] = "Автор последней активности"

	raw, err := caller.callDealMethodPOST(ctx, "crm.deal.fields", url.Values{})
	if err != nil {
		defs.codes = dealDefaultFieldCodes()
		return defs
	}

	codes := make([]string, 0, 128)
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err == nil {
		for _, item := range items {
			mergeDealFieldDefinition(defs, item, &codes)
		}
	} else {
		var byCode map[string]any
		if err := json.Unmarshal(raw, &byCode); err == nil {
			for code, value := range byCode {
				item, ok := value.(map[string]any)
				if !ok {
					continue
				}
				item["code"] = code
				mergeDealFieldDefinition(defs, item, &codes)
			}
		}
	}
	if len(codes) == 0 {
		codes = dealDefaultFieldCodes()
	}
	defs.codes = codes
	return defs
}

func dealDefaultFieldCodes() []string {
	return []string{
		"ID", "TITLE", "TYPE_ID", "STAGE_ID", "STAGE_SEMANTIC_ID", "CATEGORY_ID", "PROBABILITY",
		"CURRENCY_ID", "OPPORTUNITY", "IS_MANUAL_OPPORTUNITY", "TAX_VALUE", "LEAD_ID", "COMPANY_ID",
		"CONTACT_ID", "QUOTE_ID", "BEGINDATE", "CLOSEDATE", "ASSIGNED_BY_ID", "CREATED_BY_ID",
		"MODIFY_BY_ID", "MOVED_BY_ID", "DATE_CREATE", "DATE_MODIFY", "OPENED", "CLOSED", "COMMENTS",
		"ADDITIONAL_INFO", "LOCATION_ID", "SOURCE_ID", "SOURCE_DESCRIPTION", "ORIGINATOR_ID", "ORIGIN_ID",
		"IS_NEW", "IS_RECURRING", "IS_RETURN_CUSTOMER", "IS_REPEATED_APPROACH",
		"UTM_SOURCE", "UTM_MEDIUM", "UTM_CAMPAIGN", "UTM_CONTENT", "UTM_TERM",
		"LAST_ACTIVITY_TIME", "LAST_ACTIVITY_BY", "LAST_COMMUNICATION_TIME",
	}
}

func mergeDealFieldDefinition(defs dealFieldDefinitions, item map[string]any, codes *[]string) {
	code := strings.ToUpper(strings.TrimSpace(fieldFromRow(item, "FIELD_NAME", "fieldName", "CODE", "code")))
	if code == "" {
		return
	}
	*codes = append(*codes, code)
	title := strings.TrimSpace(fieldFromRow(item, "EDIT_FORM_LABEL", "editFormLabel", "LIST_COLUMN_LABEL", "listColumnLabel", "TITLE", "title"))
	if title != "" {
		defs.labels[code] = title
	}
	meta := dealFieldMeta{
		fieldType: strings.ToLower(strings.TrimSpace(fieldFromRow(item, "TYPE", "type", "USER_TYPE_ID", "userTypeId"))),
		readOnly:  boolFromRow(item, "IS_READONLY", "isReadOnly", "READONLY"),
		immutable: boolFromRow(item, "IS_IMMUTABLE", "isImmutable"),
	}
	enumItems, ok := item["items"].([]any)
	if ok {
		valueMap := make(map[string]string)
		options := make([]BitrixDealFieldOption, 0, len(enumItems))
		for _, enumItem := range enumItems {
			m, ok := enumItem.(map[string]any)
			if !ok {
				continue
			}
			id := strings.TrimSpace(fieldFromRow(m, "ID", "id"))
			value := strings.TrimSpace(fieldFromRow(m, "VALUE", "value", "NAME", "name"))
			if id != "" && value != "" {
				valueMap[id] = value
				options = append(options, BitrixDealFieldOption{ID: id, Label: value})
			}
		}
		if len(valueMap) > 0 {
			defs.enums[code] = valueMap
			meta.options = options
		}
	}
	defs.meta[code] = meta
}

func resolveDealEnumFields(row map[string]any, enums map[string]map[string]string) map[string]any {
	if len(enums) == 0 {
		return row
	}
	out := make(map[string]any, len(row))
	for key, raw := range row {
		upper := strings.ToUpper(key)
		if enumMap, ok := enums[upper]; ok {
			out[key] = resolveEnumDealValue(raw, enumMap)
			continue
		}
		out[key] = raw
	}
	return out
}

func resolveEnumDealValue(raw any, enumMap map[string]string) any {
	switch v := raw.(type) {
	case string:
		if label, ok := enumMap[strings.TrimSpace(v)]; ok {
			return label
		}
		return raw
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			id := strings.TrimSpace(fmt.Sprint(item))
			if label, ok := enumMap[id]; ok {
				parts = append(parts, label)
			} else if id != "" {
				parts = append(parts, id)
			}
		}
		return strings.Join(parts, ", ")
	default:
		id := strings.TrimSpace(fmt.Sprint(raw))
		if label, ok := enumMap[id]; ok {
			return label
		}
		return raw
	}
}
