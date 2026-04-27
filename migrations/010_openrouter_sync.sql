-- Migration: OpenRouter free-model sync feature
--
-- Adds two new tables for the sync engine, two columns to channels for
-- per-channel sync state (managed model set + circuit-breaker baseline),
-- and a composite index on logs to support the hourly rank aggregator.
--
-- GORM auto-migrate also creates these on master startup; this SQL serves
-- as the auditable migration record and lets ops run it ahead of time.

-- 1. Channels: state for sync engine (idempotent ADD COLUMN IF NOT EXISTS)
ALTER TABLE channels ADD COLUMN IF NOT EXISTS managed_models_by_sync text    NOT NULL DEFAULT '';
ALTER TABLE channels ADD COLUMN IF NOT EXISTS last_sync_fetch_count  integer NOT NULL DEFAULT 0;

-- 2. OpenRouter sync jobs (per-rule config)
CREATE TABLE IF NOT EXISTS openrouter_sync_jobs (
    id                 SERIAL PRIMARY KEY,
    name               varchar(128) NOT NULL,
    target_channel_id  integer      NOT NULL,
    categories         text         NOT NULL DEFAULT '',
    top_n              integer      NOT NULL DEFAULT 0,
    schedule           varchar(32)  NOT NULL DEFAULT 'manual',
    enabled            boolean      NOT NULL DEFAULT true,
    last_run_at        timestamptz,
    last_error         text         NOT NULL DEFAULT '',
    created_at         timestamptz  NOT NULL DEFAULT NOW(),
    updated_at         timestamptz  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_openrouter_sync_jobs_channel ON openrouter_sync_jobs (target_channel_id);

-- 3. Model usage stats (hourly pre-aggregated rank input)
CREATE TABLE IF NOT EXISTS model_usage_stats (
    model_name        varchar(128) NOT NULL,
    channel_type      integer      NOT NULL,
    count_24h         bigint       NOT NULL DEFAULT 0,
    last_updated_at   timestamptz  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (model_name, channel_type)
);
CREATE INDEX IF NOT EXISTS idx_model_usage_stats_updated ON model_usage_stats (last_updated_at);

-- 4. Composite index on logs for the rank aggregator's GROUP BY query
--    (channel_type, created_at) supports: WHERE channel_type=? AND created_at>?  GROUP BY model_name
--    The existing idx_gov_channel_type only indexes channel_type; this index lets PG
--    range-scan the recent 24h slice without a full table scan.
CREATE INDEX IF NOT EXISTS idx_logs_channel_created ON logs (channel_type, created_at);
