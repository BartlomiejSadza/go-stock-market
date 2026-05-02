package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/bartlomiejsadza/remitly-stock-market/internal/model"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	rdb *redis.Client
}

func New(redisURL string) (*Store, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(opts)

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Store{rdb: rdb}, nil
}

// SetBankStocks POST /stocks
func (s *Store) SetBankStocks(ctx context.Context, stocks []model.Stock) error {
	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, "bank:stocks")
	if len(stocks) > 0 {
		values := make(map[string]interface{}, len(stocks))
		for _, stock := range stocks {
			values[stock.Name] = stock.Quantity
		}
		pipe.HSet(ctx, "bank:stocks", values)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("set bank stocks: %w", err)
	}

	return nil
}

// GetBankStocks  GET /stocks
func (s *Store) GetBankStocks(ctx context.Context) ([]model.Stock, error) {
	raw, err := s.rdb.HGetAll(ctx, "bank:stocks").Result()
	if err != nil {
		return nil, fmt.Errorf("get bank stocks: %w", err)
	}

	stocks := make([]model.Stock, 0, len(raw))
	for name, quantityStr := range raw {
		quantity, err := strconv.Atoi(quantityStr)
		if err != nil {
			return nil, fmt.Errorf("parse quantity for %q: %w", name, err)
		}
		stocks = append(stocks, model.Stock{Name: name, Quantity: quantity})
	}
	return stocks, nil
}

var buyScript = redis.NewScript(`
local stock = redis.call("HGET", KEYS[1], ARGV[1])
if not stock then
	return -1 
end
if tonumber(stock) <= 0 then
	return 0
end
redis.call("HINCRBY", KEYS[1], ARGV[1], -1) -- decrement bank
redis.call("HINCRBY", KEYS[2], ARGV[1], 1)   -- increment wallet
redis.call("RPUSH", KEYS[3], ARGV[2])		-- append audit entry
redis.call("LTRIM", KEYS[3], -10000, -1)	-- cap log at 10k entries
return 1
`)

// Buy attempts to transfer 1 unit of stockName from bank to wallet.
// Returns 1 = success, 0 = bank empty, -1 = stock unknown
func (s *Store) Buy(ctx context.Context, walletID, stockName string) (int, error) {
	entry := model.LogEntry{
		Type:      "buy",
		WalletID:  walletID,
		StockName: stockName,
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return 0, fmt.Errorf("marshal log entry: %w", err)
	}

	keys := []string{
		"bank:stocks",
		"wallet:" + walletID,
		"audit:log",
	}

	result, err := buyScript.Run(ctx, s.rdb, keys, stockName, string(payload)).Int()
	if err != nil {
		return 0, fmt.Errorf("run buy script: %w", err)
	}

	return result, nil
}

var sellScript = redis.NewScript(`
-- Sell script: atomically transfers 1 unit from wallet to bank.
-- KEYS[1]=bank:stocks, KEYS[2]=wallet:<id>, KEYS[3]=audit:log
-- ARGV[1]=stock name, ARGV[2]=JSON log entry
-- Returns: 1=success, 0=wallet empty for this stock, -1=stock unknown

local stock = redis.call("HGET", KEYS[1], ARGV[1])
if not stock then
	return -1
end

local walletStock = redis.call("HGET", KEYS[2], ARGV[1])
if not walletStock or tonumber(walletStock) <= 0 then
	return 0
end

redis.call("HINCRBY", KEYS[2], ARGV[1], -1) -- wallet -1
redis.call("HINCRBY", KEYS[1], ARGV[1], 1)	-- bank +1
redis.call("RPUSH", KEYS[3], ARGV[2])
redis.call("LTRIM", KEYS[3], -10000, -1)
return 1
`)

// Sell mirrors Buy intentionally. "Three strikes, and you refactor" (Fowler, "Refactoring", 1999). :)

// Sell Returns 1 = success, 0 = wallet empty, -1 = stock unknown.
func (s *Store) Sell(ctx context.Context, walletID, stockName string) (int, error) {
	entry := model.LogEntry{
		Type:      "sell",
		WalletID:  walletID,
		StockName: stockName,
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return 0, fmt.Errorf("marshal log entry: %w", err)
	}

	keys := []string{
		"bank:stocks",
		"wallet:" + walletID,
		"audit:log",
	}
	result, err := sellScript.Run(ctx, s.rdb, keys, stockName, string(payload)).Int()
	if err != nil {
		return 0, fmt.Errorf("run sell script: %w", err)
	}

	return result, nil
}

// GetWallet /wallets/{id}
func (s *Store) GetWallet(ctx context.Context, walletID string) ([]model.Stock, error) {
	raw, err := s.rdb.HGetAll(ctx, "wallet:"+walletID).Result()
	if err != nil {
		return nil, fmt.Errorf("get wallet %q: %w", walletID, err)
	}

	walletStocks := make([]model.Stock, 0, len(raw))
	for name, quantityStr := range raw {
		quantity, err := strconv.Atoi(quantityStr)
		if err != nil {
			return nil, fmt.Errorf("parse quantity for %q: %w", name, err)
		}
		walletStocks = append(walletStocks, model.Stock{
			Name:     name,
			Quantity: quantity,
		})
	}
	return walletStocks, nil
}

// GetWalletStock /wallets/{id}/stocks/{name}
func (s *Store) GetWalletStock(ctx context.Context, walletID, stockName string) (int, bool, error) {
	quantity, err := s.rdb.HGet(ctx, "wallet:"+walletID, stockName).Int()
	if errors.Is(err, redis.Nil) {
		return 0, false, nil // there's no stock, but it's not an error
	}
	if err != nil {
		return 0, false, fmt.Errorf("get wallet stock %q: %w", stockName, err)
	}

	return quantity, true, nil
}

// GetLog /log
func (s *Store) GetLog(ctx context.Context) ([]model.LogEntry, error) {
	rawString, err := s.rdb.LRange(ctx, "audit:log", 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("get logs: %w", err)
	}

	entries := make([]model.LogEntry, 0, len(rawString))
	for _, str := range rawString {
		var entry model.LogEntry
		if err := json.Unmarshal([]byte(str), &entry); err != nil {
			return nil, fmt.Errorf("unmarshal log entry: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *Store) Close() error {
	return s.rdb.Close()
}
