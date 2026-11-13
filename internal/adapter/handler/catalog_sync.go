package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	catalog_sync "github.com/LurusTech/lurus-hub/internal/app/catalog_sync"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

const catalogSyncRunTimeout = 120 * time.Second

// RunCatalogSync handles POST /api/catalog-sync/run (force one cycle).
func RunCatalogSync(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), catalogSyncRunTimeout)
	defer cancel()
	engine := catalog_sync.NewEngine()
	result, err := engine.Run(ctx)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// ListCatalogSyncRuns handles GET /api/catalog-sync/runs.
func ListCatalogSyncRuns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	rows, err := catalog_sync.ListRecentRuns(limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

// ListCatalogStale handles GET /api/catalog-sync/stale.
func ListCatalogStale(c *gin.Context) {
	channelID, _ := strconv.Atoi(c.Query("channel_id"))
	rows, err := catalog_sync.ListOpenStale(channelID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

// ConfirmCatalogStale handles POST /api/catalog-sync/stale/confirm.
func ConfirmCatalogStale(c *gin.Context) {
	var req catalog_sync.ConfirmStaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := catalog_sync.ConfirmStaleRemovals(req); err != nil {
		writeCatalogSyncErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListCatalogPriceProposals handles GET /api/catalog-sync/price-proposals.
func ListCatalogPriceProposals(c *gin.Context) {
	rows, err := catalog_sync.ListOpenPriceProposals()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

// ApplyCatalogPriceProposals handles POST /api/catalog-sync/price-proposals/apply.
func ApplyCatalogPriceProposals(c *gin.Context) {
	var req catalog_sync.ApplyPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := catalog_sync.ApplyPriceProposals(req); err != nil {
		writeCatalogSyncErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DismissCatalogPriceProposals handles POST /api/catalog-sync/price-proposals/dismiss.
func DismissCatalogPriceProposals(c *gin.Context) {
	var body struct {
		SourceVersion string  `json:"source_version"`
		IDs           []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := catalog_sync.DismissPriceProposals(body.IDs, body.SourceVersion); err != nil {
		writeCatalogSyncErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func writeCatalogSyncErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, catalog_sync.ErrSourceVersionMismatch):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, catalog_sync.ErrEmptySelection):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
	default:
		common.ApiError(c, err)
	}
}
