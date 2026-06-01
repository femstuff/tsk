package estimateintent

import (
	"strings"
	"testing"
)

func TestParseHeaderFields(t *testing.T) {
	text := `
		Локальный сметный расчёт номер 12.
		Наименование стройки: Жилой дом на ул. Ленина.
		Наименование работ: устройство кровли.
		Основание чертежи номер АР-01, АР-02.
		Сметная стоимость 1 250 000 рублей.
		Средства на оплату труда 320 000 руб.
		Составлен в текущих ценах по состоянию на 1 квартал 2026 года.
		Составил инженер-сметчик Иванов И.И.
	`
	est := Parse(text)

	if est.EstimateNumber != "12" {
		t.Fatalf("estimateNumber: got %q", est.EstimateNumber)
	}
	if est.ProjectName == "" {
		t.Fatal("projectName empty")
	}
	if est.ObjectDescription == "" {
		t.Fatal("objectDescription empty")
	}
	if est.Basis == "" {
		t.Fatal("basis empty")
	}
	if est.EstimatedCost == "" {
		t.Fatal("estimatedCost empty")
	}
	if est.LaborCosts == "" {
		t.Fatal("laborCosts empty")
	}
	if est.PriceDate == "" {
		t.Fatal("priceDate empty")
	}
	if est.Approver == "" {
		t.Fatal("approver empty")
	}
}

func TestParseLineItems(t *testing.T) {
	text := `
		Позиция 1. Шифр 06-01-001. Наименование монтаж кровли.
		Единица измерения квадратный метр. Количество 500.
		Базисная цена на единицу 1200. Базисная общая 600000.
		Текущая цена на единицу 1500. Текущая общая 750000.
		Позиция 2. Шифр 08-02-003. Наименование гидроизоляция.
		Единица измерения м2. Количество 120.
	`
	est := Parse(text)
	if len(est.LineItems) < 2 {
		t.Fatalf("expected 2 line items, got %d (%+v)", len(est.LineItems), est.LineItems)
	}
	if est.LineItems[0].Code == "" || est.LineItems[0].Description == "" {
		t.Fatalf("first item incomplete: %+v", est.LineItems[0])
	}
}

func TestIsEstimateCategory(t *testing.T) {
	if !IsEstimateCategory("estimate") || !IsEstimateCategory("Смета") {
		t.Fatal("expected estimate categories")
	}
	if IsEstimateCategory("sales") {
		t.Fatal("sales should not be estimate")
	}
}

func TestParseVoiceStyleWithoutPunctuation(t *testing.T) {
	text := `локальный сметный расчет номер 12 наименование стройки трц премьер реконструкция фасада ` +
		`наименование работ устройство кровли гидроизоляция основание чертежи номер ар01 ар02 ` +
		`сметная стоимость миллион 250 тысяч рублей средства на оплату труда 320 тысяч рублей ` +
		`составлен в текущих ценах по состоянию на первый квартал 2026 года составил инженер сметчик иванов и и ` +
		`позиция 1 шифр 0601001 наименование монтаж кровли единица измерения квадратный метр количество 500 ` +
		`базисная цена на единицу 1200 базисная общая 600 тысяч текущая цена на единицу 1500 текущая общая 750 тысяч ` +
		`позиция 2 шифр 0802003 наименование гидроизоляция единица измерения метр квадратный количество 120 ` +
		`итого прямые затраты по смете 980 тысяч рублей всего по смете миллион 250 тысяч рублей`

	est := Parse(text)

	if est.EstimateNumber != "12" {
		t.Fatalf("estimateNumber: got %q", est.EstimateNumber)
	}
	if !strings.Contains(strings.ToLower(est.ProjectName), "трц") && !strings.Contains(strings.ToLower(est.ProjectName), "премьер") {
		t.Fatalf("projectName: got %q", est.ProjectName)
	}
	if est.ObjectDescription == "" || len(est.ObjectDescription) > 120 {
		t.Fatalf("objectDescription: got %q (len=%d)", est.ObjectDescription, len(est.ObjectDescription))
	}
	if est.Basis == "" || strings.Contains(strings.ToLower(est.Basis), "сметная") {
		t.Fatalf("basis: got %q", est.Basis)
	}
	if est.EstimatedCost != "1250000" {
		t.Fatalf("estimatedCost: got %q", est.EstimatedCost)
	}
	if est.LaborCosts != "320000" {
		t.Fatalf("laborCosts: got %q", est.LaborCosts)
	}
	if est.GrandTotal != "1250000" {
		t.Fatalf("grandTotal: got %q", est.GrandTotal)
	}
	if len(est.LineItems) < 2 {
		t.Fatalf("line items: got %d", len(est.LineItems))
	}
	if !strings.Contains(strings.ToLower(est.LineItems[0].Description), "кровл") {
		t.Fatalf("line1 desc: %+v", est.LineItems[0])
	}
}

