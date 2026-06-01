package estimateintent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMergeEstimatesPrefersLLMForEmptyAndBadMoney(t *testing.T) {
	base := Estimate{EstimatedCost: "1", ProjectName: "", LaborCosts: ""}
	llm := Estimate{EstimatedCost: "1250000", ProjectName: "TRC-Кристалл", LaborCosts: "320000"}
	got := FinalizeEstimate(mergeEstimates(base, llm))
	if got.EstimatedCost != "1250000" {
		t.Fatalf("estimatedCost: %q", got.EstimatedCost)
	}
	if got.ProjectName != "ТРЦ-Кристалл" {
		t.Fatalf("projectName: %q", got.ProjectName)
	}
	if got.LaborCosts != "320000" {
		t.Fatalf("laborCosts: %q", got.LaborCosts)
	}
}

func TestLLMEnricherHTTP(t *testing.T) {
	dto := llmEstimateDTO{
		ProjectName: "ТРЦ-Кристалл",
		LaborCosts:  "320000",
		GrandTotal:  "1250000",
	}
	raw, _ := json.Marshal(dto)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(llmChatResponse{
			Choices: []struct {
				Message llmMessage `json:"message"`
			}{{Message: llmMessage{Content: string(raw)}}},
		})
	}))
	defer srv.Close()

	e := NewLLMEnricher(srv.URL, "test-key", "test-model", srv.Client())
	base := Estimate{RawTranscript: "test", LaborCosts: ""}
	got, err := e.Enrich(context.Background(), "транскрипт", base)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProjectName != "ТРЦ-Кристалл" {
		t.Fatalf("projectName: %q", got.ProjectName)
	}
	if got.LaborCosts != "320000" {
		t.Fatalf("laborCosts: %q", got.LaborCosts)
	}
}
