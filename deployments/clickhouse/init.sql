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

CREATE TABLE IF NOT EXISTS logs.gateway_access
(
    timestamp              DateTime64(3),
    status                 UInt16,
    method                 LowCardinality(String),
    path                   String,
    upstream               LowCardinality(String) DEFAULT '',
    upstream_addr          String DEFAULT '',
    upstream_status        LowCardinality(String) DEFAULT '',
    request_time           Float32 DEFAULT 0,
    upstream_response_time Float32 DEFAULT 0,
    request_id             String DEFAULT '',
    remote_addr            String DEFAULT '',
    user_agent             String DEFAULT '',

    INDEX idx_request_id request_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_status     status     TYPE set(0)       GRANULARITY 1,
    INDEX idx_upstream   upstream   TYPE set(0)       GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (upstream, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;
