package estimateintent

import (
	"strings"
	"testing"
)

func TestFillTemplate(t *testing.T) {
	t.Parallel()

	tpl := `Смета № {{estimateNumber}}
Стройка: {{projectName}}
{{lineItems}}
Итого: {{grandTotal}}`
	est := Estimate{
		EstimateNumber: "7",
		ProjectName:    "ТРЦ Кристалл",
		GrandTotal:     "1250000",
		LineItems: []LineItem{{
			Seq:         1,
			Code:        "06-01",
			Description: "Кровля",
			Unit:        "м2",
			Quantity:    "100",
		}},
		RawTranscript: "тест",
	}

	out := FillTemplate(tpl, est)
	if !strings.Contains(out, "Смета № 7") {
		t.Fatalf("number: %q", out)
	}
	if !strings.Contains(out, "ТРЦ Кристалл") {
		t.Fatalf("project: %q", out)
	}
	if !strings.Contains(out, "Кровля") {
		t.Fatalf("line item: %q", out)
	}
	if !strings.Contains(out, "1250000") {
		t.Fatalf("total: %q", out)
	}
}
