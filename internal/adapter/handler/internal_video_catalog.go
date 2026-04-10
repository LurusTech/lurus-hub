package handler

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// VideoModelEntry is the API response format for video model catalog.
// Matches the Creator's VideoModelInfo struct for direct consumption.
type VideoModelEntry struct {
	ID            string   `json:"id"`
	DisplayName   string   `json:"display_name"`
	CostPerClip   float64  `json:"cost_per_clip"`
	QualityRating int      `json:"quality_rating"`
	DailyLimit    int      `json:"daily_limit"`
	SpeedRating   int      `json:"speed_rating"`
	Notes         string   `json:"notes"`
	SupportsI2V   bool     `json:"supports_i2v"`
	Provider      string   `json:"provider"`
	FalEndpoint   string   `json:"fal_endpoint"`
	MaxDuration   int      `json:"max_duration"`
	Category      string   `json:"category"`
	StyleTags     []string `json:"style_tags"`
}

// VideoModelStatus is the per-model health info returned by the status endpoint.
type VideoModelStatus struct {
	ModelID     string `json:"model_id"`
	Available   bool   `json:"available"`
	ErrorRate   float64 `json:"error_rate"`    // 0.0-1.0 error ratio in last hour
	AvgLatency  int64  `json:"avg_latency_ms"` // Average generation time in ms
	LastError   string `json:"last_error,omitempty"`
	LastErrorAt string `json:"last_error_at,omitempty"`
}

// videoModelHealth tracks per-model health data collected from relay usage.
type videoModelHealth struct {
	mu      sync.RWMutex
	models  map[string]*modelHealthEntry
}

type modelHealthEntry struct {
	successes   int
	failures    int
	totalMs     int64
	lastError   string
	lastErrorAt time.Time
}

var globalVideoHealth = &videoModelHealth{
	models: make(map[string]*modelHealthEntry),
}

// RecordVideoModelSuccess records a successful video generation via the relay.
func RecordVideoModelSuccess(modelID string, durationMs int64) {
	globalVideoHealth.mu.Lock()
	defer globalVideoHealth.mu.Unlock()
	e := globalVideoHealth.getOrCreate(modelID)
	e.successes++
	e.totalMs += durationMs
}

// RecordVideoModelFailure records a failed video generation via the relay.
func RecordVideoModelFailure(modelID string, errMsg string) {
	globalVideoHealth.mu.Lock()
	defer globalVideoHealth.mu.Unlock()
	e := globalVideoHealth.getOrCreate(modelID)
	e.failures++
	e.lastError = errMsg
	e.lastErrorAt = time.Now()
}

func (h *videoModelHealth) getOrCreate(modelID string) *modelHealthEntry {
	e, ok := h.models[modelID]
	if !ok {
		e = &modelHealthEntry{}
		h.models[modelID] = e
	}
	return e
}

