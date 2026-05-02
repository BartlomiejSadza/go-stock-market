package handler

import (
	"encoding/json"
	"net/http"

	"github.com/bartlomiejsadza/remitly-stock-market/internal/model"
)

func (h *Handler) GetStocks(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.store.GetBankStocks(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(model.StocksResponse{Stocks: stocks}); err != nil {
		h.logger.Error("encode response", "err", err)
		return
	}
}

func (h *Handler) SetStocks(w http.ResponseWriter, r *http.Request) {
	var req model.StocksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.store.SetBankStocks(r.Context(), req.Stocks); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
