package bitrixclient

import (
	"encoding/json"
	"testing"
)

func TestMapRowToTaskDetail(t *testing.T) {
	t.Parallel()

	raw := `{
		"task": {
			"id": "8017",
			"title": "Task Example",
			"description": "Task description",
			"status": "3",
			"priority": "2",
			"createdDate": "2025-10-01T10:00:00+03:00",
			"changedDate": "2025-10-06T17:02:43+02:00",
			"deadline": "2025-10-24T19:00:00+02:00",
			"creator": {
				"id": "503",
				"name": "Maria Johnson",
				"workPosition": "admin"
			},
			"responsible": {
				"id": "547",
				"name": "Maria",
				"workPosition": "Tester"
			},
			"accomplices": [{"id": "12", "name": "Alex"}],
			"ufCrmTask": ["C_627", "D_1723"],
			"favorite": "Y",
			"commentsCount": "4",
			"timeEstimate": "3600"
		}
	}`

	var row map[string]any
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	detail := mapRowToTaskDetail(row)

	if detail.ID != "8017" {
		t.Fatalf("id = %q", detail.ID)
	}
	if detail.Creator.Name != "Maria Johnson" {
		t.Fatalf("creator = %+v", detail.Creator)
	}
	if detail.Responsible.Name != "Maria" {
		t.Fatalf("responsible = %+v", detail.Responsible)
	}
	if len(detail.Accomplices) != 1 || detail.Accomplices[0].Name != "Alex" {
		t.Fatalf("accomplices = %+v", detail.Accomplices)
	}
	if len(detail.CrmLinks) != 2 {
		t.Fatalf("crm = %+v", detail.CrmLinks)
	}
	if !detail.Favorite {
		t.Fatal("expected favorite")
	}
	if detail.CommentsCount != "4" {
		t.Fatalf("comments = %q", detail.CommentsCount)
	}
}
