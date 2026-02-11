-- 005_create_releases_tables.sql
-- Create tables for release and download management system

-- ============================================================
-- Table: releases
-- Purpose: Store release versions for different products
-- ============================================================
CREATE TABLE IF NOT EXISTS releases (
    id BIGSERIAL PRIMARY KEY,
    product_id VARCHAR(50) NOT NULL,
    version VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    changelog_md TEXT,
    release_type VARCHAR(20) DEFAULT 'stable' CHECK (release_type IN ('stable', 'beta', 'alpha')),
    is_draft BOOLEAN DEFAULT true,
    is_prerelease BOOLEAN DEFAULT false,
    is_published BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    UNIQUE (product_id, version)
);

CREATE INDEX idx_releases_product_id ON releases(product_id);
CREATE INDEX idx_releases_published_at ON releases(published_at DESC);
CREATE INDEX idx_releases_type ON releases(release_type);

-- ============================================================
-- Table: release_artifacts
-- Purpose: Store downloadable files for each release
-- ============================================================
CREATE TABLE IF NOT EXISTS release_artifacts (
    id BIGSERIAL PRIMARY KEY,
    release_id BIGINT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    platform VARCHAR(20) NOT NULL CHECK (platform IN ('windows', 'darwin', 'linux', 'android', 'ios')),
    arch VARCHAR(20) NOT NULL CHECK (arch IN ('x64', 'arm64', 'amd64', 'universal')),
    filename VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100),
    storage_path VARCHAR(500) NOT NULL,
    checksum_sha256 VARCHAR(64) NOT NULL,
    download_count BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (release_id, platform, arch)
);

CREATE INDEX idx_artifacts_release_id ON release_artifacts(release_id);
CREATE INDEX idx_artifacts_platform ON release_artifacts(platform);

-- ============================================================
-- Table: download_logs
-- Purpose: Track download events for analytics (GDPR compliant)
-- ============================================================
CREATE TABLE IF NOT EXISTS download_logs (
    id BIGSERIAL PRIMARY KEY,
    artifact_id BIGINT NOT NULL REFERENCES release_artifacts(id) ON DELETE CASCADE,
    ip_address INET,
    user_agent TEXT,
    referer TEXT,
    country_code VARCHAR(2),
    status VARCHAR(20) DEFAULT 'initiated' CHECK (status IN ('initiated', 'completed', 'failed')),
    downloaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_download_logs_artifact_id ON download_logs(artifact_id);
CREATE INDEX idx_download_logs_downloaded_at ON download_logs(downloaded_at);

-- ============================================================
-- Triggers for updated_at columns
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_releases_updated_at BEFORE UPDATE ON releases
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_release_artifacts_updated_at BEFORE UPDATE ON release_artifacts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- Comments for documentation
-- ============================================================
COMMENT ON TABLE releases IS 'Product release versions and metadata';
COMMENT ON TABLE release_artifacts IS 'Platform-specific downloadable files';
COMMENT ON TABLE download_logs IS 'Download tracking (90-day retention for GDPR)';

COMMENT ON COLUMN releases.product_id IS 'Product identifier (lurus-switch, lurus-cli)';
COMMENT ON COLUMN releases.version IS 'Semantic version (1.0.0)';
COMMENT ON COLUMN releases.release_type IS 'Release stability level';
COMMENT ON COLUMN release_artifacts.storage_path IS 'MinIO object path';
COMMENT ON COLUMN release_artifacts.checksum_sha256 IS 'File integrity verification';
COMMENT ON COLUMN download_logs.ip_address IS 'Client IP (anonymized after 30 days per GDPR)';
