package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	openrouter_sync "github.com/LurusTech/lurus-hub/internal/app/openrouter_sync"
	entity "github.com/LurusTech/lurus-hub/internal/domain/entity"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

// runRequestTimeout caps a single sync run. The fetch + transaction usually
// completes in under a second, but allow generous slack for large channels.
const runRequestTimeout = 60 * time.Second

// listOpenRouterSyncJobs handles GET /api/openrouter-sync/jobs.
func ListOpenRouterSyncJobs(c *gin.Context) {
	jobs, err := repo.ListOpenRouterSyncJobs()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    jobs,
	})
}

// CreateOpenRouterSyncJob handles POST /api/openrouter-sync/jobs.
func CreateOpenRouterSyncJob(c *gin.Context) {
	var body struct {
		Name            string   `json:"name"`
		TargetChannelId int      `json:"target_channel_id"`
		Categories      []string `json:"categories"`
		TopN            int      `json:"top_n"`
		Schedule        string   `json:"schedule"`
		Enabled         *bool    `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := validateJobBody(body.Name, body.TargetChannelId, body.Categories, body.Schedule); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	job := &repo.OpenRouterSyncJob{
		Name:            body.Name,
		TargetChannelId: body.TargetChannelId,
		TopN:            body.TopN,
		Schedule:        body.Schedule,
		Enabled:         body.Enabled == nil || *body.Enabled,
	}
	if err := job.SetCategories(body.Categories); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := repo.CreateOpenRouterSyncJob(job); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": job})
}

// UpdateOpenRouterSyncJob handles PUT /api/openrouter-sync/jobs/:id.
func UpdateOpenRouterSyncJob(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	var body struct {
		Name            *string   `json:"name"`
		TargetChannelId *int      `json:"target_channel_id"`
		Categories      *[]string `json:"categories"`
		TopN            *int      `json:"top_n"`
		Schedule        *string   `json:"schedule"`
		Enabled         *bool     `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		common.ApiError(c, err)
		return
	}
	job, err := repo.GetOpenRouterSyncJob(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if body.Name != nil {
		job.Name = *body.Name
	}
	if body.TargetChannelId != nil {
		job.TargetChannelId = *body.TargetChannelId
	}
	if body.Categories != nil {
		if err := job.SetCategories(*body.Categories); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if body.TopN != nil {
		job.TopN = *body.TopN
	}
	if body.Schedule != nil {
		job.Schedule = *body.Schedule
	}
	if body.Enabled != nil {
		job.Enabled = *body.Enabled
	}
	if err := validateJobBody(job.Name, job.TargetChannelId, job.GetCategories(), job.Schedule); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := repo.UpdateOpenRouterSyncJob(job); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": job})
}

// DeleteOpenRouterSyncJob handles DELETE /api/openrouter-sync/jobs/:id.
func DeleteOpenRouterSyncJob(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	if err := repo.DeleteOpenRouterSyncJob(id); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RunOpenRouterSyncJob handles POST /api/openrouter-sync/jobs/:id/run[?force=true].
// Manually triggers the engine treating only this job as "due".
func RunOpenRouterSyncJob(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	if _, err := repo.GetOpenRouterSyncJob(id); err != nil {
		common.ApiError(c, err)
		return
	}
	force := c.Query("force") == "true"
	ctx, cancel := context.WithTimeout(c.Request.Context(), runRequestTimeout)
	defer cancel()
	result, err := openrouter_sync.NewEngine().Run(ctx, []int{id}, force)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// RunAllOpenRouterSyncJobs handles POST /api/openrouter-sync/run-all[?force=true].
// Treats every enabled job as due — convenience for ops.
func RunAllOpenRouterSyncJobs(c *gin.Context) {
	jobs, err := repo.ListEnabledOpenRouterSyncJobs()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(jobs) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"skipped": true, "skip_reason": "no enabled jobs"}})
		return
	}
	ids := make([]int, 0, len(jobs))
	for _, j := range jobs {
		ids = append(ids, j.Id)
	}
	force := c.Query("force") == "true"
	ctx, cancel := context.WithTimeout(c.Request.Context(), runRequestTimeout)
	defer cancel()
	result, err := openrouter_sync.NewEngine().Run(ctx, ids, force)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// PreviewOpenRouterSyncJob handles GET /api/openrouter-sync/jobs/:id/preview.
// Runs fetch + classify + rank without writing anything to the channel.
func PreviewOpenRouterSyncJob(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	job, err := repo.GetOpenRouterSyncJob(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), runRequestTimeout)
	defer cancel()
	models, err := openrouter_sync.NewEngine().Preview(ctx, job)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// Trim payload to id + name + created (clients don't need the full pricing struct).
	type preview struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Created int64  `json:"created"`
	}
	out := make([]preview, 0, len(models))
	for _, m := range models {
		out = append(out, preview{ID: m.ID, Name: m.Name, Created: m.Created})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

// ListOpenRouterSyncCategories handles GET /api/openrouter-sync/categories.
func ListOpenRouterSyncCategories(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": []gin.H{
		{"key": entity.OpenRouterCategoryLLMReasoning, "label": "推理语言大模型"},
		{"key": entity.OpenRouterCategoryVision, "label": "多模态视觉"},
		{"key": entity.OpenRouterCategoryImageGen, "label": "文生图"},
		{"key": entity.OpenRouterCategoryASR, "label": "语音转文字"},
		{"key": entity.OpenRouterCategoryTTS, "label": "文字转语音"},
	}})
}

