-- Orders for confirmed holds
CREATE TABLE IF NOT EXISTS orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hold_id         UUID NOT NULL REFERENCES holds(id) ON DELETE CASCADE,
    idempotency_key TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS orders_hold_id_unique ON orders(hold_id);
