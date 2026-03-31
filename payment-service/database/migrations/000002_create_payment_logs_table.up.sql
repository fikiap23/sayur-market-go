CREATE TABLE IF NOT EXISTS payment_logs (
    id BIGSERIAL PRIMARY KEY,
    payment_id BIGINT NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_payment_logs_payment_id ON payment_logs(payment_id);
