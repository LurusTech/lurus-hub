package entity

import "time"

// ToolVersion represents the latest known version of an AI CLI tool.
// Instances are stored in the Redis hash "switch:tool_versions" with a 6-hour TTL.
// No database table is needed — Redis provides sufficient durability for a version cache.
type ToolVersion struct {
	Tool      string    `json:"tool"`
	Version   string    `json:"version"`
	Source    string    `json:"source"` // "npm" or "github"
	UpdatedAt time.Time `json:"updated_at"`
}
