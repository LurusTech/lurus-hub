package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LurusTech/lurus-hub/internal/domain/entity"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
	"github.com/redis/go-redis/v9"
)

const (
	// toolVersionRedisKey is the Redis hash key that stores tool → version mappings.
	toolVersionRedisKey = "switch:tool_versions"
	// toolVersionTTL is how long the cached versions remain valid.
	toolVersionTTL = 6 * time.Hour
	// toolVersionPollInterval is how often the background worker re-polls sources.
	toolVersionPollInterval = 6 * time.Hour
	// toolVersionRequestTimeout caps individual HTTP calls to npm / GitHub.
	toolVersionRequestTimeout = 15 * time.Second
)

// npmVersionSource describes how to fetch the latest version of an npm-distributed tool.
type npmVersionSource struct {
	tool    string
	npmPkg  string
}

// githubVersionSource describes how to fetch the latest version of a GitHub-released tool.
type githubVersionSource struct {
	tool  string
	owner string
	repo  string
}

// toolVersionSources lists all tools whose versions the worker monitors.
var npmVersionSources = []npmVersionSource{
	{tool: "claude", npmPkg: "@anthropic-ai/claude-code"},
	{tool: "codex", npmPkg: "@openai/codex"},
	{tool: "gemini", npmPkg: "@google/gemini-cli"},
	{tool: "picoclaw", npmPkg: "picoclaw"},
	{tool: "nullclaw", npmPkg: "nullclaw"},
	{tool: "openclaw", npmPkg: "openclaw"},
}

var githubVersionSources = []githubVersionSource{
	{tool: "zeroclaw", owner: "zeroclaw-labs", repo: "zeroclaw"},
}

// StartToolVersionWorker launches a background goroutine that polls npm / GitHub every 6 hours
// and caches the results in the Redis hash "switch:tool_versions".
// The worker respects context cancellation for graceful shutdown.
func StartToolVersionWorker(ctx context.Context, rdb *redis.Client) {
	go func() {
		// Run immediately on startup so the cache is populated before the first request.
		pollAndCache(ctx, rdb)

		ticker := time.NewTicker(toolVersionPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pollAndCache(ctx, rdb)
			}
		}
	}()
}

// pollAndCache fetches all tool versions and writes them to Redis.
func pollAndCache(ctx context.Context, rdb *redis.Client) {
	client := &http.Client{Timeout: toolVersionRequestTimeout}
	versions := make(map[string]*entity.ToolVersion)

	for _, src := range npmVersionSources {
		v := fetchNpmVersion(ctx, client, src.tool, src.npmPkg)
		if v != nil {
			versions[src.tool] = v
		}
	}

	for _, src := range githubVersionSources {
		v := fetchGitHubVersion(ctx, client, src.tool, src.owner, src.repo)
		if v != nil {
			versions[src.tool] = v
		}
	}

	if len(versions) == 0 {
		return
	}

	// Persist to Redis hash; use HSET + EXPIRE to atomically refresh TTL.
	fields := make(map[string]interface{}, len(versions))
	for tool, ver := range versions {
		raw, err := json.Marshal(ver)
		if err != nil {
			continue
		}
		fields[tool] = string(raw)
	}

	pipe := rdb.Pipeline()
	pipe.HSet(ctx, toolVersionRedisKey, fields)
	pipe.Expire(ctx, toolVersionRedisKey, toolVersionTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		common.SysError(fmt.Sprintf("tool_version_worker: redis write failed: %v", err))
	}
}

// fetchNpmVersion queries the npm registry for the latest version of an npm package.
func fetchNpmVersion(ctx context.Context, client *http.Client, tool, pkg string) *entity.ToolVersion {
	url := "https://registry.npmjs.org/" + pkg + "/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Version string `json:"version"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10)) // 512 KB
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Version == "" {
		return nil
	}

	return &entity.ToolVersion{
		Tool:      tool,
		Version:   result.Version,
		Source:    "npm",
		UpdatedAt: time.Now().UTC(),
	}
}

// fetchGitHubVersion queries the GitHub releases API for the latest release tag.
func fetchGitHubVersion(ctx context.Context, client *http.Client, tool, owner, repo string) *entity.ToolVersion {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		TagName string `json:"tag_name"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(body, &result); err != nil || result.TagName == "" {
		return nil
	}

	version := strings.TrimPrefix(result.TagName, "v")
	return &entity.ToolVersion{
		Tool:      tool,
		Version:   version,
		Source:    "github",
		UpdatedAt: time.Now().UTC(),
	}
}
