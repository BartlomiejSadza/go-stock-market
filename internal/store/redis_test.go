package store

import (
	"context"
	"fmt"
	"reflect"
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

func newTestStore(t *testing.T) *Store {
	t.Helper()
	const url = "redis://localhost:6379/15"
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

// TODO: refactor tests <<DRY>>

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

	got, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks failed: %v", err)
	}

	gotMap := make(map[string]int, len(got))
	for _, st := range got {
		gotMap[st.Name] = st.Quantity
	}

	wantMap := make(map[string]int, len(want))
	for _, st := range want {
		wantMap[st.Name] = st.Quantity
	}

	if !reflect.DeepEqual(gotMap, wantMap) {
		t.Errorf("got %v, want %v", gotMap, wantMap)
	}
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

	got, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks failed: %v", err)
	}

	gotMap := make(map[string]int, len(got))
	for _, st := range got {
		gotMap[st.Name] = st.Quantity
	}

	want := map[string]int{"MSFT": 3}

	if !reflect.DeepEqual(gotMap, want) {
		t.Errorf("got: %v, want: %v", gotMap, want)
	}
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

	status, err := s.Buy(ctx, "bartek", "AAPL")
	if err != nil {
		t.Fatalf("Buy failed, status: %v, err: %v", status, err)
	}
	if status != 1 {
		t.Errorf("Buy status: got %d, want 1", status)
	}

	// Bank check
	bankStatus, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks failed: %v", err)
	}
	bankMap := make(map[string]int, len(bankStatus))
	for _, st := range bankStatus {
		bankMap[st.Name] = st.Quantity
	}
	wantBank := map[string]int{"AAPL": 9}
	if !reflect.DeepEqual(bankMap, wantBank) {
		t.Errorf("bank: got %v, want %v", bankMap, wantBank)
	}

	// personal wallet check
	walletStatus, err := s.GetWallet(ctx, "bartek")
	if err != nil {
		t.Fatalf("GetWallet failed: %v", err)
	}
	walletMap := make(map[string]int, len(walletStatus))
	for _, st := range walletStatus {
		walletMap[st.Name] = st.Quantity
	}
	wantWallet := map[string]int{"AAPL": 1}
	if !reflect.DeepEqual(walletMap, wantWallet) {
		t.Errorf("wallet: got %v, want: %v", walletMap, wantWallet)
	}

	// log check
	gotLog, err := s.GetLog(ctx)
	if err != nil {
		t.Fatalf("GetLog failure: %v", err)
	}
	wantLog := []model.LogEntry{
		{Type: "buy", WalletID: "bartek", StockName: "AAPL"},
	}
	if !reflect.DeepEqual(gotLog, wantLog) {
		t.Errorf("log: got %v, want %v", gotLog, wantLog)
	}
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

	status, err := s.Buy(ctx, "bartek", "AAPL")
	if err != nil {
		t.Fatalf("Buy failed, status: %v, err: %v", status, err)
	}
	if status != 0 {
		t.Errorf("Buy status: got %d, want 0", status)
	}

	// Bank unchanged
	gotBank, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks failed: %v", err)
	}
	bankMap := make(map[string]int, len(gotBank))
	for _, st := range gotBank {
		bankMap[st.Name] = st.Quantity
	}
	wantBank := map[string]int{"AAPL": 0}
	if !reflect.DeepEqual(bankMap, wantBank) {
		t.Errorf("bank: got %v, want %v", bankMap, wantBank)
	}

	// Wallet empty
	gotWallet, err := s.GetWallet(ctx, "bartek")
	if err != nil {
		t.Fatalf("GetWallet failed: %v", err)
	}
	if len(gotWallet) != 0 {
		t.Errorf("wallet should be empty, got %v", gotWallet)
	}

	// Log empty
	gotLog, err := s.GetLog(ctx)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}
	if len(gotLog) != 0 {
		t.Errorf("log should be empty, got %v", gotLog)
	}
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
	if _, err := s.Buy(ctx, "bartek", "AAPL"); err != nil {
		t.Fatalf("Buy (setup) failed: %v", err)
	}

	status, err := s.Sell(ctx, "bartek", "AAPL")
	if err != nil {
		t.Fatalf("Sell failed, status: %v, err: %v", status, err)
	}
	if status != 1 {
		t.Errorf("Sell status: got %d, want 1", status)
	}

	bankStatus, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks failed: %v", err)
	}
	bankMap := make(map[string]int, len(bankStatus))
	for _, st := range bankStatus {
		bankMap[st.Name] = st.Quantity
	}
	wantBank := map[string]int{"AAPL": 10}
	if !reflect.DeepEqual(bankMap, wantBank) {
		t.Errorf("bank: got %v, want %v", bankMap, wantBank)
	}

	walletStatus, err := s.GetWallet(ctx, "bartek")
	if err != nil {
		t.Fatalf("GetWallet failed: %v", err)
	}
	walletMap := make(map[string]int, len(walletStatus))
	for _, st := range walletStatus {
		walletMap[st.Name] = st.Quantity
	}
	wantWallet := map[string]int{"AAPL": 0}
	if !reflect.DeepEqual(walletMap, wantWallet) {
		t.Errorf("wallet: got %v, want %v", walletMap, wantWallet)
	}

	gotLog, err := s.GetLog(ctx)
	if err != nil {
		t.Fatalf("GetLog failure: %v", err)
	}
	wantLog := []model.LogEntry{
		{Type: "buy", WalletID: "bartek", StockName: "AAPL"},
		{Type: "sell", WalletID: "bartek", StockName: "AAPL"},
	}
	if !reflect.DeepEqual(gotLog, wantLog) {
		t.Errorf("log: got %v, want %v", gotLog, wantLog)
	}
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

	status, err := s.Sell(ctx, "bartek", "GOOG")
	if err != nil {
		t.Fatalf("Sell failed, status: %v, err: %v", status, err)
	}
	if status != -1 {
		t.Errorf("Sell status: got %d, want -1", status)
	}

	gotBank, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks failed: %v", err)
	}
	bankMap := make(map[string]int, len(gotBank))
	for _, st := range gotBank {
		bankMap[st.Name] = st.Quantity
	}
	wantBank := map[string]int{"AAPL": 10}
	if !reflect.DeepEqual(bankMap, wantBank) {
		t.Errorf("bank: got %v, want %v", bankMap, wantBank)
	}

	gotWallet, err := s.GetWallet(ctx, "bartek")
	if err != nil {
		t.Fatalf("GetWallet failed: %v", err)
	}
	if len(gotWallet) != 0 {
		t.Errorf("wallet should be empty, got %v", gotWallet)
	}

	gotLog, err := s.GetLog(ctx)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}
	if len(gotLog) != 0 {
		t.Errorf("log should be empty, got %v", gotLog)
	}
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

	status, err := s.Sell(ctx, "bartek", "AAPL")
	if err != nil {
		t.Fatalf("Sell failed, status: %v, err: %v", status, err)
	}
	if status != 0 {
		t.Errorf("Sell status: got %d, want 0", status)
	}

	gotBank, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks failed: %v", err)
	}
	bankMap := make(map[string]int, len(gotBank))
	for _, st := range gotBank {
		bankMap[st.Name] = st.Quantity
	}
	wantBank := map[string]int{"AAPL": 10}
	if !reflect.DeepEqual(bankMap, wantBank) {
		t.Errorf("bank: got %v, want %v", bankMap, wantBank)
	}

	gotWallet, err := s.GetWallet(ctx, "bartek")
	if err != nil {
		t.Fatalf("GetWallet failed: %v", err)
	}
	if len(gotWallet) != 0 {
		t.Errorf("wallet should be empty, got %v", gotWallet)
	}

	gotLog, err := s.GetLog(ctx)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}
	if len(gotLog) != 0 {
		t.Errorf("log should be empty, got %v", gotLog)
	}
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

	status, err := s.Buy(ctx, "bartek", "GOOG")
	if err != nil {
		t.Fatalf("Buy failed, status: %v, err: %v", status, err)
	}
	if status != -1 {
		t.Errorf("Buy status: got %d, want -1", status)
	}
	gotBank, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks failed: %v", err)
	}
	bankMap := make(map[string]int, len(gotBank))
	for _, st := range gotBank {
		bankMap[st.Name] = st.Quantity
	}
	wantBank := map[string]int{"AAPL": 10}
	if !reflect.DeepEqual(bankMap, wantBank) {
		t.Errorf("bank: got %v, want %v", bankMap, wantBank)
	}

	gotWallet, err := s.GetWallet(ctx, "bartek")
	if err != nil {
		t.Fatalf("GetWallet failed: %v", err)
	}
	if len(gotWallet) != 0 {
		t.Errorf("wallet should be empty, got %v", gotWallet)
	}

	gotLog, err := s.GetLog(ctx)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}
	if len(gotLog) != 0 {
		t.Errorf("log should be empty, got %v", gotLog)
	}
}

