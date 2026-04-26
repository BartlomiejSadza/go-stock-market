package store

import (
	"context"
	"encoding/json"
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
redis.call("HINCRBY", KEYS[2], ARGV[1], 1   -- increment wallet
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

func (s *Store) Close() error {
	return s.rdb.Close()
}
