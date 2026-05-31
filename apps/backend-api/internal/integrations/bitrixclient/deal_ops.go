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

type BitrixDealBrief struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	StageID      string `json:"stageId"`
	StageLabel   string `json:"stageLabel"`
	CategoryID   string `json:"categoryId"`
	Opportunity  string `json:"opportunity,omitempty"`
	CurrencyID   string `json:"currencyId,omitempty"`
	AssignedByID string `json:"assignedById,omitempty"`
	DateCreate   string `json:"dateCreate,omitempty"`
	DateModify   string `json:"dateModify,omitempty"`
	Closed       string `json:"closed,omitempty"`
}

type BitrixDealDetail struct {
	BitrixDealBrief
	AssignedBy   BitrixTaskPerson        `json:"assignedBy,omitempty"`
	CreatedBy    BitrixTaskPerson        `json:"createdBy,omitempty"`
	ModifiedBy   BitrixTaskPerson        `json:"modifiedBy,omitempty"`
	CompanyID    string                  `json:"companyId,omitempty"`
	CompanyTitle string                  `json:"companyTitle,omitempty"`
	ContactID    string                  `json:"contactId,omitempty"`
	ContactTitle string                  `json:"contactTitle,omitempty"`
	LeadID       string                  `json:"leadId,omitempty"`
	LeadTitle    string                  `json:"leadTitle,omitempty"`
	QuoteID      string                  `json:"quoteId,omitempty"`
	Comments     string                  `json:"comments,omitempty"`
	AdditionalInfo string                `json:"additionalInfo,omitempty"`
	TypeID       string                  `json:"typeId,omitempty"`
	Probability  string                  `json:"probability,omitempty"`
	TaxValue     string                  `json:"taxValue,omitempty"`
	BeginDate    string                  `json:"beginDate,omitempty"`
	CloseDate    string                  `json:"closeDate,omitempty"`
	SourceID     string                  `json:"sourceId,omitempty"`
	SourceDesc   string                  `json:"sourceDescription,omitempty"`
	UtmSource    string                  `json:"utmSource,omitempty"`
	UtmMedium    string                  `json:"utmMedium,omitempty"`
	UtmCampaign  string                  `json:"utmCampaign,omitempty"`
	UtmContent   string                  `json:"utmContent,omitempty"`
	UtmTerm      string                  `json:"utmTerm,omitempty"`
	Opened       bool                    `json:"opened,omitempty"`
	IsNew        bool                    `json:"isNew,omitempty"`
	IsRecurring  bool                    `json:"isRecurring,omitempty"`
	IsReturnCustomer bool                `json:"isReturnCustomer,omitempty"`
	Fields       []BitrixDealField       `json:"fields,omitempty"`
	StageOptions []BitrixDealStageOption `json:"stageOptions"`
}

type BitrixDealStageOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Sort  int    `json:"-"`
}

type dealRESTCaller interface {
	callDealListGET(ctx context.Context, form url.Values) (json.RawMessage, error)
	callDealMethodPOST(ctx context.Context, method string, form url.Values) (json.RawMessage, error)
	callStatusList(ctx context.Context, entityID string) (json.RawMessage, error)
}

type webhookDealCaller struct {
	webhookURL string
	httpClient *http.Client
}

func (w *webhookDealCaller) callDealListGET(ctx context.Context, form url.Values) (json.RawMessage, error) {
	if strings.TrimSpace(w.webhookURL) == "" {
		return nil, fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	endpoint := strings.TrimRight(w.webhookURL, "/") + "/crm.deal.list.json?" + form.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseBitrixResult(body, resp.StatusCode)
}

func (w *webhookDealCaller) callDealMethodPOST(ctx context.Context, method string, form url.Values) (json.RawMessage, error) {
	if strings.TrimSpace(w.webhookURL) == "" {
		return nil, fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	endpoint := strings.TrimRight(w.webhookURL, "/") + "/" + strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(method), ".json"), "/") + ".json"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseBitrixResult(body, resp.StatusCode)
}

