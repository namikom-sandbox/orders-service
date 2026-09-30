CREATE TABLE IF NOT EXISTS orders (
    id           BIGSERIAL   PRIMARY KEY,
    customer_id  TEXT        NOT NULL,
    currency     TEXT        NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    amount_cents BIGINT      NOT NULL CHECK (amount_cents > 0),
    status       TEXT        NOT NULL DEFAULT 'placed',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS orders_customer_created_idx
    ON orders (customer_id, created_at DESC);