// serverVideoModels is the canonical registry served by the Hub.
// Clients (Creator, etc.) pull this on startup instead of hardcoding.
var serverVideoModels = []VideoModelEntry{
	// --- Gateway models (via newapi relay) ---
	{ID: "hailuo", DisplayName: "Hailuo (MiniMax)", CostPerClip: 0.05, QualityRating: 3, SpeedRating: 4, DailyLimit: 20, MaxDuration: 6, Provider: "gateway", Category: "budget", StyleTags: []string{"fast", "draft"}, Notes: "Fast budget model by MiniMax"},
	{ID: "hailuo-01", DisplayName: "Hailuo 01", CostPerClip: 0.05, QualityRating: 3, SpeedRating: 4, DailyLimit: 20, MaxDuration: 6, Provider: "gateway", Category: "budget", StyleTags: []string{"fast", "draft"}, Notes: "Fast budget model variant"},
	{ID: "kling-v1", DisplayName: "Kling v1", CostPerClip: 0.10, QualityRating: 4, SpeedRating: 3, DailyLimit: 10, MaxDuration: 10, Provider: "gateway", Category: "balanced", SupportsI2V: true, StyleTags: []string{"cinematic", "realistic"}, Notes: "Kuaishou balanced video model"},
	{ID: "kling-v1-5", DisplayName: "Kling v1.5", CostPerClip: 0.15, QualityRating: 4, SpeedRating: 3, DailyLimit: 10, MaxDuration: 10, Provider: "gateway", Category: "balanced", SupportsI2V: true, StyleTags: []string{"cinematic", "realistic", "motion"}, Notes: "Improved motion quality"},
	{ID: "kling-v2", DisplayName: "Kling v2", CostPerClip: 0.20, QualityRating: 5, SpeedRating: 2, DailyLimit: 5, MaxDuration: 10, Provider: "gateway", Category: "premium", SupportsI2V: true, StyleTags: []string{"cinematic", "realistic", "motion", "character"}, Notes: "Premium Kuaishou model"},
	{ID: "cogvideox", DisplayName: "CogVideoX", CostPerClip: 0.08, QualityRating: 3, SpeedRating: 3, DailyLimit: 0, MaxDuration: 6, Provider: "gateway", Category: "budget", StyleTags: []string{"creative", "abstract"}, Notes: "Open-source by Zhipu AI"},
	{ID: "runway-gen3", DisplayName: "Runway Gen-3 Alpha", CostPerClip: 0.50, QualityRating: 5, SpeedRating: 2, DailyLimit: 5, MaxDuration: 10, Provider: "gateway", Category: "premium", SupportsI2V: true, StyleTags: []string{"cinematic", "film", "realistic", "commercial"}, Notes: "Hollywood-grade generation"},
	{ID: "pika-v2", DisplayName: "Pika v2", CostPerClip: 0.15, QualityRating: 4, SpeedRating: 3, DailyLimit: 10, MaxDuration: 4, Provider: "gateway", Category: "balanced", StyleTags: []string{"creative", "stylized", "motion"}, Notes: "Stylized creative videos"},
	{ID: "luma-ray2", DisplayName: "Luma Ray2", CostPerClip: 0.25, QualityRating: 5, SpeedRating: 2, DailyLimit: 5, MaxDuration: 10, Provider: "gateway", Category: "premium", SupportsI2V: true, StyleTags: []string{"cinematic", "realistic", "character"}, Notes: "Luma AI flagship"},
	{ID: "vidu-q1", DisplayName: "Vidu Q1", CostPerClip: 0.15, QualityRating: 4, SpeedRating: 3, DailyLimit: 10, MaxDuration: 8, Provider: "gateway", Category: "balanced", SupportsI2V: true, StyleTags: []string{"realistic", "character", "chinese"}, Notes: "Good for Chinese content"},
	{ID: "wan-2.1", DisplayName: "Wan 2.1", CostPerClip: 0.10, QualityRating: 4, SpeedRating: 3, DailyLimit: 0, MaxDuration: 5, Provider: "gateway", Category: "balanced", StyleTags: []string{"creative", "artistic", "anime"}, Notes: "Alibaba open-source model"},
	{ID: "seedance-1.5", DisplayName: "Seedance 1.5", CostPerClip: 0.12, QualityRating: 4, SpeedRating: 3, DailyLimit: 10, MaxDuration: 10, Provider: "gateway", Category: "balanced", SupportsI2V: true, StyleTags: []string{"motion", "character", "narrative"}, Notes: "ByteDance dance/motion model"},
	{ID: "jimeng", DisplayName: "Jimeng (ByteDance)", CostPerClip: 0.05, QualityRating: 3, SpeedRating: 4, DailyLimit: 0, MaxDuration: 6, Provider: "gateway", Category: "fast", StyleTags: []string{"fast", "chinese", "draft"}, Notes: "ByteDance fast generation"},
	{ID: "doubao", DisplayName: "Doubao Video", CostPerClip: 0.08, QualityRating: 3, SpeedRating: 4, DailyLimit: 0, MaxDuration: 6, Provider: "gateway", Category: "fast", StyleTags: []string{"fast", "chinese"}, Notes: "ByteDance Doubao"},
	{ID: "veo-2.0-generate-001", DisplayName: "Google Veo 2", CostPerClip: 0.25, QualityRating: 5, SpeedRating: 2, DailyLimit: 5, MaxDuration: 8, Provider: "gateway", Category: "premium", StyleTags: []string{"cinematic", "realistic", "film"}, Notes: "Google DeepMind Veo 2"},
	{ID: "veo-3.0-generate-001", DisplayName: "Google Veo 3", CostPerClip: 0.35, QualityRating: 5, SpeedRating: 2, DailyLimit: 3, MaxDuration: 8, Provider: "gateway", Category: "premium", StyleTags: []string{"cinematic", "realistic", "film"}, Notes: "Latest Google Veo with audio"},
	// --- fal.ai models (direct queue API) ---
	{ID: "fal/kling-v2", DisplayName: "Kling v2 (fal)", CostPerClip: 0.22, QualityRating: 5, SpeedRating: 2, DailyLimit: 0, MaxDuration: 10, Provider: "fal", FalEndpoint: "fal-ai/kling-video/v2/master", Category: "premium", SupportsI2V: true, StyleTags: []string{"cinematic", "realistic", "motion", "character"}, Notes: "Kling v2 via fal.ai CDN"},
	{ID: "fal/hailuo", DisplayName: "Hailuo (fal)", CostPerClip: 0.06, QualityRating: 3, SpeedRating: 4, DailyLimit: 0, MaxDuration: 6, Provider: "fal", FalEndpoint: "fal-ai/minimax-video", Category: "budget", StyleTags: []string{"fast", "draft"}, Notes: "MiniMax via fal.ai"},
	{ID: "fal/luma-ray2", DisplayName: "Luma Ray2 (fal)", CostPerClip: 0.28, QualityRating: 5, SpeedRating: 2, DailyLimit: 0, MaxDuration: 10, Provider: "fal", FalEndpoint: "fal-ai/luma-dream-machine", Category: "premium", SupportsI2V: true, StyleTags: []string{"cinematic", "realistic", "character"}, Notes: "Luma AI via fal.ai"},
	{ID: "fal/runway-gen3", DisplayName: "Runway Gen-3 (fal)", CostPerClip: 0.55, QualityRating: 5, SpeedRating: 2, DailyLimit: 0, MaxDuration: 10, Provider: "fal", FalEndpoint: "fal-ai/runway-gen3/turbo/image-to-video", Category: "premium", SupportsI2V: true, StyleTags: []string{"cinematic", "film", "realistic", "commercial"}, Notes: "Runway via fal.ai"},
	{ID: "fal/cogvideox", DisplayName: "CogVideoX (fal)", CostPerClip: 0.10, QualityRating: 3, SpeedRating: 3, DailyLimit: 0, MaxDuration: 6, Provider: "fal", FalEndpoint: "fal-ai/cogvideox-5b", Category: "budget", StyleTags: []string{"creative", "abstract"}, Notes: "CogVideoX via fal.ai"},
	{ID: "fal/wan-2.1", DisplayName: "Wan 2.1 (fal)", CostPerClip: 0.12, QualityRating: 4, SpeedRating: 3, DailyLimit: 0, MaxDuration: 5, Provider: "fal", FalEndpoint: "fal-ai/wan", Category: "balanced", StyleTags: []string{"creative", "artistic", "anime"}, Notes: "Wan via fal.ai"},
	{ID: "fal/seedance-1.5", DisplayName: "Seedance 1.5 (fal)", CostPerClip: 0.14, QualityRating: 4, SpeedRating: 3, DailyLimit: 0, MaxDuration: 10, Provider: "fal", FalEndpoint: "fal-ai/seedance", Category: "balanced", SupportsI2V: true, StyleTags: []string{"motion", "character", "narrative"}, Notes: "Seedance via fal.ai"},
	{ID: "fal/vidu-q1", DisplayName: "Vidu Q1 (fal)", CostPerClip: 0.18, QualityRating: 4, SpeedRating: 3, DailyLimit: 0, MaxDuration: 8, Provider: "fal", FalEndpoint: "fal-ai/vidu/q1", Category: "balanced", SupportsI2V: true, StyleTags: []string{"realistic", "character", "chinese"}, Notes: "Vidu via fal.ai"},
}

