package handler

import (
	"encoding/json"
	"net/http"

	"github.com/bartlomiejsadza/remitly-stock-market/internal/model"
)

func (h *Handler) GetLog(w http.ResponseWriter, r *http.Request) {
	entries, err := h.store.GetLog(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp := model.LogResponse{Log: entries}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("encode response", "err", err)
		return
	}

}
