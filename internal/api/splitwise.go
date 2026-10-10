package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"local-finance/internal/db"
	"local-finance/internal/models"
	"local-finance/internal/parser"
)

func (h *Handler) ListSplitwise(c *gin.Context) {
	entries, err := h.db.ListSplitwise()
	if err != nil {
		c.JSON(500, gin.H{"error": "could not load Splitwise entries"})
		return
	}
	c.JSON(200, entries)
}

func (h *Handler) PreviewSplitwise(c *gin.Context) { h.splitwiseUpload(c, true) }
func (h *Handler) ImportSplitwise(c *gin.Context)  { h.splitwiseUpload(c, false) }
func (h *Handler) splitwiseUpload(c *gin.Context, preview bool) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "choose a Splitwise CSV export"})
		return
	}
	defer file.Close()
	entries, err := parser.ParseSplitwise(file, c.PostForm("group"), c.PostForm("person"))
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if preview {
		c.JSON(200, entries)
		return
	}
	inserted, err := h.db.ImportSplitwise(entries)
	if err != nil {
		c.JSON(500, gin.H{"error": "could not import Splitwise entries"})
		return
	}
	c.JSON(200, gin.H{"inserted": inserted, "duplicates": len(entries) - inserted})
}

func (h *Handler) ConfirmSplitwise(c *gin.Context) {
	var req models.SplitwiseConfirmation
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid confirmation"})
		return
	}
	if err := h.db.ConfirmSplitwise(c.Param("id"), req); err != nil {
		splitwiseError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *Handler) ResetSplitwise(c *gin.Context) {
	if err := h.db.ResetSplitwise(c.Param("id")); err != nil {
		splitwiseError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *Handler) SplitwiseCandidates(c *gin.Context) {
	share, err := strconv.ParseInt(c.Query("share_cents"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid share"})
		return
	}
	items, err := h.db.SplitwiseCandidates(c.Param("id"), share)
	if err != nil {
		splitwiseError(c, err)
		return
	}
	c.JSON(200, items)
}
func splitwiseError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, db.ErrSplitwiseNotFound) {
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
