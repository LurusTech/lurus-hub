-- Migration: create switch_config_presets table
-- Purpose: store cloud-hosted configuration templates for AI CLI tools (lurus-switch)
-- Schema: lurus_api

CREATE TABLE IF NOT EXISTS switch_config_presets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool        VARCHAR(32)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    description TEXT,
    category    VARCHAR(64),
    config_json JSONB        NOT NULL,
    is_official BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_switch_presets_tool
    ON switch_config_presets (tool, is_official);
