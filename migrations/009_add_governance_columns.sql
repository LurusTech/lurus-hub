-- Migration: Add governance columns to logs table
-- These columns enable indexed queries for data flow tracing, anomaly detection,
-- and latency analysis. GORM auto-migrate will also add them, but this SQL
-- serves as the auditable migration record.
--
-- PostgreSQL 11+ ADD COLUMN ... DEFAULT is a metadata-only operation (no table rewrite).

ALTER TABLE logs ADD COLUMN IF NOT EXISTS channel_type        integer      NOT NULL DEFAULT 0;
ALTER TABLE logs ADD COLUMN IF NOT EXISTS relay_mode          integer      NOT NULL DEFAULT 0;
ALTER TABLE logs ADD COLUMN IF NOT EXISTS request_fingerprint varchar(16)  NOT NULL DEFAULT '';
ALTER TABLE logs ADD COLUMN IF NOT EXISTS upstream_model      varchar(128) NOT NULL DEFAULT '';
ALTER TABLE logs ADD COLUMN IF NOT EXISTS total_latency_ms    integer      NOT NULL DEFAULT 0;

-- Indexes for governance queries
CREATE INDEX IF NOT EXISTS idx_gov_channel_type ON logs (channel_type);
CREATE INDEX IF NOT EXISTS idx_gov_relay_mode   ON logs (relay_mode);
CREATE INDEX IF NOT EXISTS idx_gov_fingerprint  ON logs (request_fingerprint);
