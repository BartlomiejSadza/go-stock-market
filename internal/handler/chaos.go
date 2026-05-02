package handler

import (
	"net/http"
	"os"
)

func (h *Handler) Chaos(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("chaos endpoint called, terminating...")
	os.Exit(1)
}