func TestBuy_Concurrent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	const N = 100
	const bankStart = 50

	if err := s.SetBankStocks(ctx, []model.Stock{
		{Name: "AAPL", Quantity: bankStart},
	}); err != nil {
		t.Fatalf("SetBankStocks failed: %v", err)
	}

	var wg sync.WaitGroup
	var successCount int64
	var failCount int64

	for i := 0; i < N; i++ {
		id := i
		wg.Go(func() {
			walletID := fmt.Sprintf("wallet-%d", id)
			status, err := s.Buy(ctx, walletID, "AAPL")
			if err != nil {
				t.Errorf("goroutine %d: Buy err: %v", id, err)
				return
			}
			switch status {
			case 1:
				atomic.AddInt64(&successCount, 1)
			case 0:
				atomic.AddInt64(&failCount, 1)
			default:
				t.Errorf("goroutine %d: unexpected status %d", id, status)
			}
		})
	}
	wg.Wait()

	gotSuccess := atomic.LoadInt64(&successCount)
	gotFail := atomic.LoadInt64(&failCount)

	if gotSuccess != int64(bankStart) {
		t.Errorf("successCount: got %d, want %d", gotSuccess, bankStart)
	}

	if gotFail != int64(N-bankStart) {
		t.Errorf("failCount: got %d, want %d", gotFail, N-bankStart)
	}

	// bank drained to 0
	gotBank, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks failed %v", err)
	}
	bankMap := make(map[string]int, len(gotBank))
	for _, st := range gotBank {
		bankMap[st.Name] = st.Quantity
	}
	if bankMap["AAPL"] != 0 {
		t.Errorf("bank AAPL: got %d, want 0", bankMap["AAPL"])
	}

	// sum of all wallets == bankStart
	totalInWallets := 0
	for i := 0; i < N; i++ {
		id := i
		walletID := fmt.Sprintf("wallet-%d", id)
		stocks, err := s.GetWallet(ctx, walletID)
		if err != nil {
			t.Fatalf("GetWallet(%s) failed: %v", walletID, err)
		}

		for _, st := range stocks {
			if st.Name == "AAPL" {
				totalInWallets += st.Quantity
			}
		}
	}
	if totalInWallets != bankStart {
		t.Errorf("sum of wallets: got %d, want %d", totalInWallets, bankStart)
	}

	gotLog, err := s.GetLog(ctx)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}
	if len(gotLog) != bankStart { // only success is logged
		t.Errorf("log length: got %d, want: %d", len(gotLog), bankStart)
	}
}
