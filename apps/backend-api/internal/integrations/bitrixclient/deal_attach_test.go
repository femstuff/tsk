package bitrixclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveDealEstimateFieldCode(t *testing.T) {
	fields := map[string]any{
		"UF_CRM_SMETA": map[string]any{
			"TYPE":            "file",
			"EDIT_FORM_LABEL": map[string]any{"ru": "Смета"},
		},
		"UF_CRM_OTHER": map[string]any{
			"TYPE":            "string",
			"EDIT_FORM_LABEL": "Комментарий",
		},
	}
	raw, _ := json.Marshal(fields)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"result": fields})
	}))
	defer srv.Close()

	poster := webhookREST{webhookURL: srv.URL + "/rest/1/test", httpClient: srv.Client()}
	code, err := resolveDealEstimateFieldCode(context.Background(), poster, "")
	if err != nil {
		t.Fatal(err)
	}
	if code != "UF_CRM_SMETA" {
		t.Fatalf("got %q", code)
	}
	_ = raw
}

func TestAttachEstimateToDealUpdatesFieldAndComment(t *testing.T) {
	var calls []string
	fileID := 0
	const fieldCode = "UF_CRM_100"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		if strings.Contains(r.URL.Path, "crm.deal.fields") || strings.Contains(r.URL.Path, "crm.deal.userfield.list") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{
					fieldCode: map[string]any{
						"TYPE":            "file",
						"EDIT_FORM_LABEL": "Смета",
					},
				},
			})
			return
		}
		if strings.Contains(r.URL.Path, "crm.deal.get") {
			deal := map[string]any{"ID": 42}
			if fileID > 0 {
				deal[fieldCode] = map[string]any{"id": fileID, "downloadUrl": "https://example/file"}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": deal})
			return
		}
		if strings.Contains(r.URL.Path, "crm.deal.update") || strings.Contains(r.URL.Path, "crm.item.update") {
			fileID++
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"result": true})
	}))
	defer srv.Close()

	c := New(srv.URL+"/rest/1/test", srv.Client())
	err := c.AttachEstimateToDeal(context.Background(), 42, "smeta.docx", []byte("doc"), AttachEstimateParams{
		FieldCode: fieldCode,
		Notice:    "Смета добавлена для ТРЦ, на проверку.",
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(calls, " ")
	if !strings.Contains(joined, "crm.deal.update") && !strings.Contains(joined, "crm.item.update") {
		t.Fatalf("expected deal update, calls: %v", calls)
	}
	if !strings.Contains(joined, "crm.deal.get") {
		t.Fatalf("expected deal verification get, calls: %v", calls)
	}
	if !strings.Contains(joined, "crm.timeline.comment.add") {
		t.Fatalf("expected timeline comment, calls: %v", calls)
	}
}
