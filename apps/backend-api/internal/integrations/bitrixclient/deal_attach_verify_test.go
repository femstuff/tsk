package bitrixclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetDealFileFieldRejectsStaleFile(t *testing.T) {
	const fieldCode = "UF_CRM_100"
	staleID := 163

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "crm.deal.get") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{
					"ID":      42,
					fieldCode: map[string]any{"id": staleID},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"result": true})
	}))
	defer srv.Close()

	poster := webhookREST{webhookURL: srv.URL + "/rest/1/test", httpClient: srv.Client()}
	err := setDealFileField(context.Background(), poster, 42, fieldCode, "smeta.docx", []byte("payload"))
	if err == nil {
		t.Fatal("expected error when file id did not change")
	}
	if !strings.Contains(err.Error(), "не обновился") {
		t.Fatalf("unexpected error: %v", err)
	}
}
