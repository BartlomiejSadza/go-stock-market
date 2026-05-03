# Simplified Stock Market

REST API for a simplified stock exchange. Stocks are traded against a single
Bank at a fixed price of 1; wallet balances and order books are not modeled.
High-availability via two stateless Go instances behind nginx, with Redis as
shared state.

## Run

```bash
docker compose up -d                # default http://localhost:8080
# or
PORT=9000 docker compose up -d      # custom host port
```
`PORT` only changes the host-side mapping — internal container ports are fixed.

## Endpoints

| Method | Path                          | Notes                                            |
|--------|-------------------------------|--------------------------------------------------|
| `GET`  | `/stocks`                     | bank state                                       |
| `POST` | `/stocks`                     | overwrite bank state (not logged)                |
| `POST` | `/wallets/{id}/stocks/{name}` | body: `{"type":"buy"\|"sell"}` — 200 / 400 / 404 |
| `GET`  | `/wallets/{id}`               | wallet state (empty for unused id)               |
| `GET`  | `/wallets/{id}/stocks/{name}` | quantity as a bare number                        |
| `GET`  | `/log`                        | audit log of successful wallet operations        |
| `POST` | `/chaos`                      | kills the instance handling the request          |
| `GET`  | `/healthz`                    | liveness probe                                   |

## Architecture

`client` --> `nginx (LB)` --> `2x Go app` --> `Redis`

- App instances are stateless; all state lives in Redis.
- Buy/sell is a single Redis Lua script: bank check + decrement, wallet increment, and audit append happen atomically across instances.
- Audit log is capped at 10,000 entries (LTRIM after RPUSH).

### Verifying HA

```bash
docker compose up -d
curl -X POST http://localhost:8080/chaos    # kill one instance
curl http://localhost:8080/healthz          # still 200, served by the survivor

# (Replace 8080 with your PORT if customized.)
```

Compose restarts the killed container automatically (`restart: unless-stopped`).

## Testing

Tests need Redis on `localhost:6379` (DB 15, flushed before each test).

```bash
docker compose up -d redis
go test ./...
```

E2e tests run against an in-process `httptest.Server` — no full stack required.

## Layout

```
.
├── cmd/server/         # entry point, signal handling, graceful shutdown
├── internal/
│   ├── router/         # HTTP routes (single source of truth)
│   ├── handler/        # HTTP handlers
│   ├── store/          # Redis access; Lua scripts for atomic buy/sell
│   └── model/          # JSON request/response types
└── test/e2e/           # black-box API tests
```

## A note on AI assistance

Claude was used as a mentor and reviewer throughout this project —
explaining Go internals, suggesting design trade-offs, and reviewing changes
step by step.

**Every line of source code in this repository was written by
hand, by me** — no code was AI-generated :)

(except this markdown 😁)