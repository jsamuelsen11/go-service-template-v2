package handlers

import (
	"net/http"

	"github.com/jsamuelsen11/go-service-template-v2/internal/ports"
)

const (
	statusOK       = "ok"
	statusReady    = "ready"
	statusNotReady = "not_ready"
)

// HealthHandler handles liveness and readiness HTTP endpoints.
type HealthHandler struct {
	registry ports.HealthRegistry
}

// NewHealthHandler creates a new HealthHandler with the given health registry.
func NewHealthHandler(registry ports.HealthRegistry) *HealthHandler {
	return &HealthHandler{registry: registry}
}

// Liveness handles GET /health/live. Always returns 200 OK.
func (h *HealthHandler) Liveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": statusOK})
}

// Readiness handles GET /health/ready. Returns 200 if all checks pass,
// 503 if any check fails.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	results := h.registry.CheckAll(r.Context())

	checks := make(map[string]string, len(results))
	healthy := true
	for name, err := range results {
		if err != nil {
			checks[name] = err.Error()
			healthy = false
		} else {
			checks[name] = statusOK
		}
	}

	status := statusReady
	code := http.StatusOK
	if !healthy {
		status = statusNotReady
		code = http.StatusServiceUnavailable
	}

	writeJSON(w, code, map[string]any{
		"status": status,
		"checks": checks,
	})
}
