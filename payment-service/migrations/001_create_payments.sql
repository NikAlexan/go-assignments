CREATE TABLE IF NOT EXISTS payments (
    id             VARCHAR(36) PRIMARY KEY,
    order_id       VARCHAR(36) NOT NULL UNIQUE,
    transaction_id VARCHAR(36),
    amount         BIGINT      NOT NULL,
    status         VARCHAR(20) NOT NULL
);
