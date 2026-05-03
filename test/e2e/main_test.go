package e2e

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bartlomiejsadza/remitly-stock-market/internal/handler"
	"github.com/bartlomiejsadza/remitly-stock-market/internal/router"
	"github.com/bartlomiejsadza/remitly-stock-market/internal/store"
	"github.com/redis/go-redis/v9"
)

var baseURL string
var redisClient *redis.Client

const testRedisURL = "redis://localhost:6379/15"

func TestMain(m *testing.M) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	code := runTests(m, logger)
	os.Exit(code)
}

func runTests(m *testing.M, logger *slog.Logger) int {
	if err := initRedisClient(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "redis init failed: %v\n", err)
		return 1
	}
	defer func() { _ = redisClient.Close() }()

	st, err := store.New(testRedisURL)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "store init failed: %v\n", err)
		return 1
	}
	defer func() { _ = st.Close() }()

	h := handler.New(st, logger)

	srv := httptest.NewServer(router.New(h))
	defer srv.Close()

	baseURL = srv.URL
	return m.Run()
}

func initRedisClient() error {
	opts, err := redis.ParseURL(testRedisURL)
	if err != nil {
		return fmt.Errorf("initRedisClient failed: %w", err)
	}

	redisClient = redis.NewClient(opts)
	return redisClient.Ping(context.Background()).Err()
}
