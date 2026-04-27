package repo

import (
	"context"
	"fmt"

	"github.com/LurusTech/lurus-hub/internal/domain/entity"
	"gorm.io/gorm"
)

// ReleaseRepository handles database operations for releases
type ReleaseRepository struct {
	db *gorm.DB
}

// NewReleaseRepository creates a new release repository
func NewReleaseRepository(db *gorm.DB) *ReleaseRepository {
	return &ReleaseRepository{db: db}
}

// ListReleasesParams defines query parameters for listing releases
type ListReleasesParams struct {
	ProductId         string
	ReleaseType       string
	IncludePrerelease bool
	Page              int
	PageSize          int
}

// ListReleases retrieves releases with pagination and filtering
func (r *ReleaseRepository) ListReleases(ctx context.Context, params ListReleasesParams) ([]entity.Release, int64, error) {
	var releases []entity.Release
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Release{}).
		Where("is_published = ?", true)

	// Apply filters
	if params.ProductId != "" {
		query = query.Where("product_id = ?", params.ProductId)
	}

	if params.ReleaseType != "" {
		query = query.Where("release_type = ?", params.ReleaseType)
	}

	if !params.IncludePrerelease {
		query = query.Where("is_prerelease = ?", false)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count releases: %w", err)
	}

	// Apply pagination
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	offset := (params.Page - 1) * params.PageSize

	// Fetch releases with artifacts
	err := query.
		Preload("Artifacts").
		Order("published_at DESC").
		Limit(params.PageSize).
		Offset(offset).
		Find(&releases).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch releases: %w", err)
	}

	return releases, total, nil
}

// GetLatestRelease retrieves the latest published release for a product
func (r *ReleaseRepository) GetLatestRelease(ctx context.Context, productId string) (*entity.Release, error) {
	var release entity.Release

	err := r.db.WithContext(ctx).
		Preload("Artifacts").
		Where("product_id = ? AND is_published = ? AND is_prerelease = ?", productId, true, false).
		Order("published_at DESC").
		First(&release).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	return &release, nil
}

// GetReleaseByID retrieves a release by ID with artifacts
func (r *ReleaseRepository) GetReleaseByID(ctx context.Context, id int64) (*entity.Release, error) {
	var release entity.Release

	err := r.db.WithContext(ctx).
		Preload("Artifacts").
		Where("id = ? AND is_published = ?", id, true).
		First(&release).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}

	return &release, nil
}

// GetArtifactByID retrieves a release artifact by ID
func (r *ReleaseRepository) GetArtifactByID(ctx context.Context, artifactId int64) (*entity.ReleaseArtifact, error) {
	var artifact entity.ReleaseArtifact

	err := r.db.WithContext(ctx).
		Where("id = ?", artifactId).
		First(&artifact).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch artifact: %w", err)
	}

	return &artifact, nil
}

// LogDownload creates a download log entry
func (r *ReleaseRepository) LogDownload(ctx context.Context, log *entity.DownloadLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("failed to create download log: %w", err)
	}
	return nil
}

// IncrementDownloadCount atomically increments the download count for an artifact
func (r *ReleaseRepository) IncrementDownloadCount(ctx context.Context, artifactId int64) error {
	err := r.db.WithContext(ctx).
		Model(&entity.ReleaseArtifact{}).
		Where("id = ?", artifactId).
		UpdateColumn("download_count", gorm.Expr("download_count + ?", 1)).Error

	if err != nil {
		return fmt.Errorf("failed to increment download count: %w", err)
	}

	return nil
}

// GetChangelogByReleaseID retrieves the changelog for a release
func (r *ReleaseRepository) GetChangelogByReleaseID(ctx context.Context, releaseId int64) (string, error) {
	var release entity.Release

	err := r.db.WithContext(ctx).
		Select("changelog_md").
		Where("id = ? AND is_published = ?", releaseId, true).
		First(&release).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", fmt.Errorf("failed to fetch changelog: %w", err)
	}

	return release.ChangelogMd, nil
}

// CreateRelease creates a new release (admin use)
func (r *ReleaseRepository) CreateRelease(ctx context.Context, release *entity.Release) error {
	if err := r.db.WithContext(ctx).Create(release).Error; err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}
	return nil
}

// CreateArtifact creates a new artifact (admin use)
func (r *ReleaseRepository) CreateArtifact(ctx context.Context, artifact *entity.ReleaseArtifact) error {
	if err := r.db.WithContext(ctx).Create(artifact).Error; err != nil {
		return fmt.Errorf("failed to create artifact: %w", err)
	}
	return nil
}
