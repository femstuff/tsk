package estimateintent

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestDefaultEstimateDocxTemplateAndFill(t *testing.T) {
	t.Parallel()

	tpl, err := DefaultEstimateDocxTemplate()
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(tpl), int64(len(tpl)))
	if err != nil {
		t.Fatal(err)
	}
	if len(zr.File) < 2 {
		t.Fatal("expected docx zip entries")
	}

	est := Estimate{
		EstimateNumber:   "12",
		ProjectName:      "ТРЦ Кристалл",
		ObjectDescription: "Кровля",
		EstimatedCost:    "1250000",
		GrandTotal:       "1250000",
		LineItems: []LineItem{{
			Seq: 1, Description: "Монтаж кровли", Quantity: "100", Unit: "м2",
		}},
	}
	out, err := FillDocx(tpl, est)
	if err != nil {
		t.Fatal(err)
	}
	zr2, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatal(err)
	}
	var doc string
	for _, f := range zr2.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		doc = string(raw)
	}
	if !strings.Contains(doc, "ТРЦ Кристалл") {
		t.Fatalf("project not in doc: %s", doc)
	}
	if strings.Contains(doc, "{{projectName}}") {
		t.Fatal("placeholder not replaced")
	}
}
