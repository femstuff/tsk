package metrics

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	domain "tsk/backend-api/internal/domain/documentjob"
)

type Collector struct {
	startedAt             time.Time
	totalRequests         atomic.Uint64
	totalBusinessRequests atomic.Uint64
	totalJobs             atomic.Uint64
	totalErrors           atomic.Uint64

	registry             *prometheus.Registry
	httpRequests         *prometheus.CounterVec
	businessRequests     *prometheus.CounterVec
	jobStatusGauge       *prometheus.GaugeVec
	jobsTotalGauge       prometheus.Gauge
		jobProcessingLatency *prometheus.HistogramVec
		httpRequestDuration  *prometheus.HistogramVec
		jobCreated           prometheus.Counter
	errors               *prometheus.CounterVec
	uptime               prometheus.GaugeFunc
}

func NewCollector() *Collector {
	collector := &Collector{
		startedAt: time.Now(),
		registry:  prometheus.NewRegistry(),
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tsk_http_requests_total",
			Help: "Total HTTP requests processed by the backend API.",
		}, []string{"method", "path", "status"}),
		businessRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tsk_business_http_requests_total",
			Help: "Business-facing HTTP requests excluding infrastructure noise.",
		}, []string{"method", "path", "status"}),
		jobStatusGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tsk_document_jobs_by_status",
			Help: "Current number of document jobs by status.",
		}, []string{"status"}),
		jobsTotalGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tsk_document_jobs_total",
			Help: "Total number of document jobs in the database.",
		}),
		jobProcessingLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "tsk_document_job_processing_duration_seconds",
			Help:    "Processing duration of document jobs.",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		}, []string{"status"}),
		httpRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "tsk_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30},
		}, []string{"method", "path"}),
		jobCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tsk_document_jobs_created_total",
			Help: "Total number of document jobs created.",
		}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "tsk_errors_total",
			Help: "Total backend errors by kind.",
		}, []string{"kind"}),
	}

	collector.uptime = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "tsk_backend_uptime_seconds",
		Help: "Backend API uptime in seconds.",
	}, func() float64 {
		return collector.Uptime().Seconds()
	})

	collector.registry.MustRegister(
		collector.httpRequests,
		collector.businessRequests,
		collector.jobStatusGauge,
		collector.jobsTotalGauge,
		collector.jobProcessingLatency,
		collector.httpRequestDuration,
		collector.jobCreated,
		collector.errors,
		collector.uptime,
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	collector.jobCreated.Add(0)
	for _, status := range domain.ValidStatuses() {
		collector.jobStatusGauge.WithLabelValues(string(status)).Set(0)
	}
	for _, kind := range []string{"http", "storage", "database", "bitrix", "task_command", "job_processing"} {
		collector.errors.WithLabelValues(kind).Add(0)
	}

	return collector
}

func (c *Collector) RecordHTTPRequest(method string, path string, status int, duration time.Duration, business bool) {
	c.totalRequests.Add(1)
	c.httpRequests.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
	if business {
		c.totalBusinessRequests.Add(1)
		c.businessRequests.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
	}
	c.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())

	if status >= http.StatusInternalServerError {
		c.RecordError("http")
	}
}

func (c *Collector) RecordJobCreated() {
	c.totalJobs.Add(1)
	c.jobCreated.Inc()
}

func (c *Collector) RecordJobProcessed(status domain.Status, duration time.Duration) {
	c.jobProcessingLatency.WithLabelValues(string(status)).Observe(duration.Seconds())
}

func (c *Collector) RecordError(kind string) {
	c.totalErrors.Add(1)
	c.errors.WithLabelValues(kind).Inc()
}

func (c *Collector) SyncJobStatusCounts(counts map[domain.Status]int) {
	total := 0
	for _, status := range domain.ValidStatuses() {
		count := counts[status]
		c.jobStatusGauge.WithLabelValues(string(status)).Set(float64(count))
		total += count
	}
	c.jobsTotalGauge.Set(float64(total))
}

func (c *Collector) Uptime() time.Duration {
	return time.Since(c.startedAt)
}

func (c *Collector) Requests() uint64 {
	return c.totalRequests.Load()
}

func (c *Collector) BusinessRequests() uint64 {
	return c.totalBusinessRequests.Load()
}

func (c *Collector) JobsCreated() uint64 {
	return c.totalJobs.Load()
}

func (c *Collector) Errors() uint64 {
	return c.totalErrors.Load()
}

func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{})
}
