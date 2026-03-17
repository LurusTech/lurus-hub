package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// toolManifestEntry describes a single tool in the download manifest.
type toolManifestEntry struct {
	Type          string                        `json:"type"`
	NpmPackage    string                        `json:"npm_package,omitempty"`
	LatestVersion string                        `json:"latest_version"`
	Platforms     map[string]toolManifestAsset  `json:"platforms,omitempty"`
}

// toolManifestAsset holds the download URL and optional checksum for one platform.
type toolManifestAsset struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256,omitempty"`
}

// toolManifestResponse is the top-level API response.
type toolManifestResponse struct {
	GeneratedAt string                       `json:"generated_at"`
	Tools       map[string]toolManifestEntry `json:"tools"`
}

// toolManifestOnce ensures the static manifest is built exactly once at startup.
var (
	toolManifestOnce sync.Once
	cachedManifest   toolManifestResponse
)

// buildToolManifest constructs the static manifest data.
// Platform keys use the format "os/arch" (e.g. "windows/amd64", "darwin/arm64").
// Binary URLs follow the GitHub Releases download pattern; update these when new
// releases are published (CI tooling can automate the version bump).
func buildToolManifest() toolManifestResponse {
	const (
		picoBase  = "https://github.com/picoclaw-labs/picoclaw/releases/download"
		nullBase  = "https://github.com/nullclaw-labs/nullclaw/releases/download"
		zeroBase  = "https://github.com/zeroclaw-labs/zeroclaw/releases/download"
		creatorBase = "https://github.com/lurus-dev/lurus-creator/releases/download"

		picoVer   = "v1.0.0"
		nullVer   = "v1.0.0"
		zeroVer   = "v1.0.0"
		creatorVer = "v0.3.0"
	)

	return toolManifestResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Tools: map[string]toolManifestEntry{
			// ── npm tools ──────────────────────────────────────────────
			"claude": {
				Type:          "npm",
				NpmPackage:    "@anthropic-ai/claude-code",
				LatestVersion: "1.0.26",
			},
			"codex": {
				Type:          "npm",
				NpmPackage:    "@openai/codex",
				LatestVersion: "0.1.2505161344",
			},
			"gemini": {
				Type:          "npm",
				NpmPackage:    "@google/gemini-cli",
				LatestVersion: "0.1.9",
			},
			"openclaw": {
				Type:          "npm",
				NpmPackage:    "openclaw",
				LatestVersion: "1.0.0",
			},

			// ── binary tools ───────────────────────────────────────────
			"picoclaw": {
				Type:          "binary",
				LatestVersion: picoVer,
				Platforms: map[string]toolManifestAsset{
					"windows/amd64": {URL: picoBase + "/" + picoVer + "/pclaw-x86_64-pc-windows-msvc.zip"},
					"windows/arm64": {URL: picoBase + "/" + picoVer + "/pclaw-x86_64-pc-windows-msvc.zip"}, // x64 compat
					"darwin/arm64":  {URL: picoBase + "/" + picoVer + "/pclaw-aarch64-apple-darwin.tar.gz"},
					"darwin/amd64":  {URL: picoBase + "/" + picoVer + "/pclaw-x86_64-apple-darwin.tar.gz"},
					"linux/amd64":   {URL: picoBase + "/" + picoVer + "/pclaw-x86_64-unknown-linux-musl.tar.gz"},
					"linux/arm64":   {URL: picoBase + "/" + picoVer + "/pclaw-aarch64-unknown-linux-musl.tar.gz"},
				},
			},
			"nullclaw": {
				Type:          "binary",
				LatestVersion: nullVer,
				Platforms: map[string]toolManifestAsset{
					"windows/amd64": {URL: nullBase + "/" + nullVer + "/nclaw-x86_64-pc-windows-msvc.zip"},
					"windows/arm64": {URL: nullBase + "/" + nullVer + "/nclaw-x86_64-pc-windows-msvc.zip"},
					"darwin/arm64":  {URL: nullBase + "/" + nullVer + "/nclaw-aarch64-apple-darwin.tar.gz"},
					"darwin/amd64":  {URL: nullBase + "/" + nullVer + "/nclaw-x86_64-apple-darwin.tar.gz"},
					"linux/amd64":   {URL: nullBase + "/" + nullVer + "/nclaw-x86_64-unknown-linux-musl.tar.gz"},
					"linux/arm64":   {URL: nullBase + "/" + nullVer + "/nclaw-aarch64-unknown-linux-musl.tar.gz"},
				},
			},
			"zeroclaw": {
				Type:          "binary",
				LatestVersion: zeroVer,
				Platforms: map[string]toolManifestAsset{
					"windows/amd64": {URL: zeroBase + "/" + zeroVer + "/zeroclaw-x86_64-pc-windows-msvc.zip"},
					"windows/arm64": {URL: zeroBase + "/" + zeroVer + "/zeroclaw-x86_64-pc-windows-msvc.zip"},
					"darwin/arm64":  {URL: zeroBase + "/" + zeroVer + "/zeroclaw-aarch64-apple-darwin.tar.gz"},
					"darwin/amd64":  {URL: zeroBase + "/" + zeroVer + "/zeroclaw-x86_64-apple-darwin.tar.gz"},
					"linux/amd64":   {URL: zeroBase + "/" + zeroVer + "/zeroclaw-x86_64-unknown-linux-musl.tar.gz"},
					"linux/arm64":   {URL: zeroBase + "/" + zeroVer + "/zeroclaw-aarch64-unknown-linux-musl.tar.gz"},
				},
			},

			// ── desktop installer ──────────────────────────────────────
			"lurus-creator": {
				Type:          "desktop",
				LatestVersion: creatorVer,
				Platforms: map[string]toolManifestAsset{
					"windows/amd64": {URL: creatorBase + "/" + creatorVer + "/lurus-creator-setup-" + creatorVer + "-windows-x64.exe"},
					"windows/arm64": {URL: creatorBase + "/" + creatorVer + "/lurus-creator-setup-" + creatorVer + "-windows-x64.exe"},
					"darwin/arm64":  {URL: creatorBase + "/" + creatorVer + "/lurus-creator-" + creatorVer + "-arm64.dmg"},
					"darwin/amd64":  {URL: creatorBase + "/" + creatorVer + "/lurus-creator-" + creatorVer + "-x64.dmg"},
				},
			},
		},
	}
}

// GetToolDownloadManifest serves the static tool download manifest.
// No authentication required. Responses are cacheable for 1 hour.
//
// GET /api/v2/tools/download-manifest
func GetToolDownloadManifest(c *gin.Context) {
	toolManifestOnce.Do(func() {
		cachedManifest = buildToolManifest()
	})

	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, cachedManifest)
}
