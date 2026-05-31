package bitrixclient

import (
	"encoding/json"
	"testing"
)

func TestCollectDealFieldsSkipsEmptyAndResolvesStage(t *testing.T) {
	t.Parallel()

	row := map[string]any{
		"ID":          "12",
		"TITLE":       "Поставка оборудования",
		"STAGE_ID":    "NEW",
		"OPPORTUNITY": "150000.00",
		"CURRENCY_ID": "RUB",
		"COMPANY_ID":  "0",
		"CONTACT_ID":  "0",
		"COMMENTS":    "Важная сделка",
		"UF_CRM_123":  "Значение поля",
		"OPENED":      "Y",
	}
	defs := dealFieldDefinitions{
		labels: dealStandardFieldLabels,
		meta:   map[string]dealFieldMeta{},
		enums:  map[string]map[string]string{},
	}
	fields := collectDealFields(row, row, defs, dealFieldResolvers{
		stageLabel: "Новая",
	})
	if len(fields) == 0 {
		t.Fatal("expected fields")
	}
	foundStage := false
	foundUF := false
	for _, field := range fields {
		if field.Key == "STAGE_ID" && field.Value != "Новая" {
			t.Fatalf("stage = %+v", field)
		}
		if field.Key == "STAGE_ID" {
			foundStage = true
		}
		if field.Key == "UF_CRM_123" {
			foundUF = true
		}
		if field.Key == "COMPANY_ID" {
			t.Fatal("company 0 should be skipped")
		}
	}
	if !foundStage || !foundUF {
		t.Fatalf("fields = %+v", fields)
	}
}

func TestFormatDealScalar(t *testing.T) {
	t.Parallel()

	if got := formatDealScalar("Y"); got != "Да" {
		t.Fatalf("got %q", got)
	}
	if got := formatDealScalar("0"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestParseStatusListItems(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`[{"STATUS_ID":"NEW","NAME":"Новая","SORT":"10","CATEGORY_ID":"0"}]`)
	items, err := parseStatusListItems(raw)
	if err != nil {
		t.Fatal(err)
	}
	options := mapStatusItemsToStageOptions(items, 0)
	if len(options) != 1 || options[0].ID != "NEW" {
		t.Fatalf("options = %+v", options)
	}
}

func TestMapRowToDealBriefPreservesCategoryZero(t *testing.T) {
	t.Parallel()

	brief := mapRowToDealBrief(map[string]any{
		"ID":          "1",
		"CATEGORY_ID": "0",
	})
	if brief.CategoryID != "0" {
		t.Fatalf("categoryId = %q", brief.CategoryID)
	}
}
