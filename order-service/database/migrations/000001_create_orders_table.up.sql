CREATE TABLE IF NOT EXISTS "orders" (
    id SERIAL PRIMARY KEY,
    order_code VARCHAR(64) UNIQUE NOT NULL,
    buyer_id BIGINT NOT NULL,
    order_date DATE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_amount DECIMAL(10, 2) NOT NULL DEFAULT 0,
    shipping_type VARCHAR(20) NOT NULL DEFAULT 'PICKUP',
    shipping_fee DECIMAL(10, 2) NOT NULL DEFAULT 0,
    order_time TIME NULL,
    remarks text NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_order_code ON orders(order_code);