# orders-service

Places, fetches and lists orders. Go, with Postgres.

## API

| Method | Path | What |
|---|---|---|
| POST | `/orders` | Place an order: `{"customer_id", "currency", "amount_cents"}` |
| GET | `/orders/{id}` | One order |
| GET | `/orders?customer_id=…` | A customer's orders, newest first |
| GET | `/healthz` | Liveness |

## Run

```
DATABASE_URL=postgres://… go run ./cmd/orders-service
```

`ADDR` sets the listen address; the default is `:8080`.

## Test

```
go test -short ./...   # without Docker
go test ./...          # store tests start Postgres in Docker
```
