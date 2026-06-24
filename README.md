# openbank-funds

[![CI](https://github.com/Sreenivas-Sadhu-Prabhakara/openbank-funds/actions/workflows/ci.yml/badge.svg)](https://github.com/Sreenivas-Sadhu-Prabhakara/openbank-funds/actions/workflows/ci.yml) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE) [![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)

The **Funds (CBPII)** microservice — *Confirmation of Funds*, part of the BIAN *Customer Position* service domain, exposing the OBIE **CBPII** funds-confirmation API.

It validates an authorised `funds-confirmation` consent (against the consent service), reads the debtor account from that consent, and asks the **accounts** service whether sufficient funds are available — accounts is the single source of truth for balances, so this service never stores balance data.

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| POST | `/funds-confirmations` | Confirm funds availability for an amount |
| GET | `/health` | Liveness |

Response carries `Data.FundsAvailableResult.{FundsAvailable, FundsAvailableDateTime}`. Unknown consent → 400; wrong type / not `Authorised` → 403; unknown debtor account at the accounts service → 422.

## Configuration

| Env | Default | Notes |
|---|---|---|
| `ADDR` | `:8084` | Listen address |
| `BASE_URL` | `http://localhost:8084` | Used for `Links.Self` |
| `DATABASE_URL` | _(unset)_ | Postgres DSN; **unset → in-memory store** |
| `CONSENT_URL` | `http://localhost:8081` | Consent service base URL |
| `ACCOUNTS_URL` | `http://localhost:8082` | Accounts service base URL (funds availability) |

## Run

```bash
go run .                              # in-memory (requires consent + accounts running for real checks)
docker build -t openbank/funds . && docker run -p 8084:8084 openbank/funds
```

## Test

```bash
go test ./...                       # unit + handler tests (fake consent + accounts clients, no Docker)
go test -tags=integration ./...     # Postgres repo tests via testcontainers (needs Docker)
```

## Layout notes

- `internal/funds/` — domain, `Repository` port (in-memory + Postgres, persists confirmations for audit), service logic, OBIE handler.
- `migrations/` — SQL owned by this service, applied on startup when `DATABASE_URL` is set.
- `pkg/` — vendored shared OBIE library, wired via `replace ... => ./pkg`.
