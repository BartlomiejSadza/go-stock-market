package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bartlomiejsadza/remitly-stock-market/internal/store"
)

func main() {
	portFlag := flag.String("port", "8080", "HTTP server port")
	flag.Parse()

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	addr := ":" + *portFlag

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting server", "addr", addr, "redis", redisURL)

	st, err := store.New(redisURL)
	if err != nil {
		logger.Error("connect to redis failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := st.Close(); err != nil {
			logger.Error("error closing redis", "err", err)
		} else {
			logger.Info("close redis")
		}
	}()

	logger.Info("connected to redis")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	}

	logger.Info("server stopped")
}
