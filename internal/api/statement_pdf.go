package api

import (
	"database/sql"
	"errors"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetStatementPDF(c *gin.Context) {
	filename, data, err := h.db.GetStatementPDF(c.Param("id"))
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "The original PDF is not available for this import. Import the PDF again to save it for preview."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to load the statement PDF. Please try again."})
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": filepath.Base(filename)}))
	c.Data(http.StatusOK, "application/pdf", data)
}
