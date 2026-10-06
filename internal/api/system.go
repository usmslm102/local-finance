package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"local-finance/internal/updater"
)

// GetSystemVersion handles GET /api/system/version
func (h *Handler) GetSystemVersion(c *gin.Context) {
	refresh := c.Query("refresh") == "true"
	offline := c.Query("offline") == "true"
	info, err := h.updaterService.CheckForUpdate(c.Request.Context(), refresh, offline)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"current_version":   updater.CurrentVersion,
			"latest_version":    "",
			"update_available":  false,
			"can_auto_update":   false,
			"auto_update_error": err.Error(),
			"release_name":      "",
			"release_notes":     "",
			"release_url":       "",
			"published_at":      "",
			"checked_at":        "",
		})
		return
	}

	c.JSON(http.StatusOK, info)
}

// ApplySystemUpdate handles POST /api/system/update
func (h *Handler) ApplySystemUpdate(c *gin.Context) {
	result, err := h.updaterService.ApplyUpdate(c.Request.Context(), h.db)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "an update is already in progress" {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Trigger asynchronous server restart after sending successful response
	updater.RestartServer()

	c.JSON(http.StatusOK, result)
}
