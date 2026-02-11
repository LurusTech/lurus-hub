package app

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/QuantumNous/lurus-api/internal/adapter/repo"
	"github.com/QuantumNous/lurus-api/internal/domain/entity"
)

// ReleaseService handles business logic for releases
type ReleaseService struct {
	repo *repo.ReleaseRepository
	// minioClient *minio.Client // TODO: Add MinIO integration
	minioEndpoint string
	minioBucket   string
}

// NewReleaseService creates a new release service
func NewReleaseService(releaseRepo *repo.ReleaseRepository) *ReleaseService {
	return &ReleaseService{
		repo:          releaseRepo,
		minioEndpoint: "",  // TODO: Load from config
		minioBucket:   "lurus-releases",
	}
}

// ListReleasesResponse defines the response structure
type ListReleasesResponse struct {
	Releases []entity.Release `json:"releases"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// ListReleases retrieves releases with pagination
func (s *ReleaseService) ListReleases(ctx context.Context, params repo.ListReleasesParams) (*ListReleasesResponse, error) {
	releases, total, err := s.repo.ListReleases(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	return &ListReleasesResponse{
		Releases: releases,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

// LatestReleaseResponse defines the response structure for latest release
type LatestReleaseResponse struct {
	Release   *entity.Release `json:"release"`
	HasUpdate bool            `json:"has_update"`
}

// GetLatestRelease retrieves the latest release and checks for updates
func (s *ReleaseService) GetLatestRelease(ctx context.Context, productId string, currentVersion string) (*LatestReleaseResponse, error) {
	release, err := s.repo.GetLatestRelease(ctx, productId)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	if release == nil {
		return &LatestReleaseResponse{
			Release:   nil,
			HasUpdate: false,
		}, nil
	}

	hasUpdate := false
	if currentVersion != "" {
		hasUpdate = compareVersions(currentVersion, release.Version) < 0
	}

	return &LatestReleaseResponse{
		Release:   release,
		HasUpdate: hasUpdate,
	}, nil
}

// GetReleaseByID retrieves a single release
func (s *ReleaseService) GetReleaseByID(ctx context.Context, id int64) (*entity.Release, error) {
	return s.repo.GetReleaseByID(ctx, id)
}

// GenerateDownloadURL generates a presigned URL for downloading an artifact
// If MinIO is not configured, returns a fallback URL
func (s *ReleaseService) GenerateDownloadURL(ctx context.Context, artifact *entity.ReleaseArtifact) (string, error) {
	// TODO: Implement MinIO presigned URL generation
	// For now, return a placeholder or direct MinIO URL
	if s.minioEndpoint == "" {
		// Fallback: return a direct download URL that will be handled by the handler
		return fmt.Sprintf("/api/v1/releases/download/%d", artifact.Id), nil
	}

	// TODO: Generate presigned URL with MinIO SDK
	// Example:
	// presignedURL, err := s.minioClient.PresignedGetObject(ctx, s.minioBucket, artifact.StoragePath, 1*time.Hour, nil)
	// return presignedURL.String(), err

	return "", fmt.Errorf("MinIO not configured")
}

// HandleDownload handles download logic: logging and count increment
func (s *ReleaseService) HandleDownload(ctx context.Context, artifactId int64, ipAddress, userAgent, referer string) error {
	// Create download log (async to not block download)
	go func() {
		logCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log := &entity.DownloadLog{
			ArtifactId:   artifactId,
			IpAddress:    ipAddress,
			UserAgent:    userAgent,
			Referer:      referer,
			CountryCode:  extractCountryFromIP(ipAddress),
			Status:       "initiated",
			DownloadedAt: time.Now(),
		}

		if err := s.repo.LogDownload(logCtx, log); err != nil {
			// Log error but don't block download
			// Using fmt.Printf as logger package doesn't export Error function
			fmt.Printf("Error logging download: %v\n", err)
		}
	}()

	// Increment download count (atomic operation)
	if err := s.repo.IncrementDownloadCount(ctx, artifactId); err != nil {
		return fmt.Errorf("failed to increment download count: %w", err)
	}

	return nil
}

// GetChangelog retrieves the changelog for a release
func (s *ReleaseService) GetChangelog(ctx context.Context, releaseId int64) (string, error) {
	return s.repo.GetChangelogByReleaseID(ctx, releaseId)
}

// compareVersions compares two semantic versions
// Returns -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	// Simple version comparison (assumes semantic versioning)
	// TODO: Implement proper semantic version parsing
	if v1 == v2 {
		return 0
	}
	if v1 < v2 {
		return -1
	}
	return 1
}

// extractCountryFromIP extracts country code from IP (placeholder)
func extractCountryFromIP(ipAddress string) string {
	// TODO: Implement GeoIP lookup
	// For now, return empty or use a third-party service
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return ""
	}

	// Placeholder: return empty for now
	return ""
}
