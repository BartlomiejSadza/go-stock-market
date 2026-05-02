package handler

import (
	"log/slog"

	"github.com/bartlomiejsadza/remitly-stock-market/internal/store"
)

type Handler struct {
	store  *store.Store
	logger *slog.Logger
}

func New(s *store.Store, l *slog.Logger) *Handler {
	return &Handler{
		store:  s,
		logger: l,
	}
}
