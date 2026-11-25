package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type Handler struct {
	logger *slog.Logger
}

func NewHandler(logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	h := &Handler{logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	return mux
}

type healthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	}); err != nil {
		h.logger.Error("write health response", "error", err)
	}
}
