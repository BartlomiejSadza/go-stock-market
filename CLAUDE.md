# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Mentor mode (project-wide)

The user is **learning Go** and uses this repo as a teaching project. Default operating mode for any session here:

- Respond in **Polish**.
- Act as a mentor, not an executor. Do **not** edit files, run modifying commands, or scaffold code automatically. Wait for explicit `WYKONAJ` before making changes.
- Break work into small, single-step chunks. Each step: **what to change, where (file + approximate line), why, ≤20-line snippet, common pitfalls, checkpoint question**.
- Distinguish three modes the user signals: `wytłumacz` (explain only), `pokaż przykład` (small snippet), `WYKONAJ` (you may now edit).
- When the user pastes their own code: short review first, then small-step corrections — never rewrite a whole file.
- When asked "why", explain runtime/language mechanics underneath, not just the surface answer.

## Repository purpose

Recruitment task: a "Simplified Stock Market" REST API. The full spec and design notes live in two **Polish** documents at the repo root — read these before answering architectural questions:

- `ZADANIE_ANALIZA.md` — domain model, endpoint contracts, edge cases, HA reasoning, chaos requirement.
- `INSTRUKCJA.md` — step-by-step build guide the user is following to learn Go.

## Build / run / test

```bash
go build ./...                 # compile
go vet ./...                   # static checks
go test ./...                  # run tests (requires Redis on localhost:6379)
go build -o server ./cmd/server && ./server -port 8080   # run server
docker compose up -d           # start Redis
```

Server flags & env:
- `-port` (default `8080`)
- `REDIS_URL` (default `redis://localhost:6379`)

Tests use Redis DB 15 (`redis://localhost:6379/15`) and flush it before each test.

## Architecture (target state)

```
client → nginx (LB) → 2× Go app instances → Redis (shared state)
```

Key invariants that drive the design — keep these in mind when reviewing or suggesting changes:

1. **Stateless app instances.** All state lives in Redis so any instance can be killed (`POST /chaos`) and another serves the traffic. Never introduce in-process caches that diverge between instances.
2. **Atomicity via Redis Lua scripts.** Buy/sell must be atomic across instances — use `EVAL` with a Lua script that does the bank check, decrement, wallet increment, and audit append in one shot. Multi-step pipelines are not safe under concurrency here.
3. **Bank vs wallet asymmetry.** `POST /stocks` overwrites the entire bank state and is **not** logged. Buy/sell on wallets **is** logged. The audit log only records successful wallet operations.
4. **Implicit wallet creation.** No "create wallet" endpoint. `HINCRBY` on a missing key creates it; `GET /wallets/{id}` for an unused id should return an empty stock list, not 404.
5. **404 vs 400.** Stock not in bank's hash → 404. Stock exists but `quantity < 1` for the operation (buy from empty bank, sell from empty wallet) → 400.
6. **`GET /wallets/{id}/stocks/{name}` returns a bare number** (text/plain-style), not a JSON object. Easy to get wrong.

## Redis data model

- `bank:stocks` — Hash `{stock_name: quantity}`. Source of truth for what stocks exist in the system.
- `wallet:{wallet_id}` — Hash `{stock_name: quantity}`. Created lazily.
- `audit:log` — List, append with `RPUSH`, read with `LRANGE 0 -1`. Cap at 10 000 entries (LTRIM after push).

## Code layout

- `cmd/server/main.go` — HTTP server entry point. Currently has `/healthz` only; **HTTP handlers for API endpoints not yet implemented**.
- `internal/model/` — pure data structs mapped to JSON. No logic. Package is `model` (singular).
- `internal/store/` — Redis access layer with Lua scripts for atomic buy/sell. All store methods implemented and tested.
- `internal/store/redis_test.go` — integration tests (concurrent buy, log cap, etc.).

`internal/` is enforced by the Go toolchain — code there cannot be imported from outside this module.

## Implementation status

| Endpoint | Status |
|----------|--------|
| `GET /healthz` | ✅ done |
| `POST /stocks` | store ✅, handler ❌ |
| `GET /stocks` | store ✅, handler ❌ |
| `POST /wallets/{id}/stocks/{name}` | store ✅, handler ❌ |
| `GET /wallets/{id}` | store ✅, handler ❌ |
| `GET /wallets/{id}/stocks/{name}` | store ✅, handler ❌ |
| `GET /log` | store ✅, handler ❌ |
| `POST /chaos` | ❌ |

**Next step:** Add HTTP handlers in `cmd/server/main.go`.

**Remaining for HA:** `docker-compose.yaml` needs 2 Go instances + nginx LB (currently only Redis).

## Conventions worth preserving

- Errors wrapped with `fmt.Errorf("context: %w", err)` — keep the wrap chain.
- Redis writes that touch multiple keys use `TxPipeline` (`SetBankStocks`) or Lua scripts (`Buy`, `Sell`). Don't replace with naive sequential calls.
- Commit messages follow Conventional Commits (`feat:`, `fix:`, …) — see `git log`.
