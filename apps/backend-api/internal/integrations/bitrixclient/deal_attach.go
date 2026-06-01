package bitrixclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// AttachEstimateParams — прикрепление сметы к полю сделки + уведомление в ленте.
type AttachEstimateParams struct {
	// FieldCode — UF_CRM_… поля «Смета»; пусто — авто-поиск по подписи в crm.deal.fields.
	FieldCode string
	// Notice — текст комментария в ленте (без файла).
	Notice string
}

// AttachEstimateToDeal записывает файл в пользовательское поле сделки и добавляет текст в ленту.
func (c *Client) AttachEstimateToDeal(ctx context.Context, dealID int, fileName string, content []byte, params AttachEstimateParams) error {
	if !c.WebhookConfigured() {
		return fmt.Errorf("BITRIX_WEBHOOK_URL is empty")
	}
	if dealID <= 0 {
		return fmt.Errorf("deal id is required")
	}
	if len(content) == 0 {
		return fmt.Errorf("file content is empty")
	}
	return attachEstimateToDealVia(ctx, webhookREST{webhookURL: c.webhookURL, httpClient: c.httpClient}, dealID, fileName, content, params)
}

func (t *TokenREST) AttachEstimateToDeal(ctx context.Context, dealID int, fileName string, content []byte, params AttachEstimateParams) error {
	if !t.Configured() {
		return fmt.Errorf("bitrix oauth client is not configured")
	}
	if dealID <= 0 {
		return fmt.Errorf("deal id is required")
	}
	if len(content) == 0 {
		return fmt.Errorf("file content is empty")
	}
	return attachEstimateToDealVia(ctx, tokenRESTPoster{token: t}, dealID, fileName, content, params)
}

// AttachFileToDeal — совместимость; предпочтительно AttachEstimateToDeal.
func (c *Client) AttachFileToDeal(ctx context.Context, dealID int, fileName, comment string, content []byte) error {
	return c.AttachEstimateToDeal(ctx, dealID, fileName, content, AttachEstimateParams{Notice: comment})
}

func (t *TokenREST) AttachFileToDeal(ctx context.Context, dealID int, fileName, comment string, content []byte) error {
	return t.AttachEstimateToDeal(ctx, dealID, fileName, content, AttachEstimateParams{Notice: comment})
}

func attachEstimateToDealVia(ctx context.Context, poster bitrixRESTPoster, dealID int, fileName string, content []byte, params AttachEstimateParams) error {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		fileName = "smeta.docx"
	}
	notice := strings.TrimSpace(params.Notice)
	if notice == "" {
		notice = "Смета добавлена, на проверку."
	}

	fieldCode, err := resolveDealEstimateFieldCode(ctx, poster, params.FieldCode)
	if err != nil {
		return err
	}

	if err := setDealFileField(ctx, poster, dealID, fieldCode, fileName, content); err != nil {
		return fmt.Errorf("поле сделки %s: %w", fieldCode, err)
	}

	if err := addDealTimelineNotice(ctx, poster, dealID, notice); err != nil {
		return fmt.Errorf("комментарий в ленте: %w", err)
	}
	return nil
}

func resolveDealEstimateFieldCode(ctx context.Context, poster bitrixRESTPoster, override string) (string, error) {
	override = strings.ToUpper(strings.TrimSpace(override))
	if override != "" {
		return override, nil
	}

	var exact, partial string
	var fileFields []string

	collect := func(code, label, fieldType, userType string) {
		if !isFileLikeField(fieldType, userType) {
			return
		}
		fileFields = append(fileFields, fmt.Sprintf("%s (%s)", code, label))
		norm := normalizeFieldLabel(label)
		if norm == "смета" {
			exact = code
		} else if partial == "" && strings.Contains(norm, "смета") && !strings.Contains(norm, "сметная") {
			partial = code
		}
	}

	if raw, err := poster.postForm(ctx, "crm.deal.userfield.list", url.Values{}); err == nil {
		parseUserfieldList(raw, collect)
	}
	if raw, err := poster.postForm(ctx, "crm.deal.fields", url.Values{}); err == nil {
		parseDealFieldsCatalog(raw, collect)
	}

	if exact != "" {
		return exact, nil
	}
	if partial != "" {
		return partial, nil
	}
	if len(fileFields) == 1 {
		code := strings.Fields(fileFields[0])[0]
		return strings.Trim(code, "()"), nil
	}
	hint := "задайте BITRIX_DEAL_ESTIMATE_FIELD=UF_CRM_… в .env"
	if len(fileFields) > 0 {
		hint += "; файловые поля: " + strings.Join(fileFields, ", ")
	}
	return "", fmt.Errorf("не найдено поле «Смета» (тип «файл») — %s", hint)
}

