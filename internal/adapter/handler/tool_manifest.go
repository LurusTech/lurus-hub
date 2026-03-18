package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetToolDownloadManifest serves the dynamic tool download manifest.
// Binary tool entries are discovered from MinIO storage with presigned URLs;
// npm tool entries are static. Results are cached for 5 minutes.
//
// GET /api/v2/tools/download-manifest
func GetToolDownloadManifest(c *gin.Context) {
	if releaseService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "release service not initialized",
		})
		return
	}

	manifest, err := releaseService.BuildToolManifest(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to build tool manifest",
		})
		return
	}

	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, manifest)
}
