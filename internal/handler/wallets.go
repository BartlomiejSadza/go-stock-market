package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/bartlomiejsadza/remitly-stock-market/internal/model"
	"github.com/bartlomiejsadza/remitly-stock-market/internal/store"
)

func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	walletID := r.PathValue("wallet_id")

	stocks, err := h.store.GetWallet(r.Context(), walletID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp := model.WalletResponse{
		ID:     walletID,
		Stocks: stocks,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("encode response", "err", err)
		return
	}
}

func (h *Handler) GetWalletStock(w http.ResponseWriter, r *http.Request) {
	walletID := r.PathValue("wallet_id")
	stockName := r.PathValue("stock_name")

	quantity, _, err := h.store.GetWalletStock(r.Context(), walletID, stockName)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if _, err := fmt.Fprint(w, quantity); err != nil {
		h.logger.Error("write response", "err", err)
		return
	}
}

func (h *Handler) Trade(w http.ResponseWriter, r *http.Request) {
	walletID := r.PathValue("wallet_id")
	stockName := r.PathValue("stock_name")

	var req model.TradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	var err error
	switch strings.ToLower(req.Type) {
	case "buy":
		err = h.store.Buy(r.Context(), walletID, stockName)
	case "sell":
		err = h.store.Sell(r.Context(), walletID, stockName)
	default:
		http.Error(w, "trade type must be 'buy' or 'sell'", http.StatusBadRequest)
		return
	}

	switch {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, store.ErrStockNotFound):
		http.Error(w, "stock not found", http.StatusNotFound)
	case errors.Is(err, store.ErrInsufficientBank), errors.Is(err, store.ErrInsufficientWallet):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		h.logger.Error("trade failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
