package estimateintent

import "testing"

func TestFinalizeEstimateFixesCostOne(t *testing.T) {
	e := FinalizeEstimate(Estimate{
		EstimatedCost: "1",
		GrandTotal:    "1250000",
		RawTranscript: "всего по смете 1 миллион 250 тысяч",
	})
	if e.EstimatedCost != "1250000" {
		t.Fatalf("estimatedCost: %q", e.EstimatedCost)
	}
}
