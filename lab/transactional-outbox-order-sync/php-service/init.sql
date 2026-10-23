-- Runs once per MySQL container (against $MYSQL_DATABASE) via
-- docker-entrypoint-initdb.d. Idempotent so re-runs are harmless.

CREATE TABLE IF NOT EXISTS orders (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    order_number VARCHAR(64) NOT NULL,
    customer_name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'new',
    source VARCHAR(16) NOT NULL DEFAULT 'local',   -- 'local' or 'synced'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_order_number (order_number)      -- idempotency key for sync redelivery
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS outbox (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    aggregate_type VARCHAR(64) NOT NULL DEFAULT 'Order',
    aggregate_id VARCHAR(64) NOT NULL,             -- order_number
    type VARCHAR(64) NOT NULL,                     -- 'OrderCreated'
    payload JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP NULL DEFAULT NULL,      -- NULL = pending
    processing_attempts INT UNSIGNED NOT NULL DEFAULT 0,
    KEY idx_outbox_pending (processed_at, created_at),
    KEY idx_outbox_aggregate (aggregate_type, aggregate_id)
) ENGINE=InnoDB;
