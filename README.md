# Simplified Stock Market

REST API simulating a simplified stock market.

## Requirements

- Go 1.21+
- Docker (for Redis)

## Running

```bash
docker compose up -d          # start Redis
go build -o server ./cmd/server && ./server -port 8080
```

## Testing

```bash
# Start Redis and server first
docker compose up -d
go build -o server ./cmd/server && ./server -port 8080 &

# Run all tests (sequentially to avoid Redis conflicts)
go test -p 1 ./...
```

The `-p 1` flag runs test packages sequentially. This is required because store tests (Redis DB 15) and e2e tests (server on DB 0) can conflict when run in parallel.

Alternatively, run separately:
```bash
go test ./internal/store/...   # unit tests
go test ./test/e2e/...         # e2e tests (requires running server)
```
