//go:build integration

package bitrixclient

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestLiveAttachEstimateToDeal(t *testing.T) {
	webhook := strings.TrimSpace(os.Getenv("BITRIX_WEBHOOK_URL"))
	field := strings.TrimSpace(os.Getenv("BITRIX_DEAL_ESTIMATE_FIELD"))
	dealIDStr := strings.TrimSpace(os.Getenv("BITRIX_TEST_DEAL_ID"))
	if webhook == "" || field == "" || dealIDStr == "" {
		t.Skip("set BITRIX_WEBHOOK_URL, BITRIX_DEAL_ESTIMATE_FIELD, BITRIX_TEST_DEAL_ID")
	}
	dealID, err := strconv.Atoi(dealIDStr)
	if err != nil || dealID <= 0 {
		t.Fatalf("invalid BITRIX_TEST_DEAL_ID: %q", dealIDStr)
	}

	c := New(webhook, nil)
	err = c.AttachEstimateToDeal(context.Background(), dealID, "tsk-test-smeta.docx", []byte("test smeta attach"), AttachEstimateParams{
		FieldCode: field,
		Notice:    "Тест прикрепления сметы (автотест).",
	})
	if err != nil {
		t.Fatal(err)
	}
}