// InternalGetVideoCatalog returns the full video model catalog with pricing and metadata.
// GET /internal/models/video-catalog
func InternalGetVideoCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    serverVideoModels,
	})
}

// InternalGetVideoStatus returns per-model health status from in-process metrics.
// GET /internal/models/video-status
func InternalGetVideoStatus(c *gin.Context) {
	globalVideoHealth.mu.RLock()
	defer globalVideoHealth.mu.RUnlock()

	statuses := make([]VideoModelStatus, 0, len(serverVideoModels))
	for _, m := range serverVideoModels {
		st := VideoModelStatus{
			ModelID:   m.ID,
			Available: true,
		}
		if e, ok := globalVideoHealth.models[m.ID]; ok {
			total := e.successes + e.failures
			if total > 0 {
				st.ErrorRate = float64(e.failures) / float64(total)
			}
			if e.successes > 0 {
				st.AvgLatency = e.totalMs / int64(e.successes)
			}
			if e.lastError != "" {
				st.LastError = e.lastError
				st.LastErrorAt = e.lastErrorAt.Format(time.RFC3339)
			}
			// Mark unavailable if error rate exceeds 50% with at least 3 samples.
			if total >= 3 && st.ErrorRate > 0.5 {
				st.Available = false
			}
		}
		statuses = append(statuses, st)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    statuses,
	})
}

// isVideoModel checks if a model name is a video model based on keyword matching.
func isVideoModel(name string) bool {
	lower := strings.ToLower(name)
	for _, kw := range []string{"veo", "kling", "sora", "cogvideo", "video", "jimeng", "runway",
		"hailuo", "luma", "pika", "vidu", "wan", "seedance", "doubao"} {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
