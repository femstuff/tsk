package estimateintent

import (
	"fmt"
	"strings"
)

// BuildEstimateDocx — готовый docx по форме №4 (заголовок, поля, таблица, итоги).
func BuildEstimateDocx(estimate Estimate) ([]byte, error) {
	var body strings.Builder

	body.WriteString(docxParagraph("СМЕТА НА СТРОИТЕЛЬНО-ОТДЕЛОЧНЫЕ РАБОТЫ", true, "center"))
	numLine := "Локальный сметный расчёт"
	if n := strings.TrimSpace(estimate.EstimateNumber); n != "" {
		numLine += " № " + n
	}
	body.WriteString(docxParagraph(numLine, false, "center"))
	body.WriteString(docxParagraph("Форма № 4", false, "center"))
	body.WriteString(docxParagraph("", false, ""))

	body.WriteString(docxParagraph(docxLabelValue("Наименование стройки:", estimate.ProjectName), false, ""))
	body.WriteString(docxParagraph(docxLabelValue("Наименование работ и затрат, наименование объекта:", estimate.ObjectDescription), false, ""))
	body.WriteString(docxParagraph(docxLabelValue("Основание (чертежи, спецификации):", estimate.Basis), false, ""))
	body.WriteString(docxParagraph("", false, ""))

	body.WriteString(docxParagraphAmountHighlight("Сметная стоимость:", estimate.EstimatedCost))
	body.WriteString(docxParagraph(docxLabelValue("Средства на оплату труда:", formatAmountRub(estimate.LaborCosts)), false, ""))
	priceDate := strings.TrimSpace(estimate.PriceDate)
	if priceDate == "" {
		priceDate = "___________"
	}
	body.WriteString(docxParagraph("Составлен(а) в текущих (прогнозных) ценах по состоянию на "+priceDate, false, ""))
	approver := strings.TrimSpace(estimate.Approver)
	if approver == "" {
		approver = "___________"
	}
	body.WriteString(docxParagraph("Составил: "+approver, false, ""))
	body.WriteString(docxParagraph("", false, ""))

	body.WriteString(buildLineItemsTableXML(estimate.LineItems))
	body.WriteString(docxParagraph("", false, ""))

	body.WriteString(docxParagraph(docxLabelValue("Итого прямые затраты по смете:", formatAmountRub(estimate.TotalDirectCosts)), false, ""))
	body.WriteString(docxParagraphAmountHighlight("ВСЕГО ПО СМЕТЕ:", estimate.GrandTotal))

	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body>` + body.String() + `<w:sectPr/></w:body></w:document>`

	return packDocxZip(documentXML)
}

func docxLabelValue(label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "___________"
	}
	return label + " " + value
}

func formatAmountRub(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "___________ руб."
	}
	digits := compactDigits(value)
	if digits == "" {
		return value
	}
	return formatThousands(digits) + " руб."
}

func formatThousands(digits string) string {
	if len(digits) <= 3 {
		return digits
	}
	var parts []string
	for len(digits) > 3 {
		parts = append([]string{digits[len(digits)-3:]}, parts...)
		digits = digits[:len(digits)-3]
	}
	if digits != "" {
		parts = append([]string{digits}, parts...)
	}
	return strings.Join(parts, " ")
}

func docxParagraphAmountHighlight(label, rawAmount string) string {
	label = xmlEscape(strings.TrimSpace(label))
	amount := xmlEscape(formatAmountRub(rawAmount))
	return fmt.Sprintf(
		`<w:p><w:r><w:t xml:space="preserve">%s </w:t></w:r>`+
			`<w:r><w:rPr><w:color w:val="0070C0"/><w:u w:val="single"/><w:b/></w:rPr>`+
			`<w:t xml:space="preserve">%s</w:t></w:r></w:p>`,
		label, amount,
	)
}

func docxParagraph(text string, bold bool, align string) string {
	text = xmlEscape(text)
	pPr := ""
	if align != "" {
		pPr = fmt.Sprintf(`<w:pPr><w:jc w:val="%s"/></w:pPr>`, align)
	}
	rPr := ""
	if bold {
		rPr = `<w:rPr><w:b/></w:rPr>`
	}
	return fmt.Sprintf(`<w:p>%s<w:r>%s<w:t xml:space="preserve">%s</w:t></w:r></w:p>`, pPr, rPr, text)
}

func buildLineItemsTableXML(items []LineItem) string {
	headers := []string{"№", "Шифр", "Наименование", "Ед.", "Кол-во", "Базис/ед", "Базис/общ", "Текущ/ед", "Текущ/общ"}
	var rows [][]string
	rows = append(rows, headers)
	if len(items) == 0 {
		rows = append(rows, []string{"1", "—", "(позиции не распознаны)", "—", "—", "—", "—", "—", "—"})
	} else {
		for _, item := range items {
			seq := item.Seq
			if seq <= 0 {
				seq = 1
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", seq),
				displayField(item.Code),
				displayField(item.Description),
				displayField(item.Unit),
				displayAmount(item.Quantity),
				displayAmount(item.BasePricePerUnit),
				displayAmount(item.BasePriceTotal),
				displayAmount(item.CurrentPricePerUnit),
				displayAmount(item.CurrentPriceTotal),
			})
		}
	}

	colWidths := []int{600, 1100, 3600, 900, 900, 1100, 1100, 1100, 1100}

	var b strings.Builder
	b.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="5000" w:type="pct"/><w:tblBorders>`)
	for _, edge := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
		b.WriteString(fmt.Sprintf(`<w:%s w:val="single" w:sz="4" w:space="0" w:color="000000"/>`, edge))
	}
	b.WriteString(`</w:tblBorders></w:tblPr>`)

	for ri, row := range rows {
		b.WriteString(`<w:tr>`)
		for ci, cell := range row {
			isHeader := ri == 0
			width := 1200
			if ci < len(colWidths) {
				width = colWidths[ci]
			}
			b.WriteString(docxTableCell(cell, isHeader, width))
		}
		b.WriteString(`</w:tr>`)
	}
	b.WriteString(`</w:tbl>`)
	return b.String()
}

func docxTableCell(text string, header bool, widthDXA int) string {
	text = xmlEscape(strings.TrimSpace(text))
	if text == "" {
		text = "—"
	}
	rPr := ""
	pPr := `<w:pPr><w:jc w:val="left"/></w:pPr>`
	if header {
		rPr = `<w:rPr><w:b/></w:rPr>`
		pPr = `<w:pPr><w:jc w:val="center"/></w:pPr>`
	}
	return fmt.Sprintf(
		`<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/></w:tcPr><w:p>%s<w:r>%s<w:t xml:space="preserve">%s</w:t></w:r></w:p></w:tc>`,
		widthDXA, pPr, rPr, text,
	)
}

func packDocxZip(documentXML string) ([]byte, error) {
	return buildDocxFromDocumentXML(documentXML)
}
