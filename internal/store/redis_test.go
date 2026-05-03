package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/bartlomiejsadza/remitly-stock-market/internal/model"
)

func TestNew_ConnectsToRedis(t *testing.T) {
	const url = "redis://localhost:6379/15"
	s, err := New(url)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("close failed: %v", err)
		}
	})
}

func TestSetBankStocks_RoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	want := []model.Stock{
		{Name: "AAPL", Quantity: 10},
		{Name: "GOOG", Quantity: 5},
		{Name: "MSFT", Quantity: 3},
	}

	if err := s.SetBankStocks(ctx, want); err != nil {
		t.Fatalf("SetBankStocks failed: %v", err)
	}

	assertBank(t, s, map[string]int{"AAPL": 10, "GOOG": 5, "MSFT": 3})
}

func TestGetBankStocks_Empty(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	got, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks: %v", err)
	}
	if got == nil {
		t.Errorf("expected non-nil slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestSetBankStocks_Overwrites(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	initial := []model.Stock{
		{Name: "AAPL", Quantity: 10},
		{Name: "GOOG", Quantity: 5},
	}
	if err := s.SetBankStocks(ctx, initial); err != nil {
		t.Fatalf("SetBankStocks failed: %v", err)
	}

	second := []model.Stock{
		{Name: "MSFT", Quantity: 3},
	}
	if err := s.SetBankStocks(ctx, second); err != nil {
		t.Fatalf("SetBankStocks failed: %v", err)
	}

	assertBank(t, s, map[string]int{"MSFT": 3})
}

func TestBuy_Success(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	bankStocks := []model.Stock{
		{Name: "AAPL", Quantity: 10},
	}
	if err := s.SetBankStocks(ctx, bankStocks); err != nil {
		t.Fatalf("SetBankStocks failed: %v", err)
	}

	if err := s.Buy(ctx, "bartek", "AAPL"); err != nil {
		t.Fatalf("Buy failed: %v", err)
	}

	assertBank(t, s, map[string]int{"AAPL": 9})
	assertWallet(t, s, "bartek", map[string]int{"AAPL": 1})
	assertLog(t, s, []model.LogEntry{{Type: "buy", WalletID: "bartek", StockName: "AAPL"}})
}

func TestBuy_BankEmpty(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	bankStocks := []model.Stock{
		{Name: "AAPL", Quantity: 0},
	}
	if err := s.SetBankStocks(ctx, bankStocks); err != nil {
		t.Fatalf("SetBankStocks failed: %v", err)
	}

	if err := s.Buy(ctx, "bartek", "AAPL"); !errors.Is(err, ErrInsufficientBank) {
		t.Errorf("Buy: got %v, want ErrInsufficientBank", err)
	}

	assertBank(t, s, map[string]int{"AAPL": 0})
	assertWalletEmpty(t, s, "bartek")
	assertLogEmpty(t, s)
}

func TestSell_Success(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	bankStocks := []model.Stock{
		{Name: "AAPL", Quantity: 10},
	}
	if err := s.SetBankStocks(ctx, bankStocks); err != nil {
		t.Fatalf("SetBankStocks failed: %v", err)
	}
	if err := s.Buy(ctx, "bartek", "AAPL"); err != nil {
		t.Fatalf("Buy (setup) failed: %v", err)
	}

	if err := s.Sell(ctx, "bartek", "AAPL"); err != nil {
		t.Fatalf("Sell failed: %v", err)
	}

	assertBank(t, s, map[string]int{"AAPL": 10})
	assertWallet(t, s, "bartek", map[string]int{"AAPL": 0})
	assertLog(t, s, []model.LogEntry{
		{Type: "buy", WalletID: "bartek", StockName: "AAPL"},
		{Type: "sell", WalletID: "bartek", StockName: "AAPL"},
	})
}

func TestSell_StockUnknown(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	bankStocks := []model.Stock{
		{Name: "AAPL", Quantity: 10},
	}
	if err := s.SetBankStocks(ctx, bankStocks); err != nil {
		t.Fatalf("SetBankStocks failed: %v", err)
	}

	if err := s.Sell(ctx, "bartek", "GOOG"); !errors.Is(err, ErrStockNotFound) {
		t.Errorf("Sell: got %v, want ErrStockNotFound", err)
	}

	assertBank(t, s, map[string]int{"AAPL": 10})
	assertWalletEmpty(t, s, "bartek")
	assertLogEmpty(t, s)
}

