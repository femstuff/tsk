package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient: httpClient,
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != ""
}

type instantResult struct {
	Value float64
	Found bool
}

type seriesPoint struct {
	Timestamp int64
	Value     float64
}

func (c *Client) QueryInstant(ctx context.Context, query string) (instantResult, error) {
	if !c.Enabled() {
		return instantResult{}, fmt.Errorf("prometheus is not configured")
	}

	endpoint := c.baseURL + "/api/v1/query?" + url.Values{"query": {query}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return instantResult{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return instantResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return instantResult{}, err
	}
	if resp.StatusCode >= 300 {
		return instantResult{}, fmt.Errorf("prometheus query HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var envelope struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Value []any `json:"value"`
			} `json:"result"`
		} `json:"data"`
		ErrorType string `json:"errorType"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return instantResult{}, err
	}
	if envelope.Status != "success" {
		return instantResult{}, fmt.Errorf("prometheus error: %s %s", envelope.ErrorType, envelope.Error)
	}
	if len(envelope.Data.Result) == 0 || len(envelope.Data.Result[0].Value) < 2 {
		return instantResult{}, nil
	}

	value, err := parseSampleValue(envelope.Data.Result[0].Value[1])
	if err != nil {
		return instantResult{}, err
	}
	return instantResult{Value: value, Found: true}, nil
}

func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]seriesPoint, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("prometheus is not configured")
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("start", fmt.Sprintf("%d", start.Unix()))
	params.Set("end", fmt.Sprintf("%d", end.Unix()))
	params.Set("step", fmt.Sprintf("%d", int(step.Seconds())))

	endpoint := c.baseURL + "/api/v1/query_range?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("prometheus range HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var envelope struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Values [][]any `json:"values"`
			} `json:"result"`
		} `json:"data"`
		ErrorType string `json:"errorType"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if envelope.Status != "success" {
		return nil, fmt.Errorf("prometheus error: %s %s", envelope.ErrorType, envelope.Error)
	}
	if len(envelope.Data.Result) == 0 {
		return nil, nil
	}

	points := make([]seriesPoint, 0, len(envelope.Data.Result[0].Values))
	for _, sample := range envelope.Data.Result[0].Values {
		if len(sample) < 2 {
			continue
		}
		ts, err := parseSampleTimestamp(sample[0])
		if err != nil {
			continue
		}
		value, err := parseSampleValue(sample[1])
		if err != nil {
			continue
		}
		points = append(points, seriesPoint{Timestamp: ts, Value: value})
	}
	return points, nil
}

func parseSampleTimestamp(raw any) (int64, error) {
	switch v := raw.(type) {
	case float64:
		return int64(v), nil
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, err
		}
		return int64(parsed), nil
	default:
		return 0, fmt.Errorf("unexpected timestamp type %T", raw)
	}
}

func parseSampleValue(raw any) (float64, error) {
	switch v := raw.(type) {
	case float64:
		return v, nil
	case string:
		if v == "NaN" || v == "+Inf" || v == "-Inf" {
			return 0, fmt.Errorf("non-finite value %q", v)
		}
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("unexpected sample value type %T", raw)
	}
}

type DashboardSnapshot struct {
	Available         bool      `json:"available"`
	RPS               *float64  `json:"rps,omitempty"`
	ErrorRatePercent  *float64  `json:"errorRatePercent,omitempty"`
	LatencyP95Seconds *float64  `json:"latencyP95Seconds,omitempty"`
	UptimeSeconds     *float64  `json:"uptimeSeconds,omitempty"`
	MemoryBytes       *float64  `json:"memoryBytes,omitempty"`
	CPUCores          *float64  `json:"cpuCores,omitempty"`
	HTTPRateSeries    []Point   `json:"httpRateSeries"`
	JobStatusSeries   []Point   `json:"jobStatusSeries"`
	QueriedAt         time.Time `json:"queriedAt"`
}

type Point struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

func (c *Client) DashboardSnapshot(ctx context.Context) (DashboardSnapshot, error) {
	snapshot := DashboardSnapshot{
		Available: c.Enabled(),
		QueriedAt: time.Now().UTC(),
	}
	if !c.Enabled() {
		return snapshot, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	probe, err := c.QueryInstant(ctx, `up{job="backend-api"}`)
	if err != nil || !probe.Found {
		snapshot.Available = false
		return snapshot, nil
	}
	snapshot.Available = true

	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour)
	step := 10 * time.Minute

	if rps, err := c.QueryInstant(ctx, `sum(rate(tsk_business_http_requests_total{job="backend-api"}[5m]))`); err == nil && rps.Found {
		snapshot.RPS = &rps.Value
	}
	if errRate, err := c.QueryInstant(ctx, `100 * sum(rate(tsk_errors_total{job="backend-api"}[5m])) / clamp_min(sum(rate(tsk_http_requests_total{job="backend-api"}[5m])), 0.001)`); err == nil && errRate.Found {
		snapshot.ErrorRatePercent = &errRate.Value
	}
	if p95, err := c.QueryInstant(ctx, `histogram_quantile(0.95, sum(rate(tsk_http_request_duration_seconds_bucket{job="backend-api"}[5m])) by (le))`); err == nil && p95.Found && p95.Value > 0 {
		snapshot.LatencyP95Seconds = &p95.Value
	} else if p95, err := c.QueryInstant(ctx, `histogram_quantile(0.95, sum(rate(tsk_document_job_processing_duration_seconds_bucket{job="backend-api"}[5m])) by (le))`); err == nil && p95.Found && p95.Value > 0 {
		snapshot.LatencyP95Seconds = &p95.Value
	}
	if uptime, err := c.QueryInstant(ctx, `max(tsk_backend_uptime_seconds{job="backend-api"})`); err == nil && uptime.Found {
		snapshot.UptimeSeconds = &uptime.Value
	}
	if memory, err := c.QueryInstant(ctx, `max(process_resident_memory_bytes{job="backend-api"})`); err == nil && memory.Found {
		snapshot.MemoryBytes = &memory.Value
	}
	if cpu, err := c.QueryInstant(ctx, `max(rate(process_cpu_seconds_total{job="backend-api"}[5m]))`); err == nil && cpu.Found {
		snapshot.CPUCores = &cpu.Value
	}

	if series, err := c.QueryRange(ctx, `sum(rate(tsk_business_http_requests_total{job="backend-api"}[5m]))`, start, now, step); err == nil {
		for _, point := range series {
			snapshot.HTTPRateSeries = append(snapshot.HTTPRateSeries, Point{
				Label: time.Unix(point.Timestamp, 0).In(time.Local).Format("15:04"),
				Value: point.Value,
			})
		}
	}

	statuses := []struct {
		status string
		query  string
	}{
		{"completed", `tsk_document_jobs_by_status{job="backend-api",status="completed"}`},
		{"running", `tsk_document_jobs_by_status{job="backend-api",status="running"}`},
		{"queued", `tsk_document_jobs_by_status{job="backend-api",status="queued"}`},
		{"failed", `tsk_document_jobs_by_status{job="backend-api",status="failed"}`},
		{"cancelled", `tsk_document_jobs_by_status{job="backend-api",status="cancelled"}`},
	}
	for _, item := range statuses {
		if value, err := c.QueryInstant(ctx, item.query); err == nil && value.Found {
			snapshot.JobStatusSeries = append(snapshot.JobStatusSeries, Point{
				Label: item.status,
				Value: value.Value,
			})
		}
	}

	return snapshot, nil
}
