package bitrixclient

import (
	"encoding/json"
	"testing"
)

func TestParseTasksListResultNestedTask(t *testing.T) {
	raw := json.RawMessage(`{
		"tasks": [
			{
				"task": {
					"id": "408",
					"title": "Согласование КП",
					"status": "3",
					"deadline": "2026-06-01T12:00:00+03:00"
				}
			}
		]
	}`)

	items, err := parseTasksListResult(raw)
	if err != nil {
		t.Fatalf("parseTasksListResult: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 task, got %d", len(items))
	}
	if items[0].ID != "408" {
		t.Fatalf("expected id 408, got %q", items[0].ID)
	}
	if items[0].Title != "Согласование КП" {
		t.Fatalf("unexpected title %q", items[0].Title)
	}
}
