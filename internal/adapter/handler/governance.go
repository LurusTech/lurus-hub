package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/lurus-api/internal/adapter/repo"

	"github.com/gin-gonic/gin"
)

func parseGovernanceHours(c *gin.Context) int64 {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	if hours <= 0 || hours > 720 {
		hours = 24
	}
	return time.Now().Add(-time.Duration(hours) * time.Hour).Unix()
}

// GetGovernanceChannelDistribution returns request count and quota grouped by channel type.
// GET /api/v2/admin/governance/channels?hours=24
func GetGovernanceChannelDistribution(c *gin.Context) {
	stats, err := repo.GetChannelDistribution(parseGovernanceHours(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "query failed"})
		return
	}
	if stats == nil {
		stats = []repo.GovernanceChannelStat{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

// GetGovernanceFingerprintStats returns fingerprint deduplication stats per token.
// GET /api/v2/admin/governance/fingerprints?hours=24&min_count=10
func GetGovernanceFingerprintStats(c *gin.Context) {
	minCount, _ := strconv.Atoi(c.DefaultQuery("min_count", "10"))
	if minCount <= 0 {
		minCount = 10
	}
	stats, err := repo.GetFingerprintDedupStats(parseGovernanceHours(c), minCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "query failed"})
		return
	}
	if stats == nil {
		stats = []repo.GovernanceFingerprintStat{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

// GetGovernanceLatencyStats returns latency statistics per model.
// GET /api/v2/admin/governance/latency?hours=24
func GetGovernanceLatencyStats(c *gin.Context) {
	stats, err := repo.GetLatencyStats(parseGovernanceHours(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "query failed"})
		return
	}
	if stats == nil {
		stats = []repo.GovernanceLatencyStat{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

// GetGovernanceEfficiencyStats returns cost-per-request and cost-per-token metrics.
// GET /api/v2/admin/governance/efficiency?hours=24
func GetGovernanceEfficiencyStats(c *gin.Context) {
	stats, err := repo.GetEfficiencyStats(parseGovernanceHours(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "query failed"})
		return
	}
	if stats == nil {
		stats = []repo.GovernanceEfficiencyStat{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}
