package router

import (
	"net/http"

	"github.com/bartlomiejsadza/remitly-stock-market/internal/handler"
)

func New(h *handler.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /stocks", h.GetStocks)
	mux.HandleFunc("POST /stocks", h.SetStocks)
	mux.HandleFunc("GET /wallets/{wallet_id}", h.GetWallet)
	mux.HandleFunc("GET /wallets/{wallet_id}/stocks/{stock_name}", h.GetWalletStock)
	mux.HandleFunc("POST /wallets/{wallet_id}/stocks/{stock_name}", h.Trade)
	mux.HandleFunc("GET /log", h.GetLog)
	mux.HandleFunc("POST /chaos", h.Chaos)

	return mux
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
