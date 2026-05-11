package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	app "tsk/backend-api/internal/application/documentjob"
	domain "tsk/backend-api/internal/domain/documentjob"
	"tsk/backend-api/internal/infrastructure/config"
	"tsk/backend-api/internal/infrastructure/metrics"
	"tsk/backend-api/pkg/httpx"
)

type Handler struct {
	service *app.Service
	config  config.Config
	metrics *metrics.Collector
}

func NewHandler(service *app.Service, cfg config.Config, metrics *metrics.Collector) *Handler {
	return &Handler{
		service: service,
		config:  cfg,
		metrics: metrics,
	}
}

func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"service":     h.config.ServiceName,
		"environment": h.config.Environment,
		"docs": map[string]string{
			"healthShort":            "/health",
			"health":                 "/api/v1/health",
			"templates":              "/api/v1/document-templates",
			"templateDownload":       "/api/v1/document-templates/{id}/download",
			"jobs":                   "/api/v1/document-jobs",
			"jobStatus":              "/api/v1/document-jobs/{id}/status",
			"mobileVoiceRequest":     "/api/v1/mobile/voice-requests",
			"mobileBitrixIntent":     "POST /api/v1/mobile/bitrix-intent",
			"mobileBitrixTasks":      "GET /api/v1/mobile/bitrix-tasks",
			"sourceDocuments":        "/api/v1/source-documents",
			"sourceDocumentDownload": "/api/v1/source-documents/{id}/download",
			"generatedDocuments":     "/api/v1/generated-documents",
			"documentDownload":       "/api/v1/generated-documents/{id}/download",
			"taskCommands":           "/api/v1/task-commands",
			"processingEvents":       "/api/v1/processing-events",
			"adminVoiceBitrix":       "POST /api/v1/admin/voice-bitrix-pipeline",
			"metrics":                "/metrics",
		},
	})
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"status":              "ok",
		"service":             h.config.ServiceName,
		"environment":         h.config.Environment,
		"database":            "configured",
		"storageRoot":         h.config.StorageRoot,
		"uptimeSeconds":       h.metrics.Uptime().Seconds(),
		"productRequestsTotal": h.metrics.BusinessRequests(),
		"httpRequestsTotalRaw": h.metrics.Requests(),
		"jobsCreatedTotal":    h.metrics.JobsCreated(),
		"errorsTotal":         h.metrics.Errors(),
	})
}

func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.service.ListTemplates(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": templates,
	})
}

func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid multipart payload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "template file is required")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "unable to read uploaded file")
		return
	}

	template, err := h.service.CreateTemplate(r.Context(), app.CreateTemplateInput{
		Name:        r.FormValue("name"),
		Category:    r.FormValue("category"),
		Version:     r.FormValue("version"),
		Description: r.FormValue("description"),
		FileName:    header.Filename,
		MimeType:    header.Header.Get("Content-Type"),
		Content:     content,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"item": template,
	})
}

func (h *Handler) DownloadTemplate(w http.ResponseWriter, r *http.Request) {
	template, err := h.service.GetTemplate(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	path, err := h.service.ResolveStoragePath(template.StorageKey)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.serveDownload(w, r, path, template.FileName, template.MimeType)
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.service.ListJobs(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": jobs,
	})
}

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var request app.CreateJobInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	job, err := h.service.CreateJob(r.Context(), request)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"item": job,
	})
}

func (h *Handler) CreateMobileVoiceRequest(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid multipart payload")
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "audio file is required")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "unable to read uploaded audio")
		return
	}

	result, err := h.service.CreateMobileVoiceRequest(r.Context(), app.CreateMobileVoiceRequestInput{
		TemplateID:      r.FormValue("templateId"),
		SourceName:      r.FormValue("sourceName"),
		RequestedBy:     r.FormValue("requestedBy"),
		Payload:         r.FormValue("payload"),
		DeliveryChannel: domain.DeliveryChannel(strings.TrimSpace(r.FormValue("deliveryChannel"))),
		DeliveryAddress: r.FormValue("deliveryAddress"),
		TaskCommandText: r.FormValue("taskCommandText"),
		TaskTarget:      domain.TaskTargetSystem(strings.TrimSpace(r.FormValue("taskTarget"))),
	}, header.Filename, header.Header.Get("Content-Type"), content)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"item": result,
	})
}

func (h *Handler) UpdateJobStatus(w http.ResponseWriter, r *http.Request) {
	var request app.UpdateJobStatusInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	job, err := h.service.UpdateJobStatus(r.Context(), r.PathValue("id"), request)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"item": job,
	})
}

func (h *Handler) ListGeneratedDocuments(w http.ResponseWriter, r *http.Request) {
	documents, err := h.service.ListGeneratedDocuments(r.Context(), r.URL.Query().Get("jobId"))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": documents,
	})
}

func (h *Handler) ListSourceDocuments(w http.ResponseWriter, r *http.Request) {
	documents, err := h.service.ListSourceDocuments(r.Context(), r.URL.Query().Get("jobId"))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": documents,
	})
}

func (h *Handler) DownloadSourceDocument(w http.ResponseWriter, r *http.Request) {
	document, err := h.service.GetSourceDocument(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	path, err := h.service.ResolveStoragePath(document.StorageKey)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.serveDownload(w, r, path, document.FileName, document.MimeType)
}

func (h *Handler) DownloadGeneratedDocument(w http.ResponseWriter, r *http.Request) {
	document, err := h.service.GetGeneratedDocument(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	path, err := h.service.ResolveStoragePath(document.StorageKey)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.serveDownload(w, r, path, document.FileName, document.MimeType)
}

func (h *Handler) ListTaskCommands(w http.ResponseWriter, r *http.Request) {
	commands, err := h.service.ListTaskCommands(r.Context(), r.URL.Query().Get("jobId"))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": commands,
	})
}