func TestSell_WalletEmpty(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	bankStocks := []model.Stock{
		{Name: "AAPL", Quantity: 10},
	}
	if err := s.SetBankStocks(ctx, bankStocks); err != nil {
		t.Fatalf("SetBankStocks failed: %v", err)
	}

	if err := s.Sell(ctx, "bartek", "AAPL"); !errors.Is(err, ErrInsufficientWallet) {
		t.Errorf("Sell: got %v, want ErrInsufficientWallet", err)
	}

	assertBank(t, s, map[string]int{"AAPL": 10})
	assertWalletEmpty(t, s, "bartek")
	assertLogEmpty(t, s)
}

func TestBuy_StockUnknown(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	bankStocks := []model.Stock{
		{Name: "AAPL", Quantity: 10},
	}
	if err := s.SetBankStocks(ctx, bankStocks); err != nil {
		t.Fatalf("SetBankStocks failed: %v", err)
	}

	if err := s.Buy(ctx, "bartek", "GOOG"); !errors.Is(err, ErrStockNotFound) {
		t.Errorf("Buy: got %v, want ErrStockNotFound", err)
	}

	assertBank(t, s, map[string]int{"AAPL": 10})
	assertWalletEmpty(t, s, "bartek")
	assertLogEmpty(t, s)
}

func TestBuy_Concurrent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	const N = 100
	const bankStart = 50

	if err := s.SetBankStocks(ctx, []model.Stock{{Name: "AAPL", Quantity: bankStart}}); err != nil {
		t.Fatalf("SetBankStocks: %v", err)
	}

	var wg sync.WaitGroup
	var successCount int64
	var failCount int64

	for i := 0; i < N; i++ {
		id := i
		wg.Go(func() {
			walletID := fmt.Sprintf("wallet-%d", id)
			err := s.Buy(ctx, walletID, "AAPL")
			switch {
			case err == nil:
				atomic.AddInt64(&successCount, 1)
			case errors.Is(err, ErrInsufficientBank):
				atomic.AddInt64(&failCount, 1)
			default:
				t.Errorf("goroutine %d: unexpected err: %v", id, err)
			}
		})
	}
	wg.Wait()

	if gotSuccess := atomic.LoadInt64(&successCount); gotSuccess != int64(bankStart) {
		t.Errorf("successCount: got %d, want %d", gotSuccess, bankStart)
	}
	if gotFail := atomic.LoadInt64(&failCount); gotFail != int64(N-bankStart) {
		t.Errorf("failCount: got %d, want %d", gotFail, N-bankStart)
	}

	assertBank(t, s, map[string]int{"AAPL": 0})

	// sum of all wallets == bankStart
	totalInWallets := 0
	for i := 0; i < N; i++ {
		stocks, _ := s.GetWallet(ctx, fmt.Sprintf("wallet-%d", i))
		totalInWallets += stocksToMap(stocks)["AAPL"]
	}
	if totalInWallets != bankStart {
		t.Errorf("sum of wallets: got %d, want %d", totalInWallets, bankStart)
	}

	log, _ := s.GetLog(ctx)
	if len(log) != bankStart {
		t.Errorf("log length: got %d, want %d", len(log), bankStart)
	}
}

func TestGetLog_Cap(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	const cap = 10000
	const buys = cap + 1

	bankStocks := []model.Stock{
		{Name: "AAPL", Quantity: buys},
	}
	if err := s.SetBankStocks(ctx, bankStocks); err != nil {
		t.Fatalf("SetBankStocks failed: %v", err)
	}

	for i := 0; i < buys; i++ {
		walletID := fmt.Sprintf("wallet-%d", i)
		if err := s.Buy(ctx, walletID, "AAPL"); err != nil {
			t.Fatalf("Buy iteration %d: %v", i, err)
		}
	}

	gotLog, err := s.GetLog(ctx)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}

	if len(gotLog) != cap {
		t.Errorf("log length: %d, want: %d", len(gotLog), cap)
	}
	if len(gotLog) > 0 {
		wantFirst := model.LogEntry{
			Type:      "buy",
			WalletID:  "wallet-1",
			StockName: "AAPL",
		}
		if gotLog[0] != wantFirst {
			t.Errorf("first entry: got %+v, want %+v", gotLog[0], wantFirst)
		}
	}
	if len(gotLog) > 0 {
		wantLast := model.LogEntry{
			Type:      "buy",
			WalletID:  "wallet-10000",
			StockName: "AAPL",
		}
		if gotLog[len(gotLog)-1] != wantLast {
			t.Errorf("last entry: got %+v, want %+v", gotLog[len(gotLog)-1], wantLast)
		}
	}
}
