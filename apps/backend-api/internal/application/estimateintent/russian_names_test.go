package estimateintent

import (
	"strings"
	"testing"
)

func TestEnsureRussianName_TRC(t *testing.T) {
	got := ensureRussianName("TRC-Кристалл")
	if !strings.Contains(got, "ТРЦ") {
		t.Fatalf("got %q", got)
	}
	if containsLatinLetters(got) {
		t.Fatalf("latin left: %q", got)
	}
}

func TestEnsureRussianName_Works(t *testing.T) {
	got := ensureRussianName("устройство кровли и гидроизоляция")
	if got != "устройство кровли и гидроизоляция" {
		t.Fatalf("got %q", got)
	}
}

func TestFinalizeEstimate_RussianProject(t *testing.T) {
	e := FinalizeEstimate(Estimate{
		ProjectName:       "TRC-Premier",
		ObjectDescription: "roof installation",
	})
	if containsLatinLetters(e.ProjectName) {
		t.Fatalf("projectName: %q", e.ProjectName)
	}
	if !strings.Contains(e.ProjectName, "ТРЦ") {
		t.Fatalf("projectName: %q", e.ProjectName)
	}
}
