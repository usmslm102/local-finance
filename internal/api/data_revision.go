package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// DataRevision exposes no ledger contents. Browsers can invalidate cached reports
// after writes by another browser, the desktop host or an MCP client.
func (h *Handler) DataRevision(c *gin.Context) {
	revision, err := h.db.DataRevision()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to check data freshness"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"revision": revision})
}
