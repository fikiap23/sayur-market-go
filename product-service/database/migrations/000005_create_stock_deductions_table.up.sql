-- Records applied stock deductions for idempotent consumption of order messages.
CREATE TABLE IF NOT EXISTS stock_deductions (
    id BIGSERIAL PRIMARY KEY,
    dedup_key VARCHAR(512) NOT NULL UNIQUE,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_deductions_product_id ON stock_deductions(product_id);
