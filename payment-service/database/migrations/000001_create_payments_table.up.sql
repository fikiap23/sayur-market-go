CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    payment_status VARCHAR(50) NOT NULL,
    payment_gateway_id VARCHAR(50) NULL,
    gross_amount DECIMAL(10, 2) NOT NULL,
    payment_url TEXT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE NULL
);

CREATE INDEX idx_payments_deleted_at ON payments(deleted_at);
