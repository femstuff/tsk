package estimateintent

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// DefaultEstimateDocxTemplate возвращает пустой docx-шаблон сметы с плейсхолдерами {{key}}.
func DefaultEstimateDocxTemplate() ([]byte, error) {
	return buildDocxFromParagraphs([]string{
		"СМЕТА НА СТРОИТЕЛЬНО-ОТДЕЛОЧНЫЕ РАБОТЫ",
		"Локальный сметный расчёт № {{estimateNumber}}",
		"Форма № 4",
		"",
		"Наименование стройки: {{projectName}}",
		"Наименование работ и затрат, наименование объекта: {{objectDescription}}",
		"Основание (чертежи, спецификации): {{basis}}",
		"",
		"Сметная стоимость: {{estimatedCost}} руб.",
		"Средства на оплату труда: {{laborCosts}} руб.",
		"Составлен(а) в текущих (прогнозных) ценах по состоянию на {{priceDate}}",
		"Составил: {{approver}}",
		"",
		"Табличная часть:",
		"{{lineItems}}",
		"",
		"Итого прямые затраты по смете: {{totalDirectCosts}} руб.",
		"ВСЕГО ПО СМЕТЕ: {{grandTotal}} руб.",
	})
}

func buildDocxFromParagraphs(lines []string) ([]byte, error) {
	var body strings.Builder
	for _, line := range lines {
		escaped := xmlEscape(line)
		body.WriteString("<w:p><w:r><w:t xml:space=\"preserve\">")
		body.WriteString(escaped)
		body.WriteString("</w:t></w:r></w:p>")
	}
	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body>` + body.String() + `</w:body></w:document>`
	return buildDocxFromDocumentXML(documentXML)
}

func buildDocxFromDocumentXML(documentXML string) ([]byte, error) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/document.xml": documentXML,
	}

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(w, content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func xmlEscape(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	value = strings.ReplaceAll(value, "\"", "&quot;")
	return value
}

// FillDocx подставляет поля сметы в docx-шаблон (плейсхолдеры {{key}} в word/document.xml).
func FillDocx(template []byte, estimate Estimate) ([]byte, error) {
	if len(template) == 0 {
		return nil, fmt.Errorf("docx template is empty")
	}
	replacements := templateReplacements(estimate)

	zr, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, fmt.Errorf("read docx zip: %w", err)
	}

	out := new(bytes.Buffer)
	zw := zip.NewWriter(out)

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, err
		}

		if f.Name == "word/document.xml" {
			text := string(raw)
			for key, value := range replacements {
				text = strings.ReplaceAll(text, "{{"+key+"}}", xmlEscape(value))
			}
			raw = []byte(text)
		}

		hdr := f.FileHeader
		w, err := zw.CreateHeader(&hdr)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(raw); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func templateReplacements(estimate Estimate) map[string]string {
	return map[string]string{
		"estimateNumber":    displayField(estimate.EstimateNumber),
		"projectName":       displayField(estimate.ProjectName),
		"objectDescription": displayField(estimate.ObjectDescription),
		"basis":             displayField(estimate.Basis),
		"estimatedCost":     displayAmount(estimate.EstimatedCost),
		"laborCosts":        displayAmount(estimate.LaborCosts),
		"priceDate":         displayField(estimate.PriceDate),
		"approver":          displayField(estimate.Approver),
		"totalDirectCosts":  displayAmount(estimate.TotalDirectCosts),
		"grandTotal":        displayAmount(estimate.GrandTotal),
		"lineItems":         formatLineItemsBlock(estimate.LineItems),
		"rawTranscript":     strings.TrimSpace(estimate.RawTranscript),
	}
}

// IsDocxTemplate reports docx by mime or extension.
func IsDocxTemplate(fileName, mimeType string) bool {
	lower := strings.ToLower(fileName)
	if strings.HasSuffix(lower, ".docx") {
		return true
	}
	mimeType = strings.ToLower(mimeType)
	return strings.Contains(mimeType, "wordprocessingml") || strings.Contains(mimeType, "officedocument")
}
