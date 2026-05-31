package estimateintent

import "testing"

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
		t.Fatalf("expected 2 line items, got %d", len(est.LineItems))
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
