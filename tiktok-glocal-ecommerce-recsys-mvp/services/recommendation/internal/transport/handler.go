// Package transport is the delivery layer: it adapts HTTP requests/responses to
// the application service. It depends on app (inward) and never the other way.
package transport

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"recommendation/internal/app"
)

type Handler struct {
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

// Register wires the routes onto the router.
func (h *Handler) Register(r *mux.Router) {
	r.HandleFunc("/healthz", h.health).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/recommendations", h.recommend).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/configs", h.getConfig).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/configs", h.putConfig).Methods(http.MethodPut)
	r.HandleFunc("/api/v1/configs/history", h.history).Methods(http.MethodGet)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","service":"recommendation"}`))
}

func (h *Handler) recommend(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	res, err := h.svc.Recommend(r.Context(), userID)
	if err != nil {
		log.Println("content service unavailable after retries/breaker:", err)
		http.Error(w, "content service unavailable", http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"trace_id": "req-" + time.Now().UTC().Format("20060102150405.000"),
		"code":     200,
		"message":  "success",
		"data": map[string]interface{}{
			"user_id":  res.UserID,
			"strategy": res.Strategy,
			"degraded": res.Degraded,
			"videos":   res.Videos,
		},
	})
}

func (h *Handler) getConfig(w http.ResponseWriter, _ *http.Request) {
	cfg := h.svc.GetConfig()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"strategy_name": cfg.StrategyName,
			"weight":        cfg.Weight,
			"is_active":     true,
			"updated_at":    cfg.UpdatedAt,
		},
	})
}

func (h *Handler) putConfig(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		StrategyName string  `json:"strategy_name"`
		Weight       float64 `json:"weight"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if payload.StrategyName == "" {
		http.Error(w, "strategy_name is required", http.StatusBadRequest)
		return
	}

	cfg, err := h.svc.UpdateConfig(payload.StrategyName, payload.Weight)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "Configuration deployed to Ranking Shards successfully",
		"data": map[string]interface{}{
			"strategy_name": cfg.StrategyName,
			"weight":        cfg.Weight,
			"is_active":     true,
			"updated_at":    cfg.UpdatedAt,
		},
	})
}

func (h *Handler) history(w http.ResponseWriter, _ *http.Request) {
	history, err := h.svc.History(20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    history,
	})
}