func (h *Handler) CreateTaskCommand(w http.ResponseWriter, r *http.Request) {
	var request app.CreateTaskCommandInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	command, err := h.service.CreateTaskCommand(r.Context(), request)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"item": command,
	})
}

func (h *Handler) ListMobileBitrixTasks(w http.ResponseWriter, r *http.Request) {
	limit := 60
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	responsibleOverride := 0
	if value := strings.TrimSpace(r.URL.Query().Get("responsibleId")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			responsibleOverride = parsed
		}
	}

	bundle, err := h.service.ListBitrixTasksForMobile(r.Context(), limit, responsibleOverride)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"responsibleUserId": bundle.ResponsibleUserID,
		"stats":             bundle.Stats,
		"items":             bundle.Items,
	})
}

func (h *Handler) CreateMobileBitrixIntent(w http.ResponseWriter, r *http.Request) {
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(ct, "application/json") {
		var body struct {
			Text              string `json:"text"`
			DealID            int    `json:"dealId"`
			DealTitle         string `json:"dealTitle"`
			DealHint          string `json:"dealHint"`
			StageHint         string `json:"stageHint"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
		result, err := h.service.RunMobileBitrixIntent(r.Context(), app.MobileBitrixIntentInput{
			Text:              strings.TrimSpace(body.Text),
			DealIDOverride:    body.DealID,
			DealTitleOverride: strings.TrimSpace(body.DealTitle),
			DealHintText:      strings.TrimSpace(body.DealHint),
			StageHintText:     strings.TrimSpace(body.StageHint),
		})
		if err != nil {
			h.writeDomainError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"item": result})
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid multipart payload")
		return
	}

	text := strings.TrimSpace(r.FormValue("text"))
	var audio []byte
	fileName := "mobile-voice.m4a"
	mimeType := "audio/mp4"
	if file, header, err := r.FormFile("audio"); err == nil {
		defer file.Close()
		content, readErr := io.ReadAll(file)
		if readErr != nil {
			httpx.WriteError(w, http.StatusBadRequest, "unable to read uploaded audio")
			return
		}
		audio = content
		fileName = header.Filename
		if mt := strings.TrimSpace(header.Header.Get("Content-Type")); mt != "" {
			mimeType = mt
		}
	}

	dealOverride := 0
	if raw := strings.TrimSpace(r.FormValue("dealId")); raw != "" {
		if parsed, convErr := strconv.Atoi(raw); convErr == nil {
			dealOverride = parsed
		}
	}

	result, err := h.service.RunMobileBitrixIntent(r.Context(), app.MobileBitrixIntentInput{
		Text:              text,
		Audio:             audio,
		FileName:          fileName,
		MimeType:          mimeType,
		DealIDOverride:    dealOverride,
		DealTitleOverride: strings.TrimSpace(r.FormValue("dealTitle")),
		DealHintText:      strings.TrimSpace(r.FormValue("dealHint")),
		StageHintText:     strings.TrimSpace(r.FormValue("stageHint")),
	})
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"item": result})
}

func (h *Handler) AdminVoiceBitrixPipeline(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid multipart payload")
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "audio file is required")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "unable to read uploaded audio")
		return
	}

	dealOverride := 0
	if raw := strings.TrimSpace(r.FormValue("dealId")); raw != "" {
		if parsed, convErr := strconv.Atoi(raw); convErr == nil {
			dealOverride = parsed
		}
	}

	result, err := h.service.RunAdminVoiceBitrixPipeline(r.Context(), app.AdminVoiceBitrixInput{
		TemplateID:        strings.TrimSpace(r.FormValue("templateId")),
		SourceName:        strings.TrimSpace(r.FormValue("sourceName")),
		DealIDOverride:    dealOverride,
		DealTitleOverride: strings.TrimSpace(r.FormValue("dealTitle")),
		DealHintText:      strings.TrimSpace(r.FormValue("dealHint")),
		StageHintText:     strings.TrimSpace(r.FormValue("stageHint")),
		FileName:          header.Filename,
		MimeType:          header.Header.Get("Content-Type"),
		Audio:             content,
	})
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"item": result,
	})
}

func (h *Handler) ListProcessingEvents(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			limit = parsed
		}
	}

	events, err := h.service.ListProcessingEvents(r.Context(), r.URL.Query().Get("jobId"), limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": events,
	})
}

func (h *Handler) writeDomainError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, domain.ErrTemplateNotFound),
		errors.Is(err, domain.ErrJobNotFound),
		errors.Is(err, domain.ErrGeneratedDocumentNotFound),
		errors.Is(err, domain.ErrSourceDocumentNotFound),
		errors.Is(err, domain.ErrTaskCommandNotFound):
		status = http.StatusNotFound
	default:
		var pathError *os.PathError
		if errors.As(err, &pathError) {
			status = http.StatusNotFound
		}
	}

	httpx.WriteError(w, status, err.Error())
}

func (h *Handler) serveDownload(w http.ResponseWriter, r *http.Request, path string, fileName string, mimeType string) {
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	http.ServeFile(w, r, path)
}
