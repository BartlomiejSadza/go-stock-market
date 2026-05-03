package e2e

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bartlomiejsadza/remitly-stock-market/internal/model"
)

func TestBuyAndSellFlow(t *testing.T) {
	resetBank(t, []model.Stock{{Name: "AAPL", Quantity: 5}})

	resp := doTrade(t, "w1", "AAPL", "buy")
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("buy failed: %v", resp.StatusCode)
	}

	bank := getJSON[model.StocksResponse](t, baseURL+"/stocks")
	if bank.Stocks[0].Quantity != 4 {
		t.Errorf("bank: expected 4, got %v", bank.Stocks[0].Quantity)
	}

	wallet := getJSON[model.WalletResponse](t, baseURL+"/wallets/w1")
	if len(wallet.Stocks) != 1 || wallet.Stocks[0].Quantity != 1 {
		t.Errorf("wallet: expected 1 AAPL, got %+v", wallet.Stocks)
	}

	resp = doTrade(t, "w1", "AAPL", "sell")
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sell failed: %v", resp.StatusCode)
	}

	bank = getJSON[model.StocksResponse](t, baseURL+"/stocks")
	if bank.Stocks[0].Quantity != 5 {
		t.Errorf("bank after sell: expected 5, got %v", bank.Stocks[0].Quantity)
	}
}

func TestBuyNonExistentStock_404(t *testing.T) {
	resetBank(t, []model.Stock{{Name: "AAPL", Quantity: 5}})

	resp := doTrade(t, "w1", "GOOG", "buy")
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got: %v", resp.StatusCode)
	}
}

func TestBuyEmptyStock_400(t *testing.T) {
	resetBank(t, []model.Stock{{Name: "AAPL", Quantity: 0}})

	resp := doTrade(t, "w1", "AAPL", "buy")
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got: %v", resp.StatusCode)
	}
}

func TestSellFromEmptyWallet_400(t *testing.T) {
	resetBank(t, []model.Stock{{Name: "AAPL", Quantity: 5}})

	resp := doTrade(t, "w7", "AAPL", "sell")
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got: %v", resp.StatusCode)
	}
}

func TestImplicitWalletCreation(t *testing.T) {
	resetBank(t, []model.Stock{})

	resp, err := http.Get(baseURL + "/wallets/bartek")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %v", resp.StatusCode)
	}

	wallet := getJSON[model.WalletResponse](t, baseURL+"/wallets/bartek")
	if len(wallet.Stocks) != 0 {
		t.Errorf("expected empty stocks, got %+v", wallet.Stocks)
	}
}

func TestGetSingleStockQuantity(t *testing.T) {
	resetBank(t, []model.Stock{{Name: "AAPL", Quantity: 5}})

	walletID := uniqueWalletID()

	for i := 0; i < 3; i++ {
		resp := doTrade(t, walletID, "AAPL", "buy")
		resp.Body.Close()
	}

	resp, err := http.Get(baseURL + "/wallets/" + walletID + "/stocks/AAPL")
	if err != nil {
		t.Fatalf("request failed %v", err)
	}
	defer resp.Body.Close()

	body := getBody(t, resp)
	if strings.TrimSpace(body) != "3" {
		t.Errorf("expected `3`, got `%s`", body)
	}
}

func TestLogRecordsOperations(t *testing.T) {
	resetBank(t, []model.Stock{{Name: "AAPL", Quantity: 5}})

	// buy
	resp := doTrade(t, "w3", "AAPL", "buy")
	resp.Body.Close()
	// sell
	resp = doTrade(t, "w3", "AAPL", "sell")
	resp.Body.Close()

	log := getJSON[model.LogResponse](t, baseURL+"/log")

	n := len(log.Log)
	if n < 2 {
		t.Fatalf("expected at least 2 entries, got %v", log.Log)
	}

	// buy
	if log.Log[n-2].Type != "buy" || log.Log[n-2].WalletID != "w3" || log.Log[n-2].StockName != "AAPL" {
		t.Errorf("log[n-2]: expected buy w3 AAPL, got %+v", log.Log[n-2])
	}
	// sell
	if log.Log[n-1].Type != "sell" || log.Log[n-1].WalletID != "w3" || log.Log[n-1].StockName != "AAPL" {
		t.Errorf("log[n-1]: expected sell w3 AAPL, got %+v", log.Log[n-1])
	}
}

func TestSetStocksNegativeQuantity_400(t *testing.T) {
	body := `{"stocks":[{"name":"AAPL","quantity":-1}]}`
	resp, _ := http.Post(baseURL+"/stocks", "application/json", strings.NewReader(body))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %v", resp.StatusCode)
	}
}
