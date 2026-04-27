package repo

import (
	"errors"

	entity "github.com/LurusTech/lurus-hub/internal/domain/entity"
)

// OpenRouterSyncJob is aliased from the canonical entity definition.
type OpenRouterSyncJob = entity.OpenRouterSyncJob

// CreateOpenRouterSyncJob inserts a new sync job.
func CreateOpenRouterSyncJob(job *OpenRouterSyncJob) error {
	if job.Name == "" {
		return errors.New("job name is required")
	}
	if job.TargetChannelId <= 0 {
		return errors.New("target channel id is required")
	}
	return DB.Create(job).Error
}

// UpdateOpenRouterSyncJob persists arbitrary changes to an existing job.
func UpdateOpenRouterSyncJob(job *OpenRouterSyncJob) error {
	if job.Id <= 0 {
		return errors.New("invalid job id")
	}
	return DB.Save(job).Error
}

// DeleteOpenRouterSyncJob removes a job by id.
func DeleteOpenRouterSyncJob(id int) error {
	return DB.Delete(&OpenRouterSyncJob{}, id).Error
}

// GetOpenRouterSyncJob fetches a single job by id.
func GetOpenRouterSyncJob(id int) (*OpenRouterSyncJob, error) {
	var job OpenRouterSyncJob
	err := DB.First(&job, id).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// ListOpenRouterSyncJobs returns all jobs ordered by id desc.
func ListOpenRouterSyncJobs() ([]*OpenRouterSyncJob, error) {
	var jobs []*OpenRouterSyncJob
	err := DB.Order("id desc").Find(&jobs).Error
	return jobs, err
}

// ListEnabledOpenRouterSyncJobs returns all enabled jobs (used by the scheduler).
func ListEnabledOpenRouterSyncJobs() ([]*OpenRouterSyncJob, error) {
	var jobs []*OpenRouterSyncJob
	err := DB.Where("enabled = ?", true).Order("id asc").Find(&jobs).Error
	return jobs, err
}