func (w *webhookDealCaller) callStatusList(ctx context.Context, entityID string) (json.RawMessage, error) {
	if strings.TrimSpace(w.webhookURL) == "" {
		return nil, fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return nil, fmt.Errorf("status entity id is required")
	}
	endpoint := strings.TrimRight(w.webhookURL, "/") + "/crm.status.list.json?filter[ENTITY_ID]=" + url.QueryEscape(entityID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseBitrixResult(body, resp.StatusCode)
}

type tokenDealCaller struct {
	token *TokenREST
}

func (t *tokenDealCaller) callDealListGET(ctx context.Context, form url.Values) (json.RawMessage, error) {
	if t.token == nil || !t.token.Configured() {
		return nil, fmt.Errorf("bitrix oauth client is not configured")
	}
	endpoint := t.token.methodURL("crm.deal.list")
	if len(form) > 0 {
		endpoint += "&" + form.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.token.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseBitrixResult(body, resp.StatusCode)
}

func (t *tokenDealCaller) callDealMethodPOST(ctx context.Context, method string, form url.Values) (json.RawMessage, error) {
	if t.token == nil || !t.token.Configured() {
		return nil, fmt.Errorf("bitrix oauth client is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.token.methodURL(method), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.token.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseBitrixResult(body, resp.StatusCode)
}

func (t *tokenDealCaller) callStatusList(ctx context.Context, entityID string) (json.RawMessage, error) {
	if t.token == nil || !t.token.Configured() {
		return nil, fmt.Errorf("bitrix oauth client is not configured")
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return nil, fmt.Errorf("status entity id is required")
	}
	endpoint := t.token.methodURL("crm.status.list") + "&filter[ENTITY_ID]=" + url.QueryEscape(entityID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.token.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseBitrixResult(body, resp.StatusCode)
}

func parseBitrixResult(body []byte, statusCode int) (json.RawMessage, error) {
	var envelope struct {
		Result           json.RawMessage `json:"result"`
		Error            any             `json:"error"`
		ErrorDescription string          `json:"error_description"`
	}
	if len(strings.TrimSpace(string(body))) > 0 {
		_ = json.Unmarshal(body, &envelope)
	}
	if statusCode >= 300 {
		return nil, fmt.Errorf("bitrix HTTP %s: %s", http.StatusText(statusCode), strings.TrimSpace(string(body)))
	}
	if envelope.Error != nil || envelope.ErrorDescription != "" {
		return nil, fmt.Errorf("bitrix error: %v, %s", envelope.Error, strings.TrimSpace(envelope.ErrorDescription))
	}
	return envelope.Result, nil
}

func (c *Client) ListDeals(ctx context.Context, limit int, search string) ([]BitrixDealBrief, error) {
	if !c.WebhookConfigured() {
		return nil, fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	return listDeals(ctx, &webhookDealCaller{webhookURL: c.webhookURL, httpClient: c.httpClient}, limit, search, 0, "")
}

func (t *TokenREST) ListDeals(ctx context.Context, limit int, search string) ([]BitrixDealBrief, error) {
	return listDeals(ctx, &tokenDealCaller{token: t}, limit, search, 0, "")
}

func (t *TokenREST) ListDealsForUser(ctx context.Context, userID int, limit int, search string) ([]BitrixDealBrief, error) {
	if !t.Configured() {
		return nil, fmt.Errorf("bitrix oauth client is not configured")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("bitrix user id is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	byID := make(map[string]BitrixDealBrief)
	appendUnique := func(items []BitrixDealBrief) {
		for _, item := range items {
			if item.ID == "" {
				continue
			}
			byID[item.ID] = item
		}
	}

	for _, filterField := range []string{"ASSIGNED_BY_ID", "CREATED_BY"} {
		items, err := listDeals(ctx, &tokenDealCaller{token: t}, limit, search, userID, filterField)
		if err == nil {
			appendUnique(items)
		}
	}

	if len(byID) == 0 {
		items, err := listDeals(ctx, &tokenDealCaller{token: t}, limit, search, 0, "")
		if err != nil {
			return nil, err
		}
		return items, nil
	}

	out := make([]BitrixDealBrief, 0, len(byID))
	for _, item := range byID {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (c *Client) GetDealDetail(ctx context.Context, dealID string) (BitrixDealDetail, error) {
	if !c.WebhookConfigured() {
		return BitrixDealDetail{}, fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	return getDealDetail(ctx, &webhookDealCaller{webhookURL: c.webhookURL, httpClient: c.httpClient}, dealID)
}

func (t *TokenREST) GetDealDetail(ctx context.Context, dealID string) (BitrixDealDetail, error) {
	return getDealDetail(ctx, &tokenDealCaller{token: t}, dealID)
}

func (c *Client) UpdateDealStageByID(ctx context.Context, dealID string, stageID string) error {
	if !c.WebhookConfigured() {
		return fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	return updateDealStageByID(ctx, &webhookDealCaller{webhookURL: c.webhookURL, httpClient: c.httpClient}, dealID, stageID)
}

func (t *TokenREST) UpdateDealStageByID(ctx context.Context, dealID string, stageID string) error {
	return updateDealStageByID(ctx, &tokenDealCaller{token: t}, dealID, stageID)
}

func (c *Client) UpdateDealFieldsByID(ctx context.Context, dealID string, fields map[string]string) error {
	if !c.WebhookConfigured() {
		return fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	return updateDealFieldsByID(ctx, &webhookDealCaller{webhookURL: c.webhookURL, httpClient: c.httpClient}, dealID, fields)
}

func (t *TokenREST) UpdateDealFieldsByID(ctx context.Context, dealID string, fields map[string]string) error {
	return updateDealFieldsByID(ctx, &tokenDealCaller{token: t}, dealID, fields)
}

func listDeals(ctx context.Context, caller dealRESTCaller, limit int, search string, filterUserID int, filterField string) ([]BitrixDealBrief, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	search = strings.TrimSpace(search)

	rows := make([]map[string]any, 0, limit)
	for start := 0; start < 500 && len(rows) < limit; start += 50 {
		form := url.Values{}
		form.Add("select[]", "ID")
		form.Add("select[]", "TITLE")
		form.Add("select[]", "STAGE_ID")
		form.Add("select[]", "CATEGORY_ID")
		form.Add("select[]", "OPPORTUNITY")
		form.Add("select[]", "CURRENCY_ID")
		form.Add("select[]", "ASSIGNED_BY_ID")
		form.Add("select[]", "DATE_CREATE")
		form.Add("select[]", "DATE_MODIFY")
		form.Add("select[]", "CLOSED")
		form.Set("order[ID]", "DESC")
		form.Set("start", strconv.Itoa(start))
		if search != "" {
			form.Set("filter[%TITLE]", search)
		}
		if filterUserID > 0 && strings.TrimSpace(filterField) != "" {
			form.Set("filter["+filterField+"]", strconv.Itoa(filterUserID))
		}

		raw, err := caller.callDealListGET(ctx, form)
		if err != nil {
			return nil, err
		}

		page, err := parseDealListResult(raw)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		rows = append(rows, page...)
		if len(page) < 50 {
			break
		}
	}

	if search != "" {
		needle := strings.ToLower(search)
		filtered := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			title := strings.ToLower(fieldFromRow(row, "TITLE", "title"))
			if strings.Contains(title, needle) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}

	deals := mapRowsToDealBriefs(rows)
	return enrichDealsWithStageLabels(ctx, caller, deals)
}

func getDealDetail(ctx context.Context, caller dealRESTCaller, dealID string) (BitrixDealDetail, error) {
	dealID = strings.TrimSpace(dealID)
	if dealID == "" {
		return BitrixDealDetail{}, fmt.Errorf("deal id is required")
	}

	row, err := fetchFullDealRow(ctx, caller, dealID)
	if err != nil {
		return BitrixDealDetail{}, err
	}
	return buildDealDetailFromRow(ctx, caller, row)
}

func updateDealStageByID(ctx context.Context, caller dealRESTCaller, dealID string, stageID string) error {
	dealID = strings.TrimSpace(dealID)
	stageID = strings.TrimSpace(stageID)
	if dealID == "" || stageID == "" {
		return fmt.Errorf("deal id and stage id are required")
	}
	form := url.Values{}
	form.Set("id", dealID)
	form.Set("fields[STAGE_ID]", stageID)
	_, err := caller.callDealMethodPOST(ctx, "crm.deal.update", form)
	return err
}

func updateDealFieldsByID(ctx context.Context, caller dealRESTCaller, dealID string, fields map[string]string) error {
	dealID = strings.TrimSpace(dealID)
	if dealID == "" {
		return fmt.Errorf("deal id is required")
	}
	if len(fields) == 0 {
		return fmt.Errorf("no fields to update")
	}
	form := url.Values{}
	form.Set("id", dealID)
	wrote := 0
	for key, value := range fields {
		key = strings.TrimSpace(strings.ToUpper(key))
		if key == "" {
			continue
		}
		if _, skip := dealReadOnlyFieldKeys[key]; skip {
			continue
		}
		form.Set("fields["+key+"]", value)
		wrote++
	}
	if wrote == 0 {
		return fmt.Errorf("no editable fields in request")
	}
	_, err := caller.callDealMethodPOST(ctx, "crm.deal.update", form)
	return err
}

func parseDealListResult(raw json.RawMessage) ([]map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var asList []map[string]any
	if err := json.Unmarshal(raw, &asList); err == nil {
		return asList, nil
	}

	var wrapped struct {
		Deals []map[string]any `json:"deals"`
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil {
		if len(wrapped.Deals) > 0 {
			return wrapped.Deals, nil
		}
		if len(wrapped.Items) > 0 {
			return wrapped.Items, nil
		}
	}

	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err == nil && len(asMap) > 0 {
		rows := make([]map[string]any, 0, len(asMap))
		for _, value := range asMap {
			if row, ok := value.(map[string]any); ok {
				rows = append(rows, row)
			}
		}
		if len(rows) > 0 {
			sort.Slice(rows, func(i, j int) bool {
				return fieldFromRow(rows[i], "ID", "id") > fieldFromRow(rows[j], "ID", "id")
			})
			return rows, nil
		}
	}

	return nil, fmt.Errorf("unexpected crm.deal.list result shape")
}

func dealStageEntityID(categoryID int) string {
	if categoryID <= 0 {
		return "DEAL_STAGE"
	}
	return fmt.Sprintf("DEAL_STAGE_%d", categoryID)
}

func mapRowsToDealBriefs(rows []map[string]any) []BitrixDealBrief {
	out := make([]BitrixDealBrief, 0, len(rows))
	for _, row := range rows {
		brief := mapRowToDealBrief(row)
		if brief.ID != "" {
			out = append(out, brief)
		}
	}
	return out
}

func mapRowToDealBrief(row map[string]any) BitrixDealBrief {
	categoryID := dealFieldRaw(row, "CATEGORY_ID", "categoryId")
	if categoryID == "" {
		categoryID = "0"
	}
	return BitrixDealBrief{
		ID:           fieldFromRow(row, "ID", "id"),
		Title:        fieldFromRow(row, "TITLE", "title"),
		StageID:      fieldFromRow(row, "STAGE_ID", "stageId"),
		CategoryID:   categoryID,
		Opportunity:  fieldFromRow(row, "OPPORTUNITY", "opportunity"),
		CurrencyID:   fieldFromRow(row, "CURRENCY_ID", "currencyId"),
		AssignedByID: fieldFromRow(row, "ASSIGNED_BY_ID", "assignedById"),
		DateCreate:   fieldFromRow(row, "DATE_CREATE", "dateCreate"),
		DateModify:   fieldFromRow(row, "DATE_MODIFY", "dateModify"),
		Closed:       fieldFromRow(row, "CLOSED", "closed"),
	}
}

func enrichDealsWithStageLabels(ctx context.Context, caller dealRESTCaller, deals []BitrixDealBrief) ([]BitrixDealBrief, error) {
	if len(deals) == 0 {
		return deals, nil
	}
	labelsByCategory := map[int]map[string]string{}
	for i, deal := range deals {
		categoryID, _ := strconv.Atoi(strings.TrimSpace(deal.CategoryID))
		labelMap, ok := labelsByCategory[categoryID]
		if !ok {
			options, err := listDealStageOptions(ctx, caller, categoryID)
			labelMap = make(map[string]string, len(options))
			if err == nil {
				for _, option := range options {
					labelMap[option.ID] = option.Label
				}
			}
			labelsByCategory[categoryID] = labelMap
		}
		if label, ok := labelMap[deal.StageID]; ok {
			deals[i].StageLabel = label
		} else if deal.StageID != "" {
			deals[i].StageLabel = deal.StageID
		}
	}
	return deals, nil
}

func dealFieldRaw(row map[string]any, keys ...string) string {
	for _, want := range keys {
		for k, v := range row {
			if strings.EqualFold(k, want) && v != nil {
				return strings.TrimSpace(fmt.Sprint(v))
			}
		}
	}
	return ""
}

func resolveDealCategoryID(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id < 0 {
		return 0
	}
	return id
}

func listDealStageOptions(ctx context.Context, caller dealRESTCaller, categoryID int) ([]BitrixDealStageOption, error) {
	if categoryID < 0 {
		categoryID = 0
	}

	entityIDs := []string{dealStageEntityID(categoryID)}
	if categoryID != 0 {
		entityIDs = append(entityIDs, "DEAL_STAGE")
	}

	var lastErr error
	for _, entityID := range entityIDs {
		items, err := fetchDealStatusListItems(ctx, caller, entityID)
		if err != nil {
			lastErr = err
			continue
		}
		options := mapStatusItemsToStageOptions(items, categoryID)
		if len(options) > 0 {
			return options, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return []BitrixDealStageOption{}, nil
}

func fetchDealStatusListItems(ctx context.Context, caller dealRESTCaller, entityID string) ([]map[string]any, error) {
	raw, err := caller.callStatusList(ctx, entityID)
	if err != nil {
		form := url.Values{}
		form.Set("filter[ENTITY_ID]", entityID)
		raw, err = caller.callDealMethodPOST(ctx, "crm.status.list", form)
		if err != nil {
			return nil, err
		}
	}
	return parseStatusListItems(raw)
}

func parseStatusListItems(raw json.RawMessage) ([]map[string]any, error) {
	raw = json.RawMessage(bytesTrimJSON(raw))
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("empty crm.status.list result")
	}

	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err == nil {
		return items, nil
	}

	var wrapped struct {
		Statuses []map[string]any `json:"statuses"`
		Items    []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil {
		if len(wrapped.Statuses) > 0 {
			return wrapped.Statuses, nil
		}
		if len(wrapped.Items) > 0 {
			return wrapped.Items, nil
		}
	}

	return nil, fmt.Errorf("unexpected crm.status.list result shape: %s", strings.TrimSpace(string(raw)))
}

func bytesTrimJSON(raw json.RawMessage) json.RawMessage {
	return json.RawMessage(strings.TrimSpace(string(raw)))
}

func mapStatusItemsToStageOptions(items []map[string]any, categoryID int) []BitrixDealStageOption {
	wantCategory := strconv.Itoa(categoryID)
	options := make([]BitrixDealStageOption, 0, len(items))
	for _, item := range items {
		itemCategory := dealFieldRaw(item, "CATEGORY_ID", "categoryId")
		if itemCategory == "" {
			itemCategory = "0"
		}
		if itemCategory != wantCategory {
			continue
		}
		id := fieldFromRow(item, "STATUS_ID", "statusId", "ID", "id")
		label := fieldFromRow(item, "NAME", "name")
		if label == "" {
			label = id
		}
		sv, _ := strconv.Atoi(fieldFromRow(item, "SORT", "sort"))
		if id == "" {
			continue
		}
		options = append(options, BitrixDealStageOption{ID: id, Label: label, Sort: sv})
	}
	sort.Slice(options, func(i, j int) bool {
		if options[i].Sort != options[j].Sort {
			return options[i].Sort < options[j].Sort
		}
		return options[i].Label < options[j].Label
	})
	return options
}

func userDisplayNameFromRow(row map[string]any) string {
	if formatted := strings.TrimSpace(fieldFromRow(row, "formattedName", "FORMATTED_NAME")); formatted != "" {
		return formatted
	}
	last := strings.TrimSpace(fieldFromRow(row, "LAST_NAME", "lastName"))
	first := strings.TrimSpace(fieldFromRow(row, "NAME", "name"))
	second := strings.TrimSpace(fieldFromRow(row, "SECOND_NAME", "secondName"))
	parts := make([]string, 0, 3)
	if last != "" {
		parts = append(parts, last)
	}
	if first != "" {
		parts = append(parts, first)
	}
	if second != "" {
		parts = append(parts, second)
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	if login := strings.TrimSpace(fieldFromRow(row, "LOGIN", "login")); login != "" {
		return login
	}
	return ""
}

func lookupBitrixUser(ctx context.Context, caller dealRESTCaller, userID string) BitrixTaskPerson {
	cache := newBitrixUserCache(dealRESTUserPoster{caller: caller})
	return cache.Resolve(ctx, userID)
}