func TestParseWhisperCrystalTranscript(t *testing.T) {
	text := `Локальный сметный расчет номер 12 на именование строки TRC-Кристалл, на именование работ устройства кровлей гидроизоляция, ` +
		`основание чертежи номер АР01-АР02, сметная стоимость 1,250,000 рублей, средство на опаду труда 320,000 рублей, ` +
		`состав текущих ценах по состоянию на первую кварталу до 1226 года, состав у инженер-смечник Иванов ИИ, всего по смете 1,250,000 рублей.`

	est := Parse(text)

	if est.EstimateNumber != "12" {
		t.Fatalf("estimateNumber: got %q", est.EstimateNumber)
	}
	if !strings.Contains(est.ProjectName, "ТРЦ") {
		t.Fatalf("projectName: got %q", est.ProjectName)
	}
	if containsLatinLetters(est.ProjectName) {
		t.Fatalf("projectName must be cyrillic: %q", est.ProjectName)
	}
	if est.ObjectDescription == "" || strings.Contains(strings.ToLower(est.ObjectDescription), "сметная") {
		t.Fatalf("objectDescription: got %q", est.ObjectDescription)
	}
	if est.EstimatedCost != "1250000" {
		t.Fatalf("estimatedCost: got %q", est.EstimatedCost)
	}
	if est.LaborCosts != "320000" {
		t.Fatalf("laborCosts: got %q", est.LaborCosts)
	}
	if est.GrandTotal != "1250000" {
		t.Fatalf("grandTotal: got %q", est.GrandTotal)
	}
	if strings.Contains(strings.ToLower(est.PriceDate), "инженер") {
		t.Fatalf("priceDate should not include engineer: %q", est.PriceDate)
	}
	if est.Approver == "" || !strings.Contains(strings.ToLower(est.Approver), "иванов") {
		t.Fatalf("approver: got %q", est.Approver)
	}
}

func TestParseWhisperPunctuatedCrystal(t *testing.T) {
	text := `Локальный сметный расчет номер 12. Наименование строки TRC-Кристалл. Наименование работ устройства кровли и гидроизоляция. ` +
		`Основание чертежи номер AP01-AP02. Сметная стоимость 1 миллион 250 тысяч рублей. Средсанапат у труда 320 тысяч рублей. ` +
		`Составлен текущих ценах по состоянию на первой кварталу 2026 года. Составый инженер-сметчик Иванов ИИ. Всего по смете 1 миллион 250 тысяч рублей.`

	est := Parse(text)

	if !strings.Contains(est.ProjectName, "ТРЦ") {
		t.Fatalf("projectName: got %q", est.ProjectName)
	}
	if containsLatinLetters(est.ProjectName) {
		t.Fatalf("projectName must be cyrillic: %q", est.ProjectName)
	}
	if !strings.Contains(strings.ToLower(est.ObjectDescription), "кровл") {
		t.Fatalf("objectDescription: got %q", est.ObjectDescription)
	}
	if est.LaborCosts != "320000" {
		t.Fatalf("laborCosts: got %q", est.LaborCosts)
	}
	if strings.Contains(strings.ToLower(est.PriceDate), "состав") {
		t.Fatalf("priceDate should not include состав*: %q", est.PriceDate)
	}
	if est.Approver == "" || !strings.Contains(strings.ToLower(est.Approver), "иванов") {
		t.Fatalf("approver: got %q", est.Approver)
	}
}

func TestParseMoneyPhrase(t *testing.T) {
	cases := map[string]string{
		"миллион 250 тысяч рублей": "1250000",
		"980 тысяч рублей":         "980000",
		"320 тысяч":                "320000",
		"1 250 000":                "1250000",
		"1,250,000 рублей":         "1250000",
		"320,000":                  "320000",
	}
	for in, want := range cases {
		if got := parseMoneyPhrase(in); got != want {
			t.Fatalf("parseMoneyPhrase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	est := Parse("Наименование стройки: Тест. Позиция 1. Наименование работа.")
	raw, err := MarshalPayload(est)
	if err != nil {
		t.Fatal(err)
	}
	back := UnmarshalPayload(raw)
	if back.ProjectName != est.ProjectName {
		t.Fatalf("roundtrip project: %q vs %q", back.ProjectName, est.ProjectName)
	}
}
