package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/bartlomiejsadza/remitly-stock-market/internal/model"
)

func getJSON[T any](t *testing.T, url string) T {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	return result
}

func resetBank(t *testing.T, stocks []model.Stock) {
	t.Helper()
	flushTestDB(t)
	req := model.StocksRequest{Stocks: stocks}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	resp, err := http.Post(baseURL+"/stocks", "application/json", &buf)
	if err != nil {
		t.Fatalf("resetBank failed: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resetBank: expected 200, got %v", resp.StatusCode)
	}
}

func doTrade(t *testing.T, walletID, stockName, tradeType string) *http.Response {
	t.Helper()
	req := model.TradeRequest{Type: tradeType}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	resp, err := http.Post(baseURL+"/wallets/"+walletID+"/stocks/"+stockName, "application/json", &buf)
	if err != nil {
		t.Fatalf("doTradeFailed: %v", err)
	}

	return resp
}

func getBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	return string(body)
}

func uniqueWalletID() string {
	return fmt.Sprintf("w-%v", time.Now().UnixNano())
}

func flushTestDB(t *testing.T) {
	t.Helper()
	if err := redisClient.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("FlushDB failed: %v", err)
	}
}
