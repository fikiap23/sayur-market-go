CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    parent_id BIGINT NULL,
    category_slug VARCHAR(120) NULL REFERENCES categories(slug) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    image VARCHAR(255) NOT NULL,
    description TEXT NULL,
    reguler_price BIGINT DEFAULT 0,
    sale_price BIGINT DEFAULT 0,
    unit varchar(120) DEFAULT 'gram',
    weight BIGINT DEFAULT 0,
    variant INT DEFAULT 1,
    status varchar(20) DEFAULT 'DRAFT',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_products_status ON products(status);
CREATE INDEX idx_products_category_slug ON products(category_slug);