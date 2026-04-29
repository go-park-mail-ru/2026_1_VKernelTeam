CREATE DATABASE IF NOT EXISTS logs;

CREATE TABLE IF NOT EXISTS logs.service_logs
(
    timestamp    DateTime64(3),
    level        LowCardinality(String),
    service      LowCardinality(String),
    message      String,
    request_id   String DEFAULT '',
    attributes   String DEFAULT '{}',

    INDEX idx_request_id request_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_level level TYPE set(0) GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (service, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;
