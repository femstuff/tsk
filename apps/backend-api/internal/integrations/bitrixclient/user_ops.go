package bitrixclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type bitrixUserCache struct {
	poster bitrixRESTPoster
	mem    map[string]BitrixTaskPerson
}

func newBitrixUserCache(poster bitrixRESTPoster) *bitrixUserCache {
	return &bitrixUserCache{
		poster: poster,
		mem:    make(map[string]BitrixTaskPerson),
	}
}

func (c *bitrixUserCache) Prefetch(ctx context.Context, ids []string) {
	if c == nil || c.poster == nil {
		return
	}
	need := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || id == "0" {
			continue
		}
		if _, ok := c.mem[id]; ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		need = append(need, id)
	}
	if len(need) == 0 {
		return
	}

	for id, person := range fetchBitrixUsers(ctx, c.poster, need) {
		c.mem[id] = person
	}
	for _, id := range need {
		if _, ok := c.mem[id]; !ok {
			c.mem[id] = BitrixTaskPerson{ID: id}
		}
	}
}

func (c *bitrixUserCache) Resolve(ctx context.Context, userID string) BitrixTaskPerson {
	userID = strings.TrimSpace(userID)
	if userID == "" || userID == "0" {
		return BitrixTaskPerson{}
	}
	if c == nil {
		return BitrixTaskPerson{ID: userID}
	}
	if person, ok := c.mem[userID]; ok {
		return person
	}
	c.Prefetch(ctx, []string{userID})
	if person, ok := c.mem[userID]; ok {
		return person
	}
	return BitrixTaskPerson{ID: userID}
}

func (c *bitrixUserCache) Name(ctx context.Context, userID string) string {
	return c.Resolve(ctx, userID).Name
}

type dealRESTUserPoster struct {
	caller dealRESTCaller
}

func (d dealRESTUserPoster) postForm(ctx context.Context, method string, form url.Values) (json.RawMessage, error) {
	return d.caller.callDealMethodPOST(ctx, method, form)
}

func fetchBitrixUsers(ctx context.Context, poster bitrixRESTPoster, userIDs []string) map[string]BitrixTaskPerson {
	out := make(map[string]BitrixTaskPerson)
	if poster == nil || len(userIDs) == 0 {
		return out
	}

	unique := make([]string, 0, len(userIDs))
	seen := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		id = strings.TrimSpace(id)
		if id == "" || id == "0" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	const chunkSize = 50
	for start := 0; start < len(unique); start += chunkSize {
		end := start + chunkSize
		if end > len(unique) {
			end = len(unique)
		}
		chunk := unique[start:end]
		form := url.Values{}
		for i, id := range chunk {
			form.Set(fmt.Sprintf("filter[ID][%d]", i), id)
		}
		raw, err := poster.postForm(ctx, "user.get", form)
		if err != nil {
			continue
		}
		for _, person := range parseUsersListResult(raw) {
			if person.ID != "" {
				out[person.ID] = person
			}
		}
	}

	for _, id := range unique {
		if _, ok := out[id]; ok {
			continue
		}
		if person, err := fetchBitrixUser(ctx, poster, id); err == nil && person.ID != "" {
			out[person.ID] = person
		}
	}
	return out
}

func fetchBitrixUser(ctx context.Context, poster bitrixRESTPoster, userID string) (BitrixTaskPerson, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" || userID == "0" {
		return BitrixTaskPerson{}, fmt.Errorf("empty user id")
	}
	if poster == nil {
		return BitrixTaskPerson{}, fmt.Errorf("bitrix client is not configured")
	}

	attempts := []url.Values{
		{ "filter[ID]": {userID} },
		{ "ID": {userID} },
	}
	for _, form := range attempts {
		raw, err := poster.postForm(ctx, "user.get", form)
		if err != nil {
			continue
		}
		if person, ok := parseUsersResult(raw, userID); ok {
			return person, nil
		}
	}
	return BitrixTaskPerson{ID: userID}, fmt.Errorf("user %s not found", userID)
}

