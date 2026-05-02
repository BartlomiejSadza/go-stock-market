package store

import (
	"context"
	"reflect"
	"testing"

	"github.com/bartlomiejsadza/remitly-stock-market/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	const url = "redis://localhost:6379"

	s, err := New(url)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if err := s.rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("FlushDB failed: %v", err)
	}

	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("close failed: %v", err)
		}
	})
	return s
}

func stocksToMap(stocks []model.Stock) map[string]int {
	m := make(map[string]int, len(stocks))
	for _, st := range stocks {
		m[st.Name] = st.Quantity
	}
	return m
}

func assertBank(t *testing.T, s *Store, want map[string]int) {
	t.Helper()
	got, err := s.GetBankStocks(context.Background())
	if err != nil {
		t.Fatalf("GetBankStocks: %v", err)
	}
	if !reflect.DeepEqual(stocksToMap(got), want) {
		t.Errorf("bank: got %v, want %v", stocksToMap(got), want)
	}
}

func assertWallet(t *testing.T, s *Store, walletID string, want map[string]int) {
	t.Helper()
	got, err := s.GetWallet(context.Background(), walletID)
	if err != nil {
		t.Fatalf("GetWallet: %v", err)
	}
	if !reflect.DeepEqual(stocksToMap(got), want) {
		t.Errorf("wallet %s: got %v, want %v", walletID, stocksToMap(got), want)
	}
}

func assertWalletEmpty(t *testing.T, s *Store, walletID string) {
	t.Helper()
	got, err := s.GetWallet(context.Background(), walletID)
	if err != nil {
		t.Fatalf("GotWallet: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("wallet %s should be empty, got %+v", walletID, got)
	}
}

func assertLog(t *testing.T, s *Store, want []model.LogEntry) {
	t.Helper()
	got, err := s.GetLog(context.Background())
	if err != nil {
		t.Fatalf("GetLog: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("log: got %v, want: %v", got, want)
	}
}

func assertLogEmpty(t *testing.T, s *Store) {
	t.Helper()
	got, err := s.GetLog(context.Background())
	if err != nil {
		t.Fatalf("GetLog: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("log should be empty, got %v", got)
	}
}