// GetOpenRouterSyncLastStatus handles GET /api/openrouter-sync/last-status.
// Returns the latest LastRunAt and LastError across all jobs, plus per-channel
// circuit breaker baseline. Useful for the frontend status card.
func GetOpenRouterSyncLastStatus(c *gin.Context) {
	jobs, err := repo.ListOpenRouterSyncJobs()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	type jobStatus struct {
		Id              int        `json:"id"`
		Name            string     `json:"name"`
		TargetChannelId int        `json:"target_channel_id"`
		LastRunAt       *time.Time `json:"last_run_at"`
		LastError       string     `json:"last_error"`
		Enabled         bool       `json:"enabled"`
	}
	out := make([]jobStatus, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, jobStatus{
			Id: j.Id, Name: j.Name, TargetChannelId: j.TargetChannelId,
			LastRunAt: j.LastRunAt, LastError: j.LastError, Enabled: j.Enabled,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

// validateJobBody checks the user-supplied job parameters.
func validateJobBody(name string, channelId int, cats []string, schedule string) error {
	if strings.TrimSpace(name) == "" {
		return errInvalidJobField("name")
	}
	if channelId <= 0 {
		return errInvalidJobField("target_channel_id")
	}
	if len(cats) == 0 {
		return errInvalidJobField("categories")
	}
	for _, c := range cats {
		if !isValidCategory(c) {
			return errInvalidJobField("categories[" + c + "]")
		}
	}
	if !isValidSchedule(schedule) {
		return errInvalidJobField("schedule")
	}
	return nil
}

func isValidCategory(c string) bool {
	switch c {
	case entity.OpenRouterCategoryLLMReasoning,
		entity.OpenRouterCategoryVision,
		entity.OpenRouterCategoryImageGen,
		entity.OpenRouterCategoryASR,
		entity.OpenRouterCategoryTTS:
		return true
	}
	return false
}

func isValidSchedule(s string) bool {
	switch s {
	case entity.OpenRouterScheduleDaily, entity.OpenRouterScheduleWeekly, entity.OpenRouterScheduleManual:
		return true
	}
	return false
}

type fieldError struct{ field string }

func (e *fieldError) Error() string { return "invalid or missing field: " + e.field }

func errInvalidJobField(field string) error { return &fieldError{field: field} }
