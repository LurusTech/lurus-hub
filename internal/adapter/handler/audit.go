package handler

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/lurus-api/internal/adapter/repo"
	"github.com/QuantumNous/lurus-api/internal/domain/entity"

	"github.com/gin-gonic/gin"
)

// GetAuditEvents returns paginated audit events filtered by query params.
// GET /api/v2/admin/audit/events?action=&actor_id=&resource=&start_time=&end_time=&page=&per_page=
func GetAuditEvents(c *gin.Context) {
	action := c.Query("action")
	resource := c.Query("resource")
	actorID, _ := strconv.Atoi(c.Query("actor_id"))
	startTime, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage

	// Admin route — no tenant filter (can see all).
	events, total, err := repo.GetAuditEvents("", action, actorID, resource, startTime, endTime, offset, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to query audit events",
		})
		return
	}
	// Ensure JSON array (never null) for frontend compatibility.
	if events == nil {
		events = make([]*entity.AuditEvent, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"events": events,
			"total":  total,
			"page":   page,
		},
	})
}
