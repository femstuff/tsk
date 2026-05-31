package estimateintent

// LineItem — строка табличной части локальной сметы (форма № 4).
type LineItem struct {
	Seq                 int    `json:"seq"`
	Code                string `json:"code"`
	Description         string `json:"description"`
	Unit                string `json:"unit"`
	Quantity            string `json:"quantity"`
	BasePricePerUnit    string `json:"basePricePerUnit"`
	BasePriceTotal      string `json:"basePriceTotal"`
	CurrentPricePerUnit string `json:"currentPricePerUnit"`
	CurrentPriceTotal   string `json:"currentPriceTotal"`
}

// Estimate — поля шапки и таблицы локального сметного расчёта.
type Estimate struct {
	EstimateNumber    string     `json:"estimateNumber"`
	ProjectName       string     `json:"projectName"`
	ObjectDescription string     `json:"objectDescription"`
	Basis             string     `json:"basis"`
	EstimatedCost     string     `json:"estimatedCost"`
	LaborCosts        string     `json:"laborCosts"`
	PriceDate         string     `json:"priceDate"`
	Approver          string     `json:"approver"`
	LineItems         []LineItem `json:"lineItems"`
	TotalDirectCosts  string     `json:"totalDirectCosts"`
	GrandTotal        string     `json:"grandTotal"`
	RawTranscript     string     `json:"rawTranscript"`
}

// Payload хранится в document_jobs.payload как JSON.
type Payload struct {
	Estimate Estimate `json:"estimate"`
}
