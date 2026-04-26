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

The repo is in early scaffolding. Most of the binary, Docker, and test infra do not exist yet — do **not** invent commands.

Currently usable:

```bash
go build ./...        # compile everything that exists
go vet ./...          # static checks
go mod tidy           # sync go.sum (note: go.sum is currently gitignored implicitly — actually untracked)
```

Planned (per `INSTRUKCJA.md`, not yet implemented — verify before claiming they exist):

- `cmd/server/` entry point with `-port` flag and `REDIS_URL` env var.
- `docker compose up` bringing up Redis + 2 Go instances + nginx LB.
- Tests under `internal/**/_test.go` runnable with `go test ./...`.

When the user adds these, update this section rather than assuming.

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

- `internal/model/` — pure data structs mapped to JSON. No logic. Note: package is `model` (singular); the dir was recently renamed from `models/` — don't reintroduce the plural.
- `internal/store/` — Redis access layer. Wraps `*redis.Client`, exposes domain-shaped methods (e.g. `SetBankStocks`). Lua scripts for buy/sell will live here.
- Handlers and `cmd/server` do not exist yet.

`internal/` is enforced by the Go toolchain — code there cannot be imported from outside this module. That is intentional encapsulation, not a stylistic choice.

## Conventions worth preserving

- Errors wrapped with `fmt.Errorf("context: %w", err)` — keep the wrap chain.
- Redis writes that touch multiple keys use `TxPipeline` (current `SetBankStocks`) or Lua (planned for buy/sell). Don't replace either with naive sequential calls.
- Commit messages follow Conventional Commits (`feat:`, `fix:`, …) — see `git log`.
