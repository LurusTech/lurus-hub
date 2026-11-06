-- Migration: multi-provider catalog inventory sync + gated price proposals
--
-- Daily CatalogSync (internal/app/catalog_sync) discovers upstream model IDs
-- via official Models APIs, auto-adds to channels/abilities, marks stale
-- removals for human confirm, and stores price proposals (never auto-applied).
--
-- GORM AutoMigrate also creates these on boot; this SQL is the auditable record.

CREATE TABLE IF NOT EXISTS catalog_sync_runs (
    id              BIGSERIAL PRIMARY KEY,
    channel_id      integer      NOT NULL,
    fetched_total   integer      NOT NULL DEFAULT 0,
    added_count     integer      NOT NULL DEFAULT 0,
    stale_count     integer      NOT NULL DEFAULT 0,
    proposed_prices integer      NOT NULL DEFAULT 0,
    circuit_breaker boolean      NOT NULL DEFAULT false,
    error           text         NOT NULL DEFAULT '',
    source_version  varchar(64)  NOT NULL DEFAULT '',
    created_at      timestamptz  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_catalog_sync_runs_channel ON catalog_sync_runs (channel_id);
CREATE INDEX IF NOT EXISTS idx_catalog_sync_runs_source ON catalog_sync_runs (source_version);

CREATE TABLE IF NOT EXISTS catalog_stale_pending (
    id              BIGSERIAL PRIMARY KEY,
    channel_id      integer      NOT NULL,
    model_name      varchar(191) NOT NULL,
    first_seen_at   timestamptz  NOT NULL DEFAULT NOW(),
    last_seen_at    timestamptz  NOT NULL DEFAULT NOW(),
    resolved_at     timestamptz,
    source_version  varchar(64)  NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_catalog_stale_channel_model
    ON catalog_stale_pending (channel_id, model_name);

CREATE TABLE IF NOT EXISTS catalog_price_proposals (
    id              BIGSERIAL PRIMARY KEY,
    model_name      varchar(191) NOT NULL,
    field           varchar(64)  NOT NULL,
    local_value     double precision NOT NULL DEFAULT 0,
    upstream_value  double precision NOT NULL DEFAULT 0,
    source          varchar(128) NOT NULL DEFAULT '',
    source_version  varchar(64)  NOT NULL DEFAULT '',
    created_at      timestamptz  NOT NULL DEFAULT NOW(),
    applied_at      timestamptz,
    dismissed_at    timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_catalog_price_model_field
    ON catalog_price_proposals (model_name, field);
CREATE INDEX IF NOT EXISTS idx_catalog_price_proposals_source
    ON catalog_price_proposals (source_version);
