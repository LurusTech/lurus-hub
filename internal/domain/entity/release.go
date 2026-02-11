package entity

import (
	"time"
)

// Release represents a product release version
type Release struct {
	Id            int64      `json:"id" gorm:"primaryKey"`
	ProductId     string     `json:"product_id" gorm:"type:varchar(50);not null;index"`
	Version       string     `json:"version" gorm:"type:varchar(50);not null"`
	Title         string     `json:"title" gorm:"type:varchar(255);not null"`
	Description   string     `json:"description" gorm:"type:text"`
	ChangelogMd   string     `json:"changelog_md" gorm:"type:text;column:changelog_md"`
	ReleaseType   string     `json:"release_type" gorm:"type:varchar(20);default:'stable'"`
	IsDraft       bool       `json:"is_draft" gorm:"default:true"`
	IsPrerelease  bool       `json:"is_prerelease" gorm:"default:false"`
	IsPublished   bool       `json:"is_published" gorm:"default:false"`
	CreatedAt     time.Time  `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`
	PublishedAt   *time.Time `json:"published_at"`
	Artifacts     []ReleaseArtifact `json:"artifacts" gorm:"foreignKey:ReleaseId"`
}

func (Release) TableName() string {
	return "releases"
}

// ReleaseArtifact represents a downloadable file for a specific platform
type ReleaseArtifact struct {
	Id             int64     `json:"id" gorm:"primaryKey"`
	ReleaseId      int64     `json:"release_id" gorm:"not null;index"`
	Platform       string    `json:"platform" gorm:"type:varchar(20);not null;index"`
	Arch           string    `json:"arch" gorm:"type:varchar(20);not null"`
	Filename       string    `json:"filename" gorm:"type:varchar(255);not null"`
	FileSize       int64     `json:"file_size" gorm:"not null"`
	MimeType       string    `json:"mime_type" gorm:"type:varchar(100);column:mime_type"`
	StoragePath    string    `json:"storage_path" gorm:"type:varchar(500);not null;column:storage_path"`
	ChecksumSha256 string    `json:"checksum_sha256" gorm:"type:varchar(64);not null;column:checksum_sha256"`
	DownloadCount  int64     `json:"download_count" gorm:"default:0;column:download_count"`
	CreatedAt      time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`
}

func (ReleaseArtifact) TableName() string {
	return "release_artifacts"
}

// DownloadLog tracks download events for analytics
type DownloadLog struct {
	Id           int64     `json:"id" gorm:"primaryKey"`
	ArtifactId   int64     `json:"artifact_id" gorm:"not null;index;column:artifact_id"`
	IpAddress    string    `json:"ip_address" gorm:"type:inet;column:ip_address"`
	UserAgent    string    `json:"user_agent" gorm:"type:text;column:user_agent"`
	Referer      string    `json:"referer" gorm:"type:text"`
	CountryCode  string    `json:"country_code" gorm:"type:varchar(2);column:country_code"`
	Status       string    `json:"status" gorm:"type:varchar(20);default:'initiated'"`
	DownloadedAt time.Time `json:"downloaded_at" gorm:"default:CURRENT_TIMESTAMP;column:downloaded_at"`
}

func (DownloadLog) TableName() string {
	return "download_logs"
}

// ValidReleaseTypes defines allowed release types
var ValidReleaseTypes = []string{"stable", "beta", "alpha"}

// ValidPlatforms defines supported platforms
var ValidPlatforms = []string{"windows", "darwin", "linux", "android", "ios"}

// ValidArchitectures defines supported architectures
var ValidArchitectures = []string{"x64", "arm64", "amd64", "universal"}

// ValidDownloadStatuses defines download log statuses
var ValidDownloadStatuses = []string{"initiated", "completed", "failed"}
