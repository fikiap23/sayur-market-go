CREATE TABLE IF NOT EXISTS categories (
    id BIGSERIAL PRIMARY KEY,
    parent_id BIGINT NULL,
    name VARCHAR(100) NOT NULL,
    icon VARCHAR(255) NOT NULL,
    status BOOLEAN DEFAULT TRUE,
    slug VARCHAR(120) UNIQUE NOT NULL,
    description TEXT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    deleted_at TIMESTAMP NULL,

    CONSTRAINT fk_categories_parent
    FOREIGN KEY (parent_id) REFERENCES categories(id)
    ON DELETE SET NULL
);

CREATE INDEX idx_categories_status ON categories(status);
CREATE INDEX idx_categories_slug ON categories(slug);