-- 006_seed_releases.sql
-- Seed initial release data for lurus-switch and lurus-cli
-- Run AFTER AutoMigrate creates the tables

-- ============================================================
-- Lurus Switch v1.0.0 (Stable)
-- ============================================================
INSERT INTO releases (product_id, version, title, description, changelog_md, release_type, is_draft, is_prerelease, is_published, published_at)
VALUES (
    'lurus-switch',
    '1.0.0',
    'Lurus Switch v1.0.0',
    'Initial stable release with multi-provider API support',
    '## New Features

- Multi-provider API gateway support
- Cross-platform desktop application (Windows, macOS, Linux)
- Token management and usage tracking
- Real-time API status monitoring
- Customizable routing rules

## Improvements

- Optimized startup performance
- Enhanced error handling
- Improved UI/UX design

## Bug Fixes

- Fixed token expiration handling
- Resolved memory leak in long-running sessions
- Corrected Windows installer path issues',
    'stable',
    false,
    false,
    true,
    '2026-02-01 00:00:00'
) ON CONFLICT (product_id, version) DO NOTHING;

-- Artifacts for Switch v1.0.0
INSERT INTO release_artifacts (release_id, platform, arch, filename, file_size, mime_type, storage_path, checksum_sha256)
SELECT r.id, v.platform, v.arch, v.filename, v.file_size, v.mime_type, v.storage_path, v.checksum_sha256
FROM releases r,
(VALUES
    ('windows', 'x64', 'lurus-switch-windows-x64-v1.0.0.exe', 47185920, 'application/octet-stream', 'lurus-switch/v1.0.0/lurus-switch-windows-x64-v1.0.0.exe', 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'),
    ('darwin', 'arm64', 'lurus-switch-macos-arm64-v1.0.0.dmg', 52428800, 'application/x-apple-diskimage', 'lurus-switch/v1.0.0/lurus-switch-macos-arm64-v1.0.0.dmg', 'a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2'),
    ('darwin', 'x64', 'lurus-switch-macos-x64-v1.0.0.dmg', 52428800, 'application/x-apple-diskimage', 'lurus-switch/v1.0.0/lurus-switch-macos-x64-v1.0.0.dmg', 'b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3'),
    ('linux', 'x64', 'lurus-switch-linux-x64-v1.0.0.tar.gz', 44040192, 'application/gzip', 'lurus-switch/v1.0.0/lurus-switch-linux-x64-v1.0.0.tar.gz', 'c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4')
) AS v(platform, arch, filename, file_size, mime_type, storage_path, checksum_sha256)
WHERE r.product_id = 'lurus-switch' AND r.version = '1.0.0'
ON CONFLICT (release_id, platform, arch) DO NOTHING;

-- ============================================================
-- Lurus CLI v0.5.0 (Stable)
-- ============================================================
INSERT INTO releases (product_id, version, title, description, changelog_md, release_type, is_draft, is_prerelease, is_published, published_at)
VALUES (
    'lurus-cli',
    '0.5.0',
    'Lurus CLI v0.5.0',
    'Terminal-based UI for quick API access',
    '## New Features

- Interactive TUI with keyboard navigation
- Quick API key switching
- Real-time usage monitoring
- Export usage reports to CSV

## Improvements

- Faster startup time
- Reduced memory footprint
- Better error messages',
    'stable',
    false,
    false,
    true,
    '2026-01-20 00:00:00'
) ON CONFLICT (product_id, version) DO NOTHING;

-- Artifacts for CLI v0.5.0
INSERT INTO release_artifacts (release_id, platform, arch, filename, file_size, mime_type, storage_path, checksum_sha256)
SELECT r.id, v.platform, v.arch, v.filename, v.file_size, v.mime_type, v.storage_path, v.checksum_sha256
FROM releases r,
(VALUES
    ('windows', 'x64', 'lurus-cli-windows-x64-v0.5.0.exe', 8388608, 'application/octet-stream', 'lurus-cli/v0.5.0/lurus-cli-windows-x64-v0.5.0.exe', 'd4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5'),
    ('darwin', 'universal', 'lurus-cli-macos-universal-v0.5.0', 10485760, 'application/octet-stream', 'lurus-cli/v0.5.0/lurus-cli-macos-universal-v0.5.0', 'e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6'),
    ('linux', 'x64', 'lurus-cli-linux-x64-v0.5.0', 7340032, 'application/octet-stream', 'lurus-cli/v0.5.0/lurus-cli-linux-x64-v0.5.0', 'f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1')
) AS v(platform, arch, filename, file_size, mime_type, storage_path, checksum_sha256)
WHERE r.product_id = 'lurus-cli' AND r.version = '0.5.0'
ON CONFLICT (release_id, platform, arch) DO NOTHING;
