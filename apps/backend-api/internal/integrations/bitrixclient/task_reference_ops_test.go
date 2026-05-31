package bitrixclient

import "testing"

func TestFormatEntityReference(t *testing.T) {
	t.Parallel()

	if got := formatEntityReference("Сделка", "ТРЦ Премьер", "11", "D_11"); got != "Сделка: ТРЦ Премьер (#11)" {
		t.Fatalf("got %q", got)
	}
	if got := formatEntityReference("Сделка", "", "11", "D_11"); got != "Сделка #11" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveCrmTaskLinkLabelPrefixOrder(t *testing.T) {
	t.Parallel()

	if got := resolveCrmTaskLinkLabel(nil, nil, "CO_5"); got != "Компания #5" {
		t.Fatalf("got %q", got)
	}
}
