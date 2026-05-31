package estimateintent

import (
	"fmt"
	"strings"
)

// RenderForm4 форматирует смету как текстовый документ по форме № 4.
func RenderForm4(estimate Estimate) string {
	var b strings.Builder

	b.WriteString("ЛОКАЛЬНЫЙ СМЕТНЫЙ РАСЧЁТ")
	if n := strings.TrimSpace(estimate.EstimateNumber); n != "" {
		b.WriteString(" № ")
		b.WriteString(n)
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat("=", 80))
	b.WriteString("\n\n")
	b.WriteString("                                                          Форма № 4\n\n")

	b.WriteString(padLine("Наименование стройки:", estimate.ProjectName))
	b.WriteString("\n")
	b.WriteString(padLine("Наименование работ и затрат, наименование объекта:", estimate.ObjectDescription))
	b.WriteString("\n")
	b.WriteString(padLine("Основание: чертежи №", estimate.Basis))
	b.WriteString("\n")
	b.WriteString(padLine("Сметная стоимость", rubles(estimate.EstimatedCost)))
	b.WriteString("\n")
	b.WriteString(padLine("Средства на оплату труда", rubles(estimate.LaborCosts)))
	b.WriteString("\n")
	if d := strings.TrimSpace(estimate.PriceDate); d != "" {
		b.WriteString(fmt.Sprintf("Составлен(а) в текущих (прогнозных) ценах по состоянию на %s\n", d))
	} else {
		b.WriteString("Составлен(а) в текущих (прогнозных) ценах по состоянию на __________\n")
	}
	if a := strings.TrimSpace(estimate.Approver); a != "" {
		b.WriteString(fmt.Sprintf("%s\n", a))
	} else {
		b.WriteString("(должность, подпись, инициалы, фамилия)\n")
	}

	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", 80))
	b.WriteString("\n")
	b.WriteString("№ | Шифр/код | Наименование | Ед.изм. | Кол-во | Базис/ед | Базис/общ | Тек./ед | Тек./общ\n")
	b.WriteString(strings.Repeat("-", 80))
	b.WriteString("\n")

	if len(estimate.LineItems) == 0 {
		b.WriteString("(строки не распознаны из голоса — см. транскрипт ниже)\n")
	} else {
		for _, item := range estimate.LineItems {
			seq := item.Seq
			if seq <= 0 {
				seq = 1
			}
			b.WriteString(fmt.Sprintf(
				"%d | %s | %s | %s | %s | %s | %s | %s | %s\n",
				seq,
				emptyDash(item.Code),
				emptyDash(item.Description),
				emptyDash(item.Unit),
				emptyDash(item.Quantity),
				emptyDash(item.BasePricePerUnit),
				emptyDash(item.BasePriceTotal),
				emptyDash(item.CurrentPricePerUnit),
				emptyDash(item.CurrentPriceTotal),
			))
		}
	}

	b.WriteString(strings.Repeat("-", 80))
	b.WriteString("\n")
	b.WriteString(padLine("Итого прямые затраты по смете:", rubles(estimate.TotalDirectCosts)))
	b.WriteString("\n")
	b.WriteString(padLine("ВСЕГО ПО СМЕТЕ:", rubles(estimate.GrandTotal)))
	b.WriteString("\n\n")

	if raw := strings.TrimSpace(estimate.RawTranscript); raw != "" {
		b.WriteString("ТРАНСКРИПТ ГОЛОСОВОЙ ЗАПИСИ\n")
		b.WriteString(strings.Repeat("-", 40))
		b.WriteString("\n")
		b.WriteString(raw)
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String())
}

func padLine(label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "___________"
	}
	return fmt.Sprintf("%s %s", label, value)
}

func rubles(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "___________ руб."
	}
	if strings.Contains(strings.ToLower(value), "руб") {
		return value
	}
	return value + " руб."
}

func emptyDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "—"
	}
	return value
}
