package model

type Stock struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// StocksRequest for POST method /stocks
type StocksRequest struct {
	Stocks []Stock `json:"stocks"`
}

// StocksResponse for GET response /stocks
type StocksResponse struct {
	Stocks []Stock `json:"stocks"`
}

// WalletResponse for GET response /wallets/{id}
type WalletResponse struct {
	ID     string  `json:"id"`
	Stocks []Stock `json:"stocks"`
}

// TradeRequest body for POST method /wallets/{id}/stocks/{name}
type TradeRequest struct {
	Type string `json:"type"` // buy or sell
}

// LogEntry log of what operation did perform
type LogEntry struct {
	Type      string `json:"type"`
	WalletID  string `json:"wallet_id"`
	StockName string `json:"stock_name"`
}

// LogResponse response from GET /log
type LogResponse struct {
	Log []LogEntry `json:"log"`
}