func parseUsersListResult(raw json.RawMessage) []BitrixTaskPerson {
	raw = bytesTrim(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err == nil {
		out := make([]BitrixTaskPerson, 0, len(rows))
		for _, row := range rows {
			out = append(out, bitrixUserPerson(row, fieldFromRow(row, "ID", "id")))
		}
		return out
	}

	if person, ok := parseUsersResult(raw, ""); ok {
		return []BitrixTaskPerson{person}
	}
	return nil
}

func parseUsersResult(raw json.RawMessage, fallbackID string) (BitrixTaskPerson, bool) {
	raw = bytesTrim(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return BitrixTaskPerson{}, false
	}

	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err == nil {
		if len(rows) == 0 {
			return BitrixTaskPerson{}, false
		}
		return bitrixUserPerson(rows[0], fallbackID), true
	}

	var row map[string]any
	if err := json.Unmarshal(raw, &row); err == nil && len(row) > 0 {
		return bitrixUserPerson(row, fallbackID), true
	}
	return BitrixTaskPerson{}, false
}

func bitrixUserPerson(row map[string]any, fallbackID string) BitrixTaskPerson {
	name := userDisplayNameFromRow(row)
	id := fieldFromRow(row, "ID", "id")
	if id == "" {
		id = fallbackID
	}
	position := strings.TrimSpace(fieldFromRow(row, "WORK_POSITION", "workPosition"))
	return BitrixTaskPerson{ID: id, Name: name, WorkPosition: position}
}

func collectDealUserIDs(row map[string]any, defs dealFieldDefinitions) []string {
	seen := make(map[string]struct{})
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || id == "0" {
			return
		}
		seen[id] = struct{}{}
	}

	for key, value := range row {
		upper := strings.ToUpper(strings.TrimSpace(key))
		meta := defs.meta[upper]
		if isBitrixUserFieldKey(upper, meta.fieldType) {
			add(dealFieldRaw(map[string]any{key: value}))
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out
}

func isBitrixUserFieldKey(key, fieldType string) bool {
	switch strings.ToLower(strings.TrimSpace(fieldType)) {
	case "user", "employee":
		return true
	}
	switch key {
	case "ASSIGNED_BY_ID", "CREATED_BY_ID", "MODIFY_BY_ID", "MOVED_BY_ID", "LAST_ACTIVITY_BY", "CREATED_BY", "MODIFY_BY", "MOVED_BY":
		return true
	}
	return strings.HasSuffix(key, "_BY_ID") || strings.HasSuffix(key, "_BY")
}

func collectTaskUserIDs(detail BitrixTaskDetail) []string {
	seen := make(map[string]struct{})
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || id == "0" {
			return
		}
		seen[id] = struct{}{}
	}
	add(detail.Creator.ID)
	add(detail.Responsible.ID)
	for _, person := range detail.Accomplices {
		add(person.ID)
	}
	for _, person := range detail.Auditors {
		add(person.ID)
	}
	for _, comment := range detail.Comments {
		add(comment.AuthorID)
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out
}

func applyCachedUser(cache *bitrixUserCache, ctx context.Context, person BitrixTaskPerson) BitrixTaskPerson {
	id := strings.TrimSpace(person.ID)
	if id == "" {
		return person
	}
	resolved := cache.Resolve(ctx, id)
	if resolved.Name != "" {
		person.Name = resolved.Name
	}
	if resolved.WorkPosition != "" {
		person.WorkPosition = resolved.WorkPosition
	}
	if person.ID == "" {
		person.ID = id
	}
	return person
}

func enrichTaskPeople(ctx context.Context, poster bitrixRESTPoster, detail BitrixTaskDetail) BitrixTaskDetail {
	cache := newBitrixUserCache(poster)
	cache.Prefetch(ctx, collectTaskUserIDs(detail))

	detail.Creator = applyCachedUser(cache, ctx, detail.Creator)
	detail.Responsible = applyCachedUser(cache, ctx, detail.Responsible)
	for i, person := range detail.Accomplices {
		detail.Accomplices[i] = applyCachedUser(cache, ctx, person)
	}
	for i, person := range detail.Auditors {
		detail.Auditors[i] = applyCachedUser(cache, ctx, person)
	}
	for i, comment := range detail.Comments {
		if strings.TrimSpace(comment.AuthorName) != "" {
			continue
		}
		if name := cache.Name(ctx, comment.AuthorID); name != "" {
			detail.Comments[i].AuthorName = name
		}
	}
	return detail
}
