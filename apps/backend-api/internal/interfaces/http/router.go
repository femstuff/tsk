package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"tsk/backend-api/internal/infrastructure/metrics"
)

const (
	requestSourceHeader    = "X-TSK-Request-Source"
	requestSourceAdminPoll = "admin-poll"
)

func NewRouter(handler *Handler, collector *metrics.Collector) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", handler.Root)
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("GET /api/v1/health", handler.Health)
	mux.HandleFunc("GET /api/v1/document-templates", handler.ListTemplates)
	mux.HandleFunc("POST /api/v1/document-templates", handler.CreateTemplate)
	mux.HandleFunc("GET /api/v1/document-templates/{id}/download", handler.DownloadTemplate)
	mux.HandleFunc("GET /api/v1/document-jobs", handler.ListJobs)
	mux.HandleFunc("POST /api/v1/document-jobs", handler.CreateJob)
	mux.HandleFunc("PATCH /api/v1/document-jobs/{id}/status", handler.UpdateJobStatus)
	mux.HandleFunc("POST /api/v1/mobile/voice-requests", handler.CreateMobileVoiceRequest)
	mux.HandleFunc("POST /api/v1/mobile/bitrix-intent", handler.CreateMobileBitrixIntent)
	mux.HandleFunc("GET /api/v1/mobile/bitrix-tasks", handler.ListMobileBitrixTasks)
	mux.HandleFunc("GET /api/v1/source-documents", handler.ListSourceDocuments)
	mux.HandleFunc("GET /api/v1/source-documents/{id}/download", handler.DownloadSourceDocument)
	mux.HandleFunc("GET /api/v1/generated-documents", handler.ListGeneratedDocuments)
	mux.HandleFunc("GET /api/v1/generated-documents/{id}/download", handler.DownloadGeneratedDocument)
	mux.HandleFunc("GET /api/v1/task-commands", handler.ListTaskCommands)
	mux.HandleFunc("POST /api/v1/task-commands", handler.CreateTaskCommand)
	mux.HandleFunc("GET /api/v1/processing-events", handler.ListProcessingEvents)
	mux.HandleFunc("POST /api/v1/admin/voice-bitrix-pipeline", handler.AdminVoiceBitrixPipeline)
	mux.Handle("GET /metrics", collector.Handler())

	return withObservability(withCORS(mux), collector)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func withObservability(next http.Handler, collector *metrics.Collector) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)

		pathLabel := normalizePath(r.URL.Path)
		collector.RecordHTTPRequest(
			r.Method,
			pathLabel,
			recorder.statusCode,
			time.Since(startedAt),
			isBusinessRequest(r, pathLabel),
		)
		slog.Debug("request served",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.statusCode,
		)
	})
}

func isBusinessRequest(r *http.Request, normalizedPath string) bool {
	if r.Method == http.MethodOptions {
		return false
	}

	switch normalizedPath {
	case "/metrics", "/health", "/api/v1/health":
		return false
	}

	return !strings.EqualFold(strings.TrimSpace(r.Header.Get(requestSourceHeader)), requestSourceAdminPoll)
}

func normalizePath(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/v1/document-jobs/") && strings.HasSuffix(path, "/status"):
		return "/api/v1/document-jobs/{id}/status"
	case strings.HasPrefix(path, "/api/v1/document-templates/") && strings.HasSuffix(path, "/download"):
		return "/api/v1/document-templates/{id}/download"
	case strings.HasPrefix(path, "/api/v1/source-documents/") && strings.HasSuffix(path, "/download"):
		return "/api/v1/source-documents/{id}/download"
	case strings.HasPrefix(path, "/api/v1/generated-documents/") && strings.HasSuffix(path, "/download"):
		return "/api/v1/generated-documents/{id}/download"
	case path == "/api/v1/admin/voice-bitrix-pipeline":
		return "/api/v1/admin/voice-bitrix-pipeline"
	default:
		return path
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
