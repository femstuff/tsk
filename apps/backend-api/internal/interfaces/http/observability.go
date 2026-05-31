package httpapi

import (
	"net/http"

	"tsk/backend-api/internal/infrastructure/prometheus"
	"tsk/backend-api/pkg/httpx"
)

func (h *Handler) AdminObservabilityDashboard(w http.ResponseWriter, r *http.Request) {
	if h.prometheus == nil || !h.prometheus.Enabled() {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"item": prometheus.DashboardSnapshot{Available: false},
		})
		return
	}

	snapshot, err := h.prometheus.DashboardSnapshot(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"item": snapshot})
}

func (h *Handler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	view, err := h.service.AdminDashboard(r.Context())
	if err != nil {
		h.writeDomainError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"item": view})
}