func parseUserfieldList(raw json.RawMessage, fn func(code, label, fieldType, userType string)) {
	var envelope struct {
		Result []map[string]any `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return
	}
	for _, item := range envelope.Result {
		code := strings.ToUpper(strings.TrimSpace(fieldFromRow(item, "FIELD_NAME", "fieldName")))
		if code == "" {
			continue
		}
		label := bitrixFieldLabel(item)
		fieldType := strings.ToLower(strings.TrimSpace(fieldFromRow(item, "USER_TYPE_ID", "userTypeId", "TYPE", "type")))
		userType := fieldType
		fn(code, label, fieldType, userType)
	}
}

func parseDealFieldsCatalog(raw json.RawMessage, fn func(code, label, fieldType, userType string)) {
	var envelope struct {
		Result any `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Result != nil {
		raw, _ = json.Marshal(envelope.Result)
	}

	var byCode map[string]any
	if err := json.Unmarshal(raw, &byCode); err == nil {
		for code, value := range byCode {
			emitDealFieldItem(code, value, fn)
		}
		return
	}

	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err == nil {
		for _, item := range items {
			code := fieldFromRow(item, "FIELD_NAME", "fieldName", "CODE", "code")
			emitDealFieldItem(code, item, fn)
		}
	}
}

func emitDealFieldItem(code string, value any, fn func(code, label, fieldType, userType string)) {
	item, ok := value.(map[string]any)
	if !ok {
		return
	}
	if strings.TrimSpace(code) == "" {
		code = fieldFromRow(item, "FIELD_NAME", "fieldName", "CODE", "code")
	}
	label := bitrixFieldLabel(item)
	fieldType := strings.ToLower(strings.TrimSpace(fieldFromRow(item, "TYPE", "type")))
	userType := strings.ToLower(strings.TrimSpace(fieldFromRow(item, "USER_TYPE_ID", "userTypeId")))
	fn(strings.ToUpper(strings.TrimSpace(code)), label, fieldType, userType)
}

func bitrixFieldLabel(item map[string]any) string {
	for _, key := range []string{"EDIT_FORM_LABEL", "LIST_COLUMN_LABEL", "TITLE", "FIELD_NAME"} {
		if v, ok := item[key]; ok {
			if label := stringFromBitrixLocalized(v); label != "" {
				return label
			}
		}
	}
	return ""
}

func stringFromBitrixLocalized(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case map[string]any:
		for _, lang := range []string{"ru", "RU", "en", "EN"} {
			if s, ok := t[lang].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
		for _, val := range t {
			if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func normalizeFieldLabel(label string) string {
	return strings.ToLower(strings.TrimSpace(label))
}

func isFileLikeField(fieldType, userType string) bool {
	if fieldType == "file" {
		return true
	}
	return strings.Contains(userType, "file") || strings.Contains(userType, "disk")
}

func setDealFileField(ctx context.Context, poster bitrixRESTPoster, dealID int, fieldCode, fileName string, content []byte) error {
	b64 := base64.StdEncoding.EncodeToString(content)
	beforeID := readDealFileFieldID(ctx, poster, dealID, fieldCode)
	var errs []string

	attempt := func(label string, fn func() error) bool {
		if err := fn(); err != nil {
			errs = append(errs, label+": "+err.Error())
			return false
		}
		if dealFileFieldUpdated(ctx, poster, dealID, fieldCode, beforeID) {
			return true
		}
		errs = append(errs, label+": API OK, но файл в поле сделки не обновился")
		return false
	}

	if jp, ok := poster.(bitrixRESTJSONPoster); ok {
		for _, code := range fieldCodeVariants(fieldCode) {
			for _, fieldValue := range fileFieldValueVariants(fileName, b64) {
				code, fieldValue := code, fieldValue
				if attempt("crm.item.update/"+code, func() error {
					payload := map[string]any{
						"entityTypeId": 2,
						"id":           dealID,
						"fields":       map[string]any{code: fieldValue},
					}
					_, err := jp.postJSON(ctx, "crm.item.update", payload)
					return err
				}) {
					return nil
				}
				if attempt("crm.item.update/"+code+"/orig", func() error {
					payload := map[string]any{
						"entityTypeId": 2,
						"id":           dealID,
						"fields":       map[string]any{code: fieldValue},
						"params":       map[string]any{"useOriginalUfNames": "Y"},
					}
					_, err := jp.postJSON(ctx, "crm.item.update", payload)
					return err
				}) {
					return nil
				}
			}
		}

		for _, code := range fieldCodeVariants(fieldCode) {
			for _, fieldValue := range fileFieldValueVariants(fileName, b64) {
				code, fieldValue := code, fieldValue
				if attempt("crm.deal.update.json/"+code, func() error {
					payload := map[string]any{
						"id":     dealID,
						"fields": map[string]any{code: fieldValue},
					}
					_, err := jp.postJSON(ctx, "crm.deal.update", payload)
					return err
				}) {
					return nil
				}
			}
		}
	}

	if attempt("crm.deal.update.form", func() error {
		return setDealFileFieldForm(ctx, poster, dealID, fieldCode, fileName, b64, beforeID)
	}) {
		return nil
	}

	return fmt.Errorf("не удалось записать файл: %s", strings.Join(errs, "; "))
}

func fileFieldValueVariants(fileName, b64 string) []any {
	single := map[string]any{"fileData": []any{fileName, b64}}
	return []any{
		single,
		[]any{single},
		[]any{fileName, b64},
	}
}

func readDealFileFieldID(ctx context.Context, poster bitrixRESTPoster, dealID int, fieldCode string) int {
	deal, err := fetchDealFields(ctx, poster, dealID)
	if err != nil {
		return 0
	}
	for _, code := range fieldCodeVariants(fieldCode) {
		if id := bitrixFileFieldID(deal[code]); id > 0 {
			return id
		}
	}
	return 0
}

func dealFileFieldUpdated(ctx context.Context, poster bitrixRESTPoster, dealID int, fieldCode string, beforeID int) bool {
	afterID := readDealFileFieldID(ctx, poster, dealID, fieldCode)
	if afterID <= 0 {
		return false
	}
	if beforeID <= 0 {
		return true
	}
	return afterID != beforeID
}

func fetchDealFields(ctx context.Context, poster bitrixRESTPoster, dealID int) (map[string]any, error) {
	form := url.Values{}
	form.Set("id", strconv.Itoa(dealID))
	raw, err := poster.postForm(ctx, "crm.deal.get", form)
	if err != nil {
		return nil, err
	}
	var deal map[string]any
	if err := json.Unmarshal(raw, &deal); err != nil {
		return nil, err
	}
	return deal, nil
}

func bitrixFileFieldID(v any) int {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case map[string]any:
		if id, ok := t["id"]; ok {
			return bitrixIDToInt(id)
		}
	case []any:
		for _, item := range t {
			if id := bitrixFileFieldID(item); id > 0 {
				return id
			}
		}
	case float64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	}
	return 0
}

func bitrixIDToInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	default:
		n, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(v)))
		return n
	}
}

func bitrixFileFieldHasValue(v any) bool {
	if v == nil {
		return false
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		return s != "" && s != "0"
	case float64:
		return t > 0
	case []any:
		for _, item := range t {
			if bitrixFileFieldHasValue(item) {
				return true
			}
		}
		return false
	case map[string]any:
		if id, ok := t["id"]; ok && bitrixFileFieldHasValue(id) {
			return true
		}
		if _, ok := t["downloadUrl"]; ok {
			return true
		}
		if _, ok := t["showUrl"]; ok {
			return true
		}
		if fd, ok := t["fileData"]; ok && bitrixFileFieldHasValue(fd) {
			return true
		}
		return len(t) > 0
	default:
		return fmt.Sprint(v) != "" && fmt.Sprint(v) != "0"
	}
}

func fieldCodeVariants(code string) []string {
	code = strings.TrimSpace(code)
	out := []string{code}
	if strings.HasPrefix(code, "UF_CRM_") {
		suffix := strings.TrimPrefix(code, "UF_CRM_")
		camel := "ufCrm" + suffix
		out = append(out, camel)
	}
	return out
}

func setDealFileFieldForm(ctx context.Context, poster bitrixRESTPoster, dealID int, fieldCode, fileName, b64 string, beforeID int) error {
	attempts := []url.Values{
		func() url.Values {
			f := url.Values{}
			f.Set("id", strconv.Itoa(dealID))
			f.Set("fields["+fieldCode+"][fileData][0]", fileName)
			f.Set("fields["+fieldCode+"][fileData][1]", b64)
			return f
		}(),
		func() url.Values {
			f := url.Values{}
			f.Set("id", strconv.Itoa(dealID))
			f.Set("fields["+fieldCode+"][0][fileData][0]", fileName)
			f.Set("fields["+fieldCode+"][0][fileData][1]", b64)
			return f
		}(),
		func() url.Values {
			f := url.Values{}
			f.Set("id", strconv.Itoa(dealID))
			f.Set("fields["+fieldCode+"][0]", fileName)
			f.Set("fields["+fieldCode+"][1]", b64)
			return f
		}(),
		func() url.Values {
			f := url.Values{}
			f.Set("id", strconv.Itoa(dealID))
			f.Set("fields[FILE_CONTENT_"+fieldCode+"][fileData][0]", fileName)
			f.Set("fields[FILE_CONTENT_"+fieldCode+"][fileData][1]", b64)
			return f
		}(),
	}

	var last error
	for _, form := range attempts {
		_, err := poster.postForm(ctx, "crm.deal.update", form)
		if err == nil {
			if dealFileFieldUpdated(ctx, poster, dealID, fieldCode, beforeID) {
				return nil
			}
			last = fmt.Errorf("crm.deal.update: файл в поле не обновился")
			continue
		}
		last = err
	}
	if last != nil {
		return last
	}
	return fmt.Errorf("crm.deal.update failed")
}

func addDealTimelineNotice(ctx context.Context, poster bitrixRESTPoster, dealID int, notice string) error {
	form := url.Values{}
	form.Set("fields[ENTITY_ID]", strconv.Itoa(dealID))
	form.Set("fields[ENTITY_TYPE]", "deal")
	form.Set("fields[COMMENT]", notice)
	_, err := poster.postForm(ctx, "crm.timeline.comment.add", form)
	if err != nil {
		return fmt.Errorf("crm.timeline.comment.add: %w", err)
	}
	return nil
}
